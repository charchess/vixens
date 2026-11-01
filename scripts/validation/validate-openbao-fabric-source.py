#!/usr/bin/env python3
"""#4046: OpenBao Fabric empty-POC invariant guard (does not touch a cluster).

Usage in CI: git fetch origin main; python3 scripts/validation/validate-openbao-fabric-source.py
         --base origin/main --head HEAD --prod-render /tmp/txo-fabric-prod.yaml
"""

from __future__ import annotations

import argparse
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
FABRIC_PREFIX = "apps/60-services/txo-fabric/openbao-fabric/"
INFRA_CONTRACT_PREFIX = "apps/00-infra/openbao/transit-seal-contract/"
INFRA_POLICY = INFRA_CONTRACT_PREFIX + "txo-fabric-seal.hcl.example"
INFRA_README = INFRA_CONTRACT_PREFIX + "README.md"
INFRA_EVIDENCE_RUNBOOK = "docs/runbooks/00-infra/openbao-transit-seal-evidence.md"
LEGACY_INFRA_STORE = "apps/00-infra/openbao/base/cluster-secret-store.yaml"
ALLOWED_FILES = (".md", ".example")
POC_YAML = {"base/kustomization.yaml", "base/configmap.yaml", "base/statefulset.yaml", "base/service.yaml", "base/networkpolicy.yaml", "base/certificate.yaml", "transit-staged/kustomization.yaml", "transit-staged/serviceaccount.yaml", "overlays/dev/kustomization.yaml", "overlays/prod/kustomization.yaml", "overlays/prod/storage/retained-pvc.yaml", "overlays/prod/storage/retained-statefulset-volume.yaml"}
# One-time, exact desired-state retirement of retired CSI. Keep this
# allow-list closed: any new path needs an explicit review rather than inheriting
# the OpenBao/Fabric exception.
LEGACY_STORAGE_RETIREMENT_PATHS = frozenset({
    'apps/00-infra/kyverno/base/policies/policy-exception-csi-probes.yaml',
    'apps/01-storage/nfs-storage/README.md',
    'apps/04-databases/postgresql-shared/overlays/prod/recovery-staging/postgresql-shared-recovery.yaml',
    'docs/adr/043-provider-neutral-storage-contracts.md',
    'docs/reports/TXO-FABRIC-STORAGE-RESILIENCE-ROADMAP.md',
    'docs/troubleshooting/synelia-csi-umi-preflight.md',
    '.github/workflows/synelia-csi-ci.yaml',
    'apps/60-services/txo-fabric/operator/config/profiles/workspace-shared.yaml',
    'argocd/overlays/prod/apps/synelia-prod-csi.yaml',
    'argocd/overlays/prod/kustomization.yaml',
    'apps/00-infra/crowdsec/values/common.yaml',
    'apps/00-infra/crowdsec/values/prod.yaml',
    'apps/00-infra/snapshot-controller/base/snapshot-class.yaml',
    'apps/01-storage/synelia-csi/prod/values.yaml',
    'apps/01-storage/synelia-csi/umi/values.yaml',
    'apps/01-storage/truenas-csi/README.md',
    'apps/01-storage/truenas-csi/bootstrap/base/kustomization.yaml',
    'apps/01-storage/truenas-csi/bootstrap/base/namespace.yaml',
    'apps/01-storage/truenas-csi/bootstrap/overlays/dev/kustomization.yaml',
    'apps/01-storage/truenas-csi/bootstrap/overlays/prod/cilium-networkpolicy.yaml',
    'apps/01-storage/truenas-csi/bootstrap/overlays/prod/externalsecret-iscsi-config.yaml',
    'apps/01-storage/truenas-csi/bootstrap/overlays/prod/externalsecret-nfs-config.yaml',
    'apps/01-storage/truenas-csi/bootstrap/overlays/prod/kustomization.yaml',
    'apps/01-storage/truenas-csi/openbao/base/kustomization.yaml',
    'apps/01-storage/truenas-csi/openbao/base/truenas-csi-externalsecrets.yaml',
    'apps/01-storage/truenas-csi/openbao/overlays/umi/kustomization.yaml',
    'apps/01-storage/truenas-csi/secret-templates/.gitignore',
    'apps/01-storage/truenas-csi/secret-templates/truenas-iscsi-driver-config.example.yaml',
    'apps/01-storage/truenas-csi/secret-templates/truenas-nfs-driver-config.example.yaml',
    'apps/01-storage/truenas-csi/values/iscsi-prod.yaml',
    'apps/01-storage/truenas-csi/values/iscsi-umi.yaml',
    'apps/01-storage/truenas-csi/values/nfs-prod.yaml',
    'apps/01-storage/truenas-csi/values/nfs-umi.yaml',
    'apps/02-monitoring/loki/base/statefulset.yaml',
    'apps/02-monitoring/prometheus/base/values.yaml',
    'apps/04-databases/mariadb-shared/base/statefulset.yaml',
    'apps/04-databases/mongodb-shared/base/statefulset.yaml',
    'apps/04-databases/postgresql-shared/base/cluster.yaml',
    'apps/10-home/homeassistant/base/pvc.yaml',
    'apps/10-home/homeassistant/overlays/dev/kustomization.yaml',
    'apps/10-home/homeassistant/overlays/prod/kustomization.yaml',
    'apps/20-media/frigate/base/pvc.yaml',
    'apps/20-media/frigate/overlays/dev/pvc-patch.yaml',
    'apps/20-media/frigate/overlays/prod/pvc-patch.yaml',
    'apps/20-media/hydrus-client/base/pvc.yaml',
    'apps/20-media/jellyfin/base/pvc.yaml',
    'apps/20-media/jellyfin/overlays/prod/pvc-patch.yaml',
    'apps/20-media/music-assistant/base/pvc.yaml',
    'apps/40-network/adguard-home/base/pvc.yaml',
    'apps/40-network/netbird/base/pvc-data.yaml',
    'apps/40-network/netbird/base/pvc-router-data.yaml',
    'apps/60-services/actual-budget/base/pvc.yaml',
    'apps/60-services/mail/base/mailserver-statefulset.yaml',
    'apps/60-services/sakapuss/overlays/prod/pvc-storageclass.yaml',
    'apps/60-services/txo-fabric/openbao-fabric/base/statefulset.yaml',
    'apps/60-services/txo-fabric/operator/config/profiles/cliproxyapi-standard.yaml',
    'apps/60-services/txo-fabric/operator/config/profiles/hermes-canary.yaml',
    'apps/60-services/txo-fabric/operator/config/profiles/hermes-default.yaml',
    'apps/60-services/txo-fabric/operator/config/profiles/hermes-dev.yaml',
    'apps/60-services/txo-fabric/operator/config/profiles/hermes-edge.yaml',
    'apps/60-services/txo-fabric/operator/config/profiles/hermes-stable.yaml',
    'apps/60-services/txo-fabric/operator/config/profiles/hermes-upgrade-canary.yaml',
    'apps/60-services/txo-fabric/operator/config/profiles/workspace-shared.yaml',
    'apps/60-services/txo-fabric/operator/internal/controller/adoption_test.go',
    'apps/60-services/txo-fabric/operator/internal/controller/agent_storage_policy_test.go',
    'apps/60-services/txo-fabric/operator/internal/controller/reconcile_test.go',
    'apps/60-services/txo-fabric/operator/internal/controller/sharedworkspace_controller_test.go',
    'apps/60-services/txo-fabric/operator/internal/controller/tenantbundle_ai_credential_broker_test.go',
    'apps/60-services/txo-fabric/tenants/hairem/tenantbundle.yaml',
    'apps/60-services/txo-fabric/tenants/indiba/tenantbundle.yaml',
    'apps/70-tools/penpot/base/pvc.yaml',
    'apps/70-tools/trilium/base/pvc.yaml',
    'apps/99-test/bookshelf/base/pvc.yaml',
    'argocd/overlays/dev/apps/truenas-csi-iscsi-umi.yaml',
    'argocd/overlays/dev/apps/truenas-csi-nfs-umi.yaml',
    'argocd/overlays/dev/apps/truenas-csi-secrets-umi.yaml',
    'argocd/overlays/dev/kustomization.yaml',
    'argocd/overlays/prod/apps/truenas-csi-iscsi-umi.yaml',
    'argocd/overlays/prod/apps/truenas-csi-iscsi.yaml',
    'argocd/overlays/prod/apps/truenas-csi-nfs-umi.yaml',
    'argocd/overlays/prod/apps/truenas-csi-nfs.yaml',
    'argocd/overlays/prod/apps/truenas-csi-secrets-umi.yaml',
    'argocd/overlays/prod/apps/truenas-csi-secrets.yaml',
    'argocd/overlays/prod/kustomization.yaml',
    'scripts/validation/test_openbao_fabric_source.py',
    'scripts/validation/tests/test_hermes_4185_manual_gate.py',
    'scripts/validation/tests/test_hermes_4185_recovery_cutover.py',
    'scripts/validation/validate-openbao-fabric-poc.py',
    '.github/workflows/validate.yaml',
    'scripts/validation/tests/test_truenas_csi_umi_openbao.py',
    'scripts/validation/tests/test_synelia_csi_source.py',
    'scripts/validation/guard-hermes-release-channels.py',
    'scripts/validation/tests/test_guard_hermes_release_channels.py',
    'scripts/validation/validate-openbao-fabric-source.py',
})

PROTECTED_STORAGE_PATHS = {
    "apps/60-services/txo-fabric/openbao-fabric/overlays/prod/kustomization.yaml",
    "apps/60-services/txo-fabric/openbao-fabric/overlays/prod/storage/retained-pvc.yaml",
    "apps/60-services/txo-fabric/openbao-fabric/overlays/prod/storage/retained-statefulset-volume.yaml",
    "scripts/validation/validate-openbao-fabric-source.py",
    "scripts/validation/validate-openbao-fabric-poc.py",
    "scripts/validation/test_openbao_fabric_source.py",
}
# Sensitive material MUST never appear in new source, even commented examples.
ASSIGNMENT = re.compile(
    r"""(?ix)
    \b(?:root[_-]?token|unseal[_-]?key|recovery[_-]?key|private[_-]?key|
         vault[_-]?token|bao[_-]?token|password|client[_-]?secret)
    \s*[:=][ \t]*["']?[A-Za-z0-9+/=_-]{8,}
    """,
)
JWT_LIKE = re.compile(r"\beyJ[A-Za-z0-9_-]{16,}\.eyJ[A-Za-z0-9_-]{16,}\.[A-Za-z0-9_-]{12,}\b")
PEM_LIKE = re.compile(r"-----BEGIN (?:[A-Z ]*PRIVATE KEY|OPENSSH PRIVATE KEY)-----")
BASE64_MATERIAL = re.compile(r"\b[A-Za-z0-9+/]{100,}={0,2}\b")
FORBIDDEN_DOCS = ("kind: Secret\n", "stringData:", "data:\n  password:")
FORBIDDEN_RESOURCES = re.compile(
    r"(?m)^kind:\s*(?:Deployment|StatefulSet|Service|Secret|"
    r"ExternalSecret|SecretStore|ClusterSecretStore|PersistentVolumeClaim|"
    r"PersistentVolume|Namespace|Application)\s*$"
)
PRODUCTION_MARKER = re.compile(r"(?i)openbao-fabric(?:-store)?")
# No newly added TXO ExternalSecret is allowed to inherit the Infra role.
INFRA_STORE_ADDITION = re.compile(r"(?m)^\+\s*name:\s*openbao\s*$")
FORBIDDEN_INFRA_CONTRACT = re.compile(
    r"(?imx)tls_skip_verify|http://|\b(?:0\.0\.0\.0/0|::/0)\b|"
    r"\b(?:loadbalancer|nodeport|ingress|externalname)\b|"
    r"\b(?:static[_ -]?token|bearer[_ -]?token)\s*[:=]|"
    r"^\s*(?:token|client_token)\s*[:=]"
)
INFRA_POLICY_PATH = re.compile(
    r'(?ms)^\s*path\s+"([^"]+)"\s*\{\s*'
    r'capabilities\s*=\s*\[\s*"([^"]+)"\s*\]\s*\}\s*$'
)


# Prompt 07 source-only interface oracle. This is NOT a controller, network
# endpoint, OpenBao client or authorized policy installer. The trusted
# 'canonical' argument represents a VERIFIED reread by a future independent
# boundary, never caller JSON. Reject-all is the default for absent facts.
def provisioning_idempotency_key(canonical: dict, action: str) -> str:
    """Deterministic non-secret comparison key, NOT authentication."""
    import hashlib
    import json

    source = canonical.get("source", {})
    workload = canonical.get("workload", {})
    fingerprint = {
        "action": action,
        "sourceUID": source.get("uid"),
        "sourceGeneration": source.get("generation"),
        "fabricTenantId": canonical.get("fabricTenantId"),
        "namespace": workload.get("namespace"),
        "serviceAccount": workload.get("serviceAccount"),
        "audience": workload.get("audience"),
        "template": "tenant-runtime-v1",
        "contract": "v1",
    }
    encoded = json.dumps(fingerprint, sort_keys=True, separators=(",", ":")).encode()
    return hashlib.sha256(encoded).hexdigest()


def validate_provisioning_request(request: object, canonical: object) -> list[str]:
    """Pure contract validator, without OpenBao or Kubernetes side effects.

    Future implementation MUST independently authenticate its operator caller,
    reread source and workload bindings, and enforce the same closed template
    in a separate provisioner. This oracle alone cannot authorize a role.
    """
    if not isinstance(request, dict) or not isinstance(canonical, dict):
        return ["InvalidContract"]
    required = {
        "contractVersion", "action", "source", "fabricTenantId",
        "workload", "pathTemplate", "idempotencyKey",
    }
    if set(request) != required:
        return ["ForbiddenFields"]
    source_keys = {"kind", "apiVersion", "name", "uid", "generation"}
    workload_keys = {"namespace", "serviceAccount", "audience"}
    source = request.get("source")
    workload = request.get("workload")
    verified_source = canonical.get("source")
    verified_workload = canonical.get("workload")
    if not isinstance(source, dict) or set(source) != source_keys:
        return ["InvalidSource"]
    if not isinstance(workload, dict) or set(workload) != workload_keys:
        return ["InvalidWorkloadBinding"]
    if (not isinstance(verified_source, dict) or
            not isinstance(verified_workload, dict) or
            not canonical.get("operatorAuthenticated", False)):
        return ["UntrustedReconciler"]
    if (source.get("kind") != "TenantBundle" or
            source.get("apiVersion") != "fabric.truxonline.io/v1alpha1" or
            not isinstance(source.get("name"), str) or
            not re.fullmatch(r"[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?", source["name"]) or
            not isinstance(source.get("uid"), str) or
            not re.fullmatch(r"[a-zA-Z0-9-]{8,64}", source["uid"]) or
            type(source.get("generation")) is not int or
            source["generation"] < 1):
        return ["InvalidSource"]
    tenant_id = request.get("fabricTenantId")
    if not isinstance(tenant_id, str) or not re.fullmatch(r"TEN[0-9]{5,}", tenant_id):
        return ["InvalidTenantID"]
    if (source != verified_source or
            tenant_id != canonical.get("fabricTenantId")):
        return ["UnknownOrStaleTenant"]
    if (any(not isinstance(workload.get(k), str) or
            not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9.-]{0,126}", workload[k])
            for k in workload_keys) or
            workload != verified_workload):
        return ["InvalidWorkloadBinding"]
    action = request.get("action")
    if request.get("contractVersion") != "v1" or not isinstance(action, str) or action not in {"Ensure", "Revoke"}:
        return ["InvalidAction"]
    if request.get("pathTemplate") != "tenant-runtime-v1":
        return ["UnapprovedTemplate"]
    lifecycle = canonical.get("lifecycle")
    if not isinstance(lifecycle, str):
        return ["LifecycleDenied"]
    if (action == "Ensure" and lifecycle != "Active") or (
        action == "Revoke" and lifecycle not in {"Active", "Parked", "Deleting"}
    ):
        return ["LifecycleDenied"]
    if canonical.get("fabricAvailable") is not True:
        return ["FabricUnavailable"]
    if (not isinstance(request.get("idempotencyKey"), str) or
            request["idempotencyKey"] != provisioning_idempotency_key(canonical, action)):
        return ["IdempotencyMismatch"]
    return []


SEAL_SA = "openbao-fabric-seal-bootstrap"
SEAL_NAMESPACE = "txo-fabric-system"
SEAL_AUDIENCE = "txo-fabric-infra-seal"
SEAL_TOKEN_TTL = 600
SEAL_KEY = "txo-fabric-seal"

def validate_staged_seal_identity(objects: object) -> list[str]:
    """Closed unselected YAML envelope, not runtime seal acceptance."""
    if not isinstance(objects, list) or len(objects) != 1:
        return ["stage must contain exactly one dedicated SA"]
    if objects[0] != {
        "apiVersion": "v1", "kind": "ServiceAccount",
        "metadata": {
            "name": SEAL_SA, "namespace": SEAL_NAMESPACE,
            "labels": {
                "app.kubernetes.io/name": SEAL_SA,
                "app.kubernetes.io/component": "seal-bootstrap",
                "app.kubernetes.io/part-of": "txo-fabric",
            },
        },
        "automountServiceAccountToken": False,
    }:
        return ["wrong staged SA namespace, identity or authority"]
    return []


def validate_infra_transit_policy(content: str) -> list[str]:
    """Validate the non-executable #4071 policy example exactly, never install it."""
    matches = INFRA_POLICY_PATH.findall(content)
    expected = [
        ("transit/encrypt/txo-fabric-seal", "update"),
        ("transit/decrypt/txo-fabric-seal", "update"),
    ]
    if matches != expected:
        return ["Infra policy example must grant only exact Transit encrypt/decrypt update"]
    stripped = INFRA_POLICY_PATH.sub("", content)
    stripped = re.sub(r"(?m)^\s*#.*$", "", stripped)
    if stripped.strip():
        return ["Infra policy example contains material outside the two exact rules"]
    return []


def assert_infra_evidence_safe(content: str) -> list[str]:
    """Reject material that would turn the evidence runbook into configuration."""
    errors: list[str] = []
    if ASSIGNMENT.search(content) or JWT_LIKE.search(content) or PEM_LIKE.search(content):
        errors.append("credential-like value detected in Infra Transit evidence")
    if BASE64_MATERIAL.search(content):
        errors.append("unexpected long encoded payload in Infra Transit evidence")
    if FORBIDDEN_INFRA_CONTRACT.search(content):
        errors.append("forbidden transport, token, public route, or broad egress in Infra Transit evidence")
    if FORBIDDEN_RESOURCES.search(content) or re.search(r"(?m)^\s*apiVersion:\s*", content):
        errors.append("Infra Transit evidence contains runnable Kubernetes resource syntax")
    return errors


def assert_infra_contract_safe(path: str, content: str) -> list[str]:
    """Source-only #4071 envelope: no installable object or transport bypass."""
    errors: list[str] = []
    if path not in {INFRA_POLICY, INFRA_README}:
        errors.append("unexpected file in Infra Transit source contract")
    if ASSIGNMENT.search(content) or JWT_LIKE.search(content) or PEM_LIKE.search(content):
        errors.append("credential-like value detected in Infra Transit source contract")
    if BASE64_MATERIAL.search(content):
        errors.append("unexpected long encoded payload in Infra Transit source contract")
    if FORBIDDEN_INFRA_CONTRACT.search(content):
        errors.append("forbidden transport, token, public route, or broad egress in Infra Transit contract")
    if FORBIDDEN_RESOURCES.search(content) or re.search(r"(?m)^\s*apiVersion:\s*", content):
        errors.append("Infra Transit source contract contains runnable Kubernetes resource syntax")
    if path == INFRA_POLICY:
        errors.extend(validate_infra_transit_policy(content))
    return errors


def require_seal_migration_preflight(facts: object) -> list[str]:
    """Pure fail-closed preflight: no trust facts are sourced from Git."""
    required = {
        "infraTLS", "infraCA", "infraSNI", "tokenReview",
        "saAudience", "transitKeyPolicy", "periodicTokenRenewal",
        "exactPrivateEgress", "raftStorageContinuity",
        "originalShamirCustody", "explicitPhysicalApproval",
    }
    if not isinstance(facts, dict) or set(facts) != required:
        return ["incomplete independently verified seal preflight"]
    if any(type(facts[k]) is not bool or facts[k] is not True for k in required):
        return ["seal migration denied: missing verified prerequisite"]
    return []


def git(*args: str) -> str:
    return subprocess.check_output(["git", *args], cwd=ROOT, text=True)


def changed_paths(base: str, head: str) -> list[str]:
    return [
        p for p in git("diff", "--name-only", "--diff-filter=ACMRT", f"{base}...{head}").splitlines()
        if p.strip()
    ]


def assert_file_safe(path: str, content: str) -> list[str]:
    errors: list[str] = []
    is_contract = path.startswith(FABRIC_PREFIX)
    if is_contract and not (path.endswith(ALLOWED_FILES) or path[len(FABRIC_PREFIX):] in POC_YAML):
        errors.append("OpenBao Fabric source directory contains a runnable or unexpected file")
    if ASSIGNMENT.search(content) or JWT_LIKE.search(content) or PEM_LIKE.search(content):
        errors.append("credential-like value detected in source content")
    if BASE64_MATERIAL.search(content):
        errors.append("unexpected long encoded payload in source content")
    if is_contract:
        if path.endswith(ALLOWED_FILES) and FORBIDDEN_RESOURCES.search(content):
            errors.append("OpenBao Fabric source contains Kubernetes resource syntax")
        if path.endswith(".example"):
            if 'capabilities = ["read"]' not in content:
                errors.append("example policy not strictly read-only")
            if re.search(r"(?m)^\s*path\s+\"(?:sys/|auth/|identity/|fabric/metadata/)", content):
                errors.append("example policy contains admin, auth or metadata path")
            if 'fabric/data/tenants/*' in content:
                errors.append("all-tenant wildcard policy prohibited")
    return errors


def validate(base: str, head: str, prod_render: pathlib.Path) -> list[str]:
    problems: list[str] = []
    changed = changed_paths(base, head)
    # ADR-041 is already merged: small manifest fixes must not edit the
    # architectural decision solely to satisfy a change-set guard.
    # Require the authoritative ADR to continue existing in the checkout.
    expected_adr = "docs/adr/041-separate-openbao-fabric-secret-plane.md"
    if not (ROOT / expected_adr).is_file():
        problems.append("ADR-041 missing from repository")
    if (not any(p.startswith(FABRIC_PREFIX) or p.startswith(INFRA_CONTRACT_PREFIX) for p in changed)
            and not set(changed).issubset(PROTECTED_STORAGE_PATHS | LEGACY_STORAGE_RETIREMENT_PATHS)):
        problems.append("OpenBao Fabric or Infra Transit source contract directory missing")
    # Editing the protected storage manifests is not an Infra Transit seal change.
    # Permit only these exact reviewed surfaces; other Fabric changes still
    # require a separate Infra-contract review.
    retirement_changeset = set(changed).issubset(
        PROTECTED_STORAGE_PATHS | LEGACY_STORAGE_RETIREMENT_PATHS
    )
    if (not any(p.startswith(INFRA_CONTRACT_PREFIX) for p in changed)
            and not set(changed).issubset(PROTECTED_STORAGE_PATHS)
            and not retirement_changeset):
        problems.append("Infra Transit source contract directory missing")
    for required in (INFRA_POLICY, INFRA_README, INFRA_EVIDENCE_RUNBOOK):
        if not (ROOT / required).is_file():
            problems.append(f"required #4071 source contract file missing: {required}")
    if LEGACY_INFRA_STORE in changed:
        problems.append("legacy Infra ESO ClusterSecretStore must remain unchanged")
    protected = [
        "apps/00-infra/openbao/",
        "argocd/overlays/prod/",
        "apps/60-services/txo-fabric/overlays/prod/",
        "apps/60-services/txo-fabric/operator/",
        "apps/60-services/txo-fabric/webui/",
    ]
    for path in changed:
        allowed_infra_contract = path.startswith(INFRA_CONTRACT_PREFIX)
        if (path.startswith(tuple(protected)) and not allowed_infra_contract and
                not retirement_changeset and
                path not in {"apps/60-services/txo-fabric/overlays/dev/kustomization.yaml", "apps/60-services/txo-fabric/overlays/prod/kustomization.yaml"}):
            problems.append(f"protected live GitOps/consumer path changed: {path}")
        full = ROOT / path
        if not full.is_file():
            continue
        content = full.read_text(encoding="utf-8")
        for violation in assert_file_safe(path, content):
            problems.append(f"{path}: {violation}")
        if allowed_infra_contract:
            for violation in assert_infra_contract_safe(path, content):
                problems.append(f"{path}: {violation}")
        if path == INFRA_EVIDENCE_RUNBOOK:
            for violation in assert_infra_evidence_safe(content):
                problems.append(f"{path}: {violation}")
    for path in (
        "apps/60-services/txo-fabric/openbao-fabric/base/kustomization.yaml",
        "apps/60-services/txo-fabric/openbao-fabric/overlays/dev/kustomization.yaml",
        "apps/60-services/txo-fabric/openbao-fabric/overlays/prod/kustomization.yaml",
        "apps/60-services/txo-fabric/overlays/dev/kustomization.yaml",
        "apps/60-services/txo-fabric/overlays/prod/kustomization.yaml",
    ):
        if (ROOT / path).is_file() and "transit-staged" in (ROOT / path).read_text(encoding="utf-8"):
            problems.append("Transit preflight selected without approved migration")
    # Existing production Infra points to HTTP: this MUST NOT be
    # mistaken for a verified TLS anchor or a ready Transit seal.
    staged = ROOT / "apps/60-services/txo-fabric/openbao-fabric/transit-staged"
    if staged.is_dir():
        k = staged / "kustomization.yaml"
        if not k.is_file() or "serviceaccount.yaml" not in k.read_text(encoding="utf-8"):
            problems.append("seal identity stage missing or not isolated")
        if any(f.is_file() for f in staged.glob("*") if f.name not in {
            "README.md", "serviceaccount.yaml", "kustomization.yaml"
        }):
            problems.append("unauthorized executable Transit stage file")
    infra_contract = ROOT / INFRA_CONTRACT_PREFIX
    if infra_contract.is_dir():
        names = {p.name for p in infra_contract.iterdir() if p.is_file()}
        if names != {"README.md", "txo-fabric-seal.hcl.example"}:
            problems.append("Infra Transit contract must contain only its README and policy example")
    else:
        problems.append("Infra Transit contract directory missing")
    for kustomization in (ROOT / "apps/00-infra/openbao").rglob("kustomization.yaml"):
        if "transit-seal-contract" in kustomization.read_text(encoding="utf-8"):
            problems.append("non-executable Infra Transit example must never be selected")
    # Additional delta scan catches a newly introduced Infra store binding.
    patch = git("diff", "--unified=0", f"{base}...{head}", "--", "apps/60-services/txo-fabric/")
    if INFRA_STORE_ADDITION.search(patch):
        problems.append("new TXO Infra ClusterSecretStore/openbao reference is forbidden")
    if re.search(r"(?m)^\+\s*(?:kind:\s*Secret|secretStoreRef:\s*openbao)", patch):
        problems.append("new Secret or Infra store reference in TXO source changes")
    # Prove no hidden activation in the rendered desired-state output, not
    # just in a Kustomization path string.
    if not prod_render.is_file():
        problems.append("production Kustomize render evidence missing")
    else:
        body = prod_render.read_text(encoding="utf-8")
        if not PRODUCTION_MARKER.search(body):
            problems.append("Fabric OpenBao missing from authorized POC production TXO render")
    if problems:
        return problems
    print(f"PASS: {len(changed)} changed source files checked; Fabric POC selected (full render safety validated separately)")
    print("INFO: historic TXO OpenFGA uses Infra ESO; migration is deferred, not silently asserted clean")
    return []


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--base", default="origin/main")
    ap.add_argument("--head", default="HEAD")
    ap.add_argument("--prod-render", required=True, type=pathlib.Path)
    args = ap.parse_args()
    problems = validate(args.base, args.head, args.prod_render)
    for msg in problems:
        print("FAIL: " + msg, file=sys.stderr)
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
