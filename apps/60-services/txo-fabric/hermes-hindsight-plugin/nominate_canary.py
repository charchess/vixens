#!/usr/bin/env python3
"""Create a review-only GitOps candidate: immutable runtime release + canary pointer.

stdlib-only, no Kubernetes/API access. The caller must create a PR; this
script never merges or promotes and must never update hermes-default.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

DIGEST = re.compile(r"^[a-zA-Z0-9][a-zA-Z0-9._:/-]*@sha256:([a-f0-9]{64})$")
PLUGIN_DIGEST = re.compile(r"^[a-z0-9][a-z0-9._:/-]*@sha256:([a-f0-9]{64})$")
RELATIVE = Path("apps/60-services/txo-fabric/operator/config/profiles")
UPSTREAM_VERSION = "2026-09-24"


def nominate(root: Path, engine_image: str, plugin_image: str) -> str:
    engine = DIGEST.fullmatch(engine_image)
    plugin = PLUGIN_DIGEST.fullmatch(plugin_image)
    if not engine or not plugin:
        raise ValueError("both Hermes engine and plugin images must be pinned to OCI sha256 digests")
    if not engine_image.startswith("nousresearch/hermes-agent@sha256:"):
        raise ValueError("release candidate must use the official Hermes engine")
    name = f"hermes-{UPSTREAM_VERSION}-{engine.group(1)[:12]}-{plugin.group(1)[:12]}"
    directory = root / RELATIVE
    release_path = directory / f"{name}.yaml"
    release = (
        "---\n"
        "# Immutable upstream Hermes and independent Hindsight provider payload.\n"
        "apiVersion: fabric.truxonline.io/v1alpha1\n"
        "kind: HermesRuntimeRelease\n"
        "metadata:\n"
        f"  name: {name}\n"
        "  annotations:\n"
        '    argocd.argoproj.io/sync-wave: "-15"\n'
        "  labels:\n"
        "    app.kubernetes.io/part-of: txo-fabric\n"
        "    app.kubernetes.io/component: runtime-release\n"
        "spec:\n"
        f"  image: {engine_image}\n"
        f"  hindsightPluginImage: {plugin_image}\n"
    )
    if release_path.exists() and release_path.read_text(encoding="utf-8") != release:
        raise ValueError(f"refusing to alter existing immutable release {release_path}")

    kustomize = directory / "kustomization.yaml"
    kustomize_lines = kustomize.read_text(encoding="utf-8").splitlines()
    entry = f"  - {release_path.name}"
    if entry not in kustomize_lines:
        kustomize_lines.append(entry)

    profile = directory / "hermes-upgrade-canary.yaml"
    lines = profile.read_text(encoding="utf-8").splitlines()
    image_positions = [i for i, line in enumerate(lines) if line.startswith("  image: ")]
    ref_positions = [i for i, line in enumerate(lines) if line.startswith("  releaseRef: ")]
    if len(image_positions) + len(ref_positions) != 1:
        raise ValueError("canary must have exactly one image or releaseRef field")
    position = (image_positions + ref_positions)[0]
    lines[position] = f"  releaseRef: {name}"

    # This is the legacy, image-specific bootstrap payload. Never erase other
    # profile policy and never silently drop unknown new bootstrap fields.
    bootstrap_positions = [i for i, line in enumerate(lines) if line == "  bootstrap:"]
    if bootstrap_positions:
        if len(bootstrap_positions) != 1:
            raise ValueError("multiple bootstrap sections")
        start = bootstrap_positions[0]
        end = start + 1
        while end < len(lines) and not (lines[end].startswith("  ") and not lines[end].startswith("    ")):
            end += 1
        payload = lines[start + 1:end]
        if len(payload) != 1 or not payload[0].startswith("    hindsightPluginImage: "):
            raise ValueError("canary bootstrap contains fields that cannot be migrated safely")
        del lines[start:end]
    # Validate the entire proposed candidate before writing any Git file.
    release_path.write_text(release, encoding="utf-8")
    kustomize.write_text("\n".join(kustomize_lines) + "\n", encoding="utf-8")
    profile.write_text("\n".join(lines) + "\n", encoding="utf-8")
    return name


if __name__ == "__main__":
    if len(sys.argv) != 3:
        raise SystemExit("usage: nominate_canary.py ENGINE_IMAGE@sha256:... PLUGIN_IMAGE@sha256:...")
    try:
        print(nominate(Path.cwd(), sys.argv[1], sys.argv[2]))
    except ValueError as exc:
        raise SystemExit(str(exc)) from exc
