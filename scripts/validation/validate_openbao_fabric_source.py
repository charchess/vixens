#!/usr/bin/env python3
"""#4046: assert the Fabric OpenBao foundation is SOURCE ONLY.

This checks repository intent, not a live cluster, secret values, or
OpenBao availability. No third-party Python packages required.
Run from repository root; optional --base is a git ref for diff guards.
"""
from __future__ import annotations

import argparse
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
FABRIC = ROOT / "apps/60-services/txo-fabric"
NEW = FABRIC / "openbao-fabric"
INFRA = ROOT / "apps/00-infra/openbao/base/cluster-secret-store.yaml"
PROD = FABRIC / "overlays/prod/kustomization.yaml"
ARGO_PROD = ROOT / "argocd/overlays/prod/apps/openbao-secrets.yaml"

EXPECTED_POLICIES = {
    "platform-webui-bff.hcl.example":
        'fabric/data/platform/txo-fabric/webui-bff',
    "operator-authentik-reader.hcl.example":
        'fabric/data/platform/txo-fabric/authentik-reader',
    "operator-openfga-reader.hcl.example":
        'fabric/data/platform/txo-fabric/openfga-reader',
    "tenant-one-id.hcl.example":
        'fabric/data/tenants/<validated-immutable-fabric-tenant-id>/*',
}
REQUIRED_DOCS = [
    ROOT / "docs/adr/041-separate-openbao-fabric-secret-plane.md",
    ROOT / "docs/troubleshooting/txo-fabric-openbao-bootstrap-and-restore.md",
    NEW / "README.md",
    NEW / "contracts/authority-and-manifest-envelope.md",
    NEW / "contracts/workload-authz-matrix.md",
]
# Patterns target *assigned secret material*, not legitimate documentation
# about forbidden root tokens, credential stores or recovery procedures.
SENSITIVE_ASSIGNMENT = re.compile(
    r"""(?im)^\s*(?:["']?)"""
    r"(?:root[_-]?token|unseal[_-]?key|recovery[_-]?key|"
    r"private[_-]?key|password|access[_-]?token|secret[_-]?key)"
    r"""(?:["']?)\s*[:=]\s*["']?(?!\$|\{\{|<|#|\[|null\b)"""
    r"[A-Za-z0-9+/=_-]{12,}"
)
SECRET_KIND = re.compile(r"(?m)^kind:\s*(?:Secret|ExternalSecret|SecretStore|ClusterSecretStore)\s*$")
LIVE_KINDS = re.compile(
    r"(?m)^kind:\s*(?:StatefulSet|Deployment|Service|Ingress|HTTPRoute|"
    r"Gateway|Namespace|PersistentVolume|PersistentVolumeClaim|Job|CronJob)\s*$"
)


def check(ok: bool, reason: str) -> None:
    if not ok:
        raise AssertionError(reason)


def git(*args: str) -> str:
    return subprocess.check_output(
        ["git", "-C", str(ROOT), *args], text=True
    ).strip()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base", default=None, help="Git base ref, e.g. origin/main")
    args = parser.parse_args()
    for p in REQUIRED_DOCS:
        check(p.is_file(), f"missing source contract {p.relative_to(ROOT)}")
    check(not (NEW / "kustomization.yaml").exists(), "runnable Fabric OpenBao Kustomization detected")
    allowed_suffixes = {".md", ".example"}
    for file in NEW.rglob("*"):
        if not file.is_file():
            continue
        check(file.suffix in allowed_suffixes, f"runnable/unknown source in openbao-fabric: {file}")
        text = file.read_text(encoding="utf-8")
        check(not SECRET_KIND.search(text), f"Secret/ESO manifest in {file}")
        check(not LIVE_KINDS.search(text), f"live Kubernetes manifest in {file}")
        check(not SENSITIVE_ASSIGNMENT.search(text), f"possible credential assignment in {file}")
        check("0.0.0.0/0" not in text or file.suffix == ".md",
              f"unsafe broad network example {file}")
    policy_files = {p.name: p for p in (NEW / "policies").glob("*.hcl.example")}
    check(set(policy_files) == set(EXPECTED_POLICIES), "wrong or missing non-installable policies")
    for name, expected in EXPECTED_POLICIES.items():
        text = policy_files[name].read_text(encoding="utf-8")
        paths = re.findall(r'(?m)^path\s+"([^"]+)"\s*\{', text)
        check(paths == [expected], f"unexpected paths in {name}: {paths!r}")
        check(
            re.findall(r'(?m)^\s*capabilities\s*=\s*\[([^]]+)\]', text) == ['"read"'],
            f"non-read capability in {name}"
        )
        check("fabric/data/tenants/*" not in text, f"wildcard tenant policy in {name}")
        check(not any(path.startswith("fabric/metadata/") for path in paths),\n              f"metadata list path in {name}")
        check(not re.search(r"(?m)^path\s+\"(?:sys/|auth/)", text),
              f"administrative OpenBao privilege in {name}")
    tenant = policy_files["tenant-one-id.hcl.example"].read_text(encoding="utf-8")
    check("<validated-immutable-fabric-tenant-id>" in tenant, "tenant policy placeholder removed")
    check("not installable" in tenant.lower() or "not install" in tenant.lower(),
          "tenant policy install warning missing")

    prod = PROD.read_text(encoding="utf-8")
    argo_prod = ARGO_PROD.read_text(encoding="utf-8")
    check("openbao-fabric" not in prod, "OpenBao Fabric referenced in TXO production overlay")
    check("openbao-fabric" not in argo_prod, "OpenBao Fabric referenced in Argo production app")
    check("path: apps/00-infra/openbao/overlays/prod" in argo_prod,
          "OpenBao Infra target altered")
    infra = INFRA.read_text(encoding="utf-8")
    check("name: openbao\n" in infra, "Infra store identity changed")
    check("tokenSecretRef:" in infra and "name: openbao-token" in infra,
          "Infra tokenSecretRef modified")

    if args.base:
        modified = git("diff", "--name-only", f"{args.base}...HEAD").splitlines()
        protected_prefixes = (
            "apps/00-infra/openbao/",
            "argocd/",
            "apps/60-services/txo-fabric/overlays/",
            "apps/60-services/txo-fabric/openfga/",
            "apps/60-services/txo-fabric/operator/",
            "apps/60-services/txo-fabric/webui/",
        )
        for name in modified:
            check(not name.startswith(protected_prefixes), f"out-of-scope activation file changed: {name}")
            p = ROOT / name
            if p.is_file():
                data = p.read_text(encoding="utf-8")
                check(not SENSITIVE_ASSIGNMENT.search(data),
                      f"possible hardcoded secret in changed file: {name}")
                if name.startswith("apps/60-services/txo-fabric/"):
                    check("kind: ExternalSecret" not in data,
                          f"new TXO Infra ESO consumer unexpectedly staged in {name}")
        check("apps/00-infra/openbao/base/cluster-secret-store.yaml" not in modified,
              "historical Infra store modified")
        check("apps/60-services/txo-fabric/overlays/prod/kustomization.yaml" not in modified,
              "TXO production composition modified")
    print("OpenBao Fabric source-only security contract: PASS (repository-only, no runtime proof)")


if __name__ == "__main__":
    try:
        main()
    except (AssertionError, subprocess.CalledProcessError, OSError) as exc:
        print(f"OpenBao Fabric source-only security contract: FAIL ({exc})", file=sys.stderr)
        sys.exit(1)
