#!/usr/bin/env python3
"""Gate pinned Hermes edge/canary/stable channel changes on GitOps PRs.

This checks immutable configuration/history and evidence *references*, not real
physical acceptance or approval. WORKFLOW.md and human review remain binding.
"""
from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

PROFILES = Path("apps/60-services/txo-fabric/operator/config/profiles")
CHANNELS = ("hermes-edge", "hermes-canary", "hermes-stable")
LEGACY = ("hermes-default", "hermes-dev", "hermes-upgrade-canary")
NAME = re.compile(r"^hermes-[a-z0-9-]+$")
DIGEST = re.compile(r"^[a-z0-9][a-z0-9._:/-]*@sha256:[0-9a-f]{64}$")
EVIDENCE_URL = re.compile(r"^https://github\.com/charchess/vixens/issues/\d+#issuecomment-\d+$")
SNAPSHOT = re.compile(r"^dev-v\d{4}\.\d{2}\.\d+$")


def git(*args: str) -> str:
    return subprocess.run(["git", *args], check=True, capture_output=True, text=True).stdout.strip()


def field(body: str, name: str) -> str:
    values = re.findall(rf"^\s*(?:-\s+)?{re.escape(name)}:\s*(\S+)\s*$", body, re.MULTILINE)
    if len(values) != 1:
        raise ValueError(f"expected exactly one '{name}: <value>' in PR body")
    return values[0]


def validate_release(ref: str) -> None:
    if not NAME.fullmatch(ref):
        raise ValueError(f"invalid immutable Hermes release name: {ref}")
    path = PROFILES / f"{ref}.yaml"
    if not path.is_file():
        raise ValueError(f"release not declared: {path}")
    content = path.read_text(encoding="utf-8")
    if "kind: HermesRuntimeRelease\n" not in content or f"  name: {ref}\n" not in content:
        raise ValueError(f"unexpected release manifest: {path}")
    for key, prefix in (
        ("image", "nousresearch/hermes-agent@sha256:"),
        ("hindsightPluginImage", "ghcr.io/charchess/txo-hermes-hindsight-plugin@sha256:"),
    ):
        values = re.findall(rf"^  {key}:\s*(\S+)\s*$", content, re.MULTILINE)
        if len(values) != 1 or not DIGEST.fullmatch(values[0]) or not values[0].startswith(prefix):
            raise ValueError(f"{path}: {key} must use the official source and immutable OCI digest")


def read_profile(channel: str) -> str:
    path = PROFILES / f"{channel}.yaml"
    if not path.is_file():
        raise ValueError(f"missing channel profile: {path}")
    content = path.read_text(encoding="utf-8")
    if "kind: AgentRuntimeProfile\n" not in content or f"  name: {channel}\n" not in content:
        raise ValueError(f"invalid profile name in {path}")
    refs = re.findall(r"^  releaseRef:\s*(\S+)\s*$", content, re.MULTILINE)
    if len(refs) != 1 or re.search(r"^  image:", content, re.MULTILINE):
        raise ValueError(f"{path}: expected releaseRef only (no inline image)")
    validate_release(refs[0])
    return refs[0]


def previously_selected(base: str, channel: str, ref: str) -> bool:
    # History is intentional: the earlier channel can have advanced after qualification.
    path = str(PROFILES / f"{channel}.yaml")
    return bool(git("log", "--format=%H", "-S", f"releaseRef: {ref}", base, "--", path))


def already_exists(base: str, channel: str) -> bool:
    return bool(git("ls-tree", "-r", "--name-only", base, "--", str(PROFILES / f"{channel}.yaml")))


def ensure_release_immutable(base: str) -> None:
    edited = git("diff", "--name-only", "--diff-filter=MD", f"{base}...HEAD", "--", str(PROFILES))
    for item in edited.splitlines():
        name = Path(item).stem
        if NAME.fullmatch(name) and name not in (*CHANNELS, *LEGACY):
            raise ValueError(f"immutable Hermes release changed or deleted: {item}")


def gate(base: str, body: str, author_type: str) -> list[str]:
    ensure_release_immutable(base)
    changed = set(git("diff", "--name-only", f"{base}...HEAD").splitlines())
    gated = [channel for channel in CHANNELS if str(PROFILES / f"{channel}.yaml") in changed]
    if not gated:
        return []

    refs = {channel: read_profile(channel) for channel in gated}
    new = [channel for channel in gated if not already_exists(base, channel)]
    if new:
        # One-time channel registration is not evidence that upstream Hermes
        # runs on Fabric. All three profiles MUST remain unbound by this PR.
        if set(new) != set(CHANNELS) or set(gated) != set(CHANNELS):
            raise ValueError("first channel registration must add all three channels together")
        if author_type.lower() == "bot":
            raise ValueError("initial channel registration requires human-reviewed PR")
        if field(body, "Hermes-Channel-Bootstrap") != "true":
            raise ValueError("channel bootstrap requires Hermes-Channel-Bootstrap: true")
        if any("/tenants/" in path and path.startswith("apps/60-services/txo-fabric/") for path in changed):
            raise ValueError("channel bootstrap may not modify tenant or AgentIdentity manifests")
        return gated

    if "hermes-canary" in gated or "hermes-stable" in gated:
        if author_type.lower() == "bot":
            raise ValueError("bot may nominate only hermes-edge, never canary/stable")

    if "hermes-canary" in gated:
        if not previously_selected(base, "hermes-edge", refs["hermes-canary"]):
            raise ValueError("canary release must first have appeared in committed edge history")

    if "hermes-stable" in gated:
        if not previously_selected(base, "hermes-canary", refs["hermes-stable"]):
            raise ValueError("stable release must first have appeared in committed canary history")
        for evidence_field in ("Hermes-Physical-Evidence", "Hermes-Recovery-Evidence",
                               "Hermes-Owner-Approval"):
            if not EVIDENCE_URL.fullmatch(field(body, evidence_field)):
                raise ValueError(f"{evidence_field} must link an actual GitHub issue comment")
        if not SNAPSHOT.fullmatch(field(body, "Hermes-GitOps-Snapshot")):
            raise ValueError("stable requires an exact dev-v* snapshot reference")

    return gated


if __name__ == "__main__":
    if len(sys.argv) != 4:
        raise SystemExit("usage: guard-hermes-release-channels.py BASE_REF PR_BODY AUTHOR_TYPE")
    try:
        channels = gate(sys.argv[1], sys.argv[2], sys.argv[3])
        print("Hermes channel gate OK: " + (", ".join(channels) if channels else "no channel pointers moved"))
    except (ValueError, subprocess.CalledProcessError) as exc:
        raise SystemExit(f"Hermes channel gate FAILED: {exc}") from exc
