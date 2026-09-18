#!/usr/bin/env python3
"""Validate repository-level Vixens invariants.

This script deliberately validates only current sources of truth. Historical ADRs
and post-mortems are allowed to mention superseded technologies and workflows.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
ERRORS: list[str] = []

FORBIDDEN_PATHS = [
    ".beads",
    ".bmad-core",
    ".bmad-infrastructure-devops",
    ".clinerules",
    ".context",
    ".opencode",
    ".serena",
    "CLAUDE.md",
    "GEMINI.md",
    ".infisical.json",
    "docs/STATUS.md",
    "docs/reports",
]

REQUIRED_PATHS = [
    "README.md",
    "WORKFLOW.md",
    "AGENTS.md",
    "docs/README.md",
    "docs/architecture.md",
    "docs/adr/000-index.md",
    "docs/guides/gitops-workflow.md",
    "docs/guides/promotion-workflow.md",
    "docs/guides/secret-management.md",
    "docs/guides/adding-new-application.md",
]


def error(message: str) -> None:
    ERRORS.append(message)


def text_files(root: Path):
    for path in root.rglob("*"):
        if path.is_file() and path.suffix.lower() in {".yaml", ".yml", ".md", ".json", ".toml"}:
            yield path


for rel in FORBIDDEN_PATHS:
    if (ROOT / rel).exists():
        error(f"legacy path must not exist: {rel}")

for rel in REQUIRED_PATHS:
    if not (ROOT / rel).exists():
        error(f"required canonical document missing: {rel}")

# Active GitOps manifests must not use retired branch names or the retired
# Infisical CRD. Historical documentation is intentionally excluded.
for base in (ROOT / "apps", ROOT / "argocd"):
    for path in text_files(base):
        content = path.read_text(errors="replace")
        rel = path.relative_to(ROOT)
        if "feat/hermes-staging" in content:
            error(f"retired feature revision referenced by {rel}")
        if "secrets.infisical.com" in content or re.search(r"(?m)^kind:\s*InfisicalSecret\s*$", content):
            error(f"retired Infisical CRD referenced by {rel}")

for path in (ROOT / "argocd/overlays/dev/apps").glob("*.yaml"):
    content = path.read_text(errors="replace")
    if re.search(r"(?m)^\s*targetRevision:\s*['\"]?(dev|test|staging)['\"]?\s*$", content):
        error(f"retired environment branch referenced by {path.relative_to(ROOT)}")

for path in (ROOT / "argocd/overlays/prod/apps").glob("*.yaml"):
    content = path.read_text(errors="replace")
    if re.search(r"(?m)^\s*targetRevision:\s*['\"]?(main|dev|test|staging|prod)['\"]?\s*$", content):
        error(f"invalid production repository revision in {path.relative_to(ROOT)}")

# Canonical docs must have resolvable relative Markdown links. Historical
# documents and per-app runbooks are intentionally not part of this gate yet.
doc_roots = [
    ROOT / "README.md",
    ROOT / "WORKFLOW.md",
    ROOT / "AGENTS.md",
    ROOT / "docs/README.md",
    ROOT / "docs/architecture.md",
]
doc_roots.extend((ROOT / "docs/guides").glob("*.md"))
doc_roots.extend((ROOT / "docs/reference").glob("*.md"))

link_re = re.compile(r"\[[^\]]*\]\(([^)]+)\)")
for path in doc_roots:
    if not path.exists():
        continue
    content = path.read_text(errors="replace")
    for href in link_re.findall(content):
        href = href.strip().strip("<>")
        if not href or href.startswith(("#", "http://", "https://", "mailto:")):
            continue
        href = href.split("#", 1)[0].split("?", 1)[0]
        target = (path.parent / href).resolve()
        try:
            target.relative_to(ROOT.resolve())
        except ValueError:
            error(f"link escapes repository in {path.relative_to(ROOT)}: {href}")
            continue
        if not target.exists():
            error(f"broken link in {path.relative_to(ROOT)}: {href}")

# The historical duplicate ADR-018 is preserved verbatim. ADR-029 is the
# canonical successor that resolves its numbering collision.
if not (ROOT / "docs/adr/018-openbao-external-secrets-and-nas-fqdn.md").exists():
    error("historical duplicate ADR-018 source record is missing")
if not (ROOT / "docs/adr/029-openbao-external-secrets-and-nas-fqdn.md").exists():
    error("canonical ADR-029 for OpenBao/External Secrets is missing")

if ERRORS:
    print("Repository contract validation failed:")
    for item in ERRORS:
        print(f" - {item}")
    sys.exit(1)

print("Repository contract validation OK")
