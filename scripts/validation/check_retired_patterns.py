#!/usr/bin/env python3
"""Fail when retired tooling/patterns reappear in active repository zones."""

from __future__ import annotations

import argparse
import re
import tempfile
from pathlib import Path

ACTIVE_PREFIXES = (
    ".github/actions/",
    ".github/workflows/",
    "apps/",
    "argocd/",
    "scripts/",
    ".opencode/skills/",
)

FORBIDDEN_PATHS = {
    ".infisical.json": "OpenBao + External Secrets Operator; do not restore Infisical repo config",
    ".github/dependabot.yml": "Renovate is the repository dependency manager",
    ".github/workflows/ci.yaml": "Validate & Security and the current focused workflows",
    "scripts/utils/synocli": "TrueNAS/current storage tooling; the Synology client was retired",
}

PATH_RULES = (
    (re.compile(r"(^|/)infisical([^/]*)(/|$)", re.IGNORECASE),
     "OpenBao/ExternalSecret naming in active paths"),
    (re.compile(r"(^|/)synology-csi([^/]*)(/|$)", re.IGNORECASE),
     "current storage naming; Synology CSI is retired"),
)

CONTENT_RULES = (
    (re.compile(r"apiVersion\s*:\s*secrets\.infisical\.com/", re.IGNORECASE),
     "external-secrets.io API with ExternalSecret"),
    (re.compile(r"kind\s*:\s*InfisicalSecret\b", re.IGNORECASE),
     "kind: ExternalSecret backed by ClusterSecretStore/openbao"),
)

CONTENT_SUFFIXES = {".yaml", ".yml", ".json", ".toml"}
SELF_PATH = "scripts/validation/check_retired_patterns.py"


def is_active(rel: str) -> bool:
    return rel in FORBIDDEN_PATHS or any(rel.startswith(prefix) for prefix in ACTIVE_PREFIXES)


def is_disabled(path: Path) -> bool:
    return any(part.endswith(".disabled") for part in path.parts)


def check_tree(root: Path) -> list[str]:
    root = root.resolve()
    violations: list[str] = []

    # Exact retired files are checked even though some live outside active prefixes.
    for rel, replacement in FORBIDDEN_PATHS.items():
        candidate = root / rel
        if candidate.exists():
            violations.append(
                f"{rel}: retired path is present; expected replacement: {replacement}"
            )

    scan_roots: list[Path] = []
    for prefix in ACTIVE_PREFIXES:
        base = root / prefix.rstrip("/")
        if base.exists():
            scan_roots.append(base)

    seen: set[Path] = set()
    for base in scan_roots:
        for path in base.rglob("*"):
            if not path.is_file() or path in seen:
                continue
            seen.add(path)
            rel = path.relative_to(root).as_posix()
            if not is_active(rel) or is_disabled(path):
                continue

            for pattern, replacement in PATH_RULES:
                if pattern.search(rel):
                    violations.append(
                        f"{rel}: retired path pattern '{pattern.pattern}'; "
                        f"expected replacement: {replacement}"
                    )

            # The guard itself necessarily contains the literals it detects.
            if rel == SELF_PATH or path.suffix.lower() not in CONTENT_SUFFIXES:
                continue

            try:
                text = path.read_text(encoding="utf-8")
            except UnicodeDecodeError:
                continue

            for pattern, replacement in CONTENT_RULES:
                if pattern.search(text):
                    violations.append(
                        f"{rel}: retired content pattern '{pattern.pattern}'; "
                        f"expected replacement: {replacement}"
                    )

    return sorted(set(violations))


def write(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def self_test() -> None:
    with tempfile.TemporaryDirectory(prefix="retired-patterns-") as tmp:
        root = Path(tmp)

        # Active retired API must fail.
        write(
            root / "apps/demo/base/secret.yaml",
            "apiVersion: secrets.infisical.com/v1alpha1\nkind: InfisicalSecret\n",
        )
        violations = check_tree(root)
        assert any("apps/demo/base/secret.yaml" in item for item in violations), violations

        # Historical documents may preserve the same literal for provenance.
        write(
            root / "docs/adr/011-infisical-secrets-management.md",
            "apiVersion: secrets.infisical.com/v1alpha1\nkind: InfisicalSecret\n",
        )
        # Disabled manifests are intentionally inactive and must not block CI.
        write(
            root / "argocd/overlays/prod/apps/legacy.yaml.disabled",
            "apiVersion: secrets.infisical.com/v1alpha1\nkind: InfisicalSecret\n",
        )
        (root / "apps/demo/base/secret.yaml").unlink()
        assert check_tree(root) == [], check_tree(root)

        # Retired filenames/configs must fail even without retired contents.
        write(root / ".infisical.json", "{}\n")
        violations = check_tree(root)
        assert any(".infisical.json" in item for item in violations), violations
        (root / ".infisical.json").unlink()

        write(root / ".github/workflows/ci.yaml", "name: Legacy CI\n")
        violations = check_tree(root)
        assert any(".github/workflows/ci.yaml" in item for item in violations), violations
        (root / ".github/workflows/ci.yaml").unlink()

        # Active path names are guarded, while historical docs are outside scope.
        write(root / "apps/demo/base/infisical-secret.yaml", "kind: ConfigMap\n")
        violations = check_tree(root)
        assert any("infisical-secret.yaml" in item for item in violations), violations

    print("retired-pattern guard self-test: PASS")


def github_error(violation: str) -> str:
    """Render a file-aware GitHub Actions annotation when possible."""
    rel, separator, _ = violation.partition(":")
    if separator and rel:
        return f"::error file={rel}::{violation}"
    return f"::error::{violation}"


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("root", nargs="?", default=".", help="repository root")
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args()

    if args.self_test:
        self_test()
        return 0

    violations = check_tree(Path(args.root))
    if violations:
        print("Retired active patterns detected:")
        for violation in violations:
            print(github_error(violation))
        return 1

    print("retired-pattern guard: PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
