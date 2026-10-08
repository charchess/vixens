#!/usr/bin/env python3
"""Prevent accidental Hermes test/stable pointer advancement through the required PR gate.

This is a review/evidence prerequisite, NOT a substitute for physical validation.
Only dev may be advanced by an OCI publishing bot. Existing legacy pointers are
left to the separately authorized per-agent brownfield migration.
"""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

PROFILES = Path("apps/60-services/txo-fabric/operator/config/profiles")
NAME = re.compile(r"^hermes-[a-z0-9-]+$")
DIGEST = re.compile(r"^[a-z0-9][a-z0-9._:/-]*@sha256:[0-9a-f]{64}$")
EVIDENCE_URL = re.compile(r"^https://github\.com/charchess/vixens/issues/\d+#issuecomment-\d+$")
SNAPSHOT = re.compile(r"^dev-v\d{4}\.\d{2}\.\d+$")


def git(*args: str) -> str:
    result = subprocess.run(
        ["git", *args], check=True, capture_output=True, text=True
    )
    return result.stdout.strip()


def field(body: str, name: str) -> str:
    values = re.findall(rf"^{re.escape(name)}:\s*(\S+)\s*$", body, re.MULTILINE)
    if len(values) != 1:
        raise ValueError(f"expected exactly one '{name}: <value>' in PR body")
    return values[0]


def validate_release(ref: str) -> None:
    if not NAME.fullmatch(ref) or "/" in ref or ".." in ref:
        raise ValueError(f"invalid immutable Hermes release name: {ref}")
    path = PROFILES / f"{ref}.yaml"
    if not path.is_file():
        raise ValueError(f"release not declared: {path}")
    content = path.read_text(encoding="utf-8")
    if "kind: HermesRuntimeRelease\n" not in content or f"  name: {ref}\n" not in content:
        raise ValueError(f"unexpected release manifest: {path}")
    for key in ("image", "hindsightPluginImage"):
        values = re.findall(rf"^  {key}:\s*(\S+)\s*$", content, re.MULTILINE)
        if len(values) != 1 or not DIGEST.fullmatch(values[0]):
            raise ValueError(f"{path}: {key} must use a real immutable OCI digest")


def read_profile(name: str) -> str:
    path = PROFILES / f"{name}.yaml"
    if not path.is_file():
        raise ValueError(f"missing channel profile: {path}")
    content = path.read_text(encoding="utf-8")
    if "kind: AgentRuntimeProfile\n" not in content or f"  name: {name}\n" not in content:
        raise ValueError(f"invalid profile name in {path}")
    refs = re.findall(r"^  releaseRef:\s*(\S+)\s*$", content, re.MULTILINE)
    if len(refs) != 1 or re.search(r"^  image:", content, re.MULTILINE):
        raise ValueError(f"{path}: expected releaseRef only (no inline image)")
    validate_release(refs[0])
    return refs[0]


def previously_selected(base: str, channel: str, ref: str) -> bool:
    path = str(PROFILES / f"{channel}.yaml")
    # Git history, not the current channel head: dev may have moved forward since
    # the qualified candidate, while release objects remain immutable.
    commits = git("log", "--format=%H", "-S", f"releaseRef: {ref}", base, "--", path)
    return bool(commits)


def gate(base: str, body: str, author_type: str) -> list[str]:
    changed = set(git("diff", "--name-only", f"{base}...HEAD").splitlines())
    gated = [
        name for name in ("hermes-test", "hermes-stable")
        if str(PROFILES / f"{name}.yaml") in changed
    ]
    if not gated:
        return []
    if author_type.lower() == "bot":
        raise ValueError("bot-authored PR may advance only hermes-dev; test/stable are manual")

    physical = field(body, "Hermes-Physical-Evidence")
    recovery = field(body, "Hermes-Recovery-Evidence")
    if not EVIDENCE_URL.fullmatch(physical) or not EVIDENCE_URL.fullmatch(recovery):
        raise ValueError("physical and recovery evidence must link actual GitHub issue comments")

    for channel in gated:
        ref = read_profile(channel)
        prerequisite = "hermes-dev" if channel == "hermes-test" else "hermes-test"
        if not previously_selected(base, prerequisite, ref):
            raise ValueError(f"{channel} release {ref} was never selected by {prerequisite} on main")
        if channel == "hermes-stable":
            approval = field(body, "Hermes-Owner-Approval")
            snapshot = field(body, "Hermes-GitOps-Snapshot")
            if not EVIDENCE_URL.fullmatch(approval) or not SNAPSHOT.fullmatch(snapshot):
                raise ValueError("stable requires owner approval comment and exact dev-v* snapshot")
    return gated


if __name__ == "__main__":
    if len(sys.argv) != 4:
        raise SystemExit("usage: guard-hermes-release-channels.py BASE_REF PR_BODY AUTHOR_TYPE")
    try:
        channels = gate(sys.argv[1], sys.argv[2], sys.argv[3])
        print("Hermes release gate OK: " + (", ".join(channels) if channels else "no test/stable pointer changes"))
    except (ValueError, subprocess.CalledProcessError) as exc:
        raise SystemExit(f"Hermes release gate FAILED: {exc}") from exc
