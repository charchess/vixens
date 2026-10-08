#!/usr/bin/env python3
"""Nominate an immutable official upstream Hermes release for edge only.

Never changes canary, stable, default, legacy dev or customer AgentIdentity.
Creates a review-only GitOps candidate; the publishing workflow opens the PR.
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

DIGEST = re.compile(r"^[a-z0-9][a-z0-9._:/-]*@sha256:([0-9a-f]{64})$")
TAG = re.compile(r"^v[0-9]+(?:[.][0-9]+){1,3}$")
RELATIVE = Path("apps/60-services/txo-fabric/operator/config/profiles")


def nominate(root: Path, engine_image: str, plugin_image: str, upstream_tag: str) -> str:
    engine, plugin = DIGEST.fullmatch(engine_image), DIGEST.fullmatch(plugin_image)
    if not engine or not plugin:
        raise ValueError("both Hermes and Hindsight images must be pinned to OCI sha256 digests")
    if not engine_image.startswith("nousresearch/hermes-agent@sha256:"):
        raise ValueError("only the official upstream Hermes engine may be nominated")
    if not plugin_image.startswith("ghcr.io/charchess/txo-hermes-hindsight-plugin@sha256:"):
        raise ValueError("only the pinned platform Hindsight payload may be nominated")
    if not TAG.fullmatch(upstream_tag):
        raise ValueError("upstream tag must be a versioned official tag (e.g. v0.21.6)")
    name = f"hermes-{upstream_tag[1:].replace('.', '-')}-{engine.group(1)[:12]}-{plugin.group(1)[:12]}"
    folder = root / RELATIVE
    release_path = folder / (name + ".yaml")
    release = (
        "---\n"
        f"# Official upstream Hermes release {upstream_tag}; immutable OCI digests.\n"
        "# Software candidate only; physical Fabric acceptance is separate.\n"
        "apiVersion: fabric.truxonline.io/v1alpha1\n"
        "kind: HermesRuntimeRelease\n"
        "metadata:\n"
        f"  name: {name}\n"
        "  annotations:\n"
        '    argocd.argoproj.io/sync-wave: "-15"\n'
        f'    fabric.truxonline.io/upstream-tag: "{upstream_tag}"\n'
        "  labels:\n"
        "    app.kubernetes.io/part-of: txo-fabric\n"
        "    app.kubernetes.io/component: runtime-release\n"
        "spec:\n"
        f"  image: {engine_image}\n"
        f"  hindsightPluginImage: {plugin_image}\n"
    )
    if release_path.exists() and release_path.read_text(encoding="utf-8") != release:
        raise ValueError(f"refusing to alter existing immutable release: {release_path}")

    profile = folder / "hermes-edge.yaml"
    lines = profile.read_text(encoding="utf-8").splitlines()
    if lines.count("  name: hermes-edge") != 1 or lines.count("kind: AgentRuntimeProfile") != 1:
        raise ValueError("unexpected edge profile")
    refs = [i for i, line in enumerate(lines) if line.startswith("  releaseRef: ")]
    if len(refs) != 1 or any(line.startswith("  image: ") for line in lines):
        raise ValueError("edge profile must contain one releaseRef and no inline image")
    lines[refs[0]] = f"  releaseRef: {name}"

    kustomize = folder / "kustomization.yaml"
    entries = kustomize.read_text(encoding="utf-8").splitlines()
    entry = f"  - {release_path.name}"
    if entry not in entries:
        entries.append(entry)

    # All validation before any filesystem writes.
    release_path.write_text(release, encoding="utf-8")
    kustomize.write_text("\n".join(entries) + "\n", encoding="utf-8")
    profile.write_text("\n".join(lines) + "\n", encoding="utf-8")
    return name


if __name__ == "__main__":
    if len(sys.argv) != 4:
        raise SystemExit("usage: nominate_edge.py HERMES@sha256:... PLUGIN@sha256:... UPSTREAM_TAG")
    try:
        print(nominate(Path.cwd(), sys.argv[1], sys.argv[2], sys.argv[3]))
    except (ValueError, FileNotFoundError) as exc:
        raise SystemExit(str(exc)) from exc
