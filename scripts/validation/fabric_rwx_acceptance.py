#!/usr/bin/env python3
"""Read-only structural RWX acceptance for TXO Fabric. Never certifies file data.

Usage:
  python3 scripts/validation/fabric_rwx_acceptance.py
  python3 scripts/validation/fabric_rwx_acceptance.py --snapshot inventory.json

Live mode executes ONLY kubectl get pvc/pv/pods -o json. Exit 2 = structural
failure; exit 3 = no structural failure but physical data proof still pending.
"""
import argparse
import json
from pathlib import Path
import subprocess
import sys

import yaml

ROOT = Path(__file__).resolve().parents[2]
TENANTS = ("hairem", "indiba")
LABEL = "fabric.truxonline.io/"
STORAGE_CLASS = "storage-rwx-retain"


def expected_claims(bundle):
    """Mirror the native workspace PVC naming contract, not CSI migration history."""
    workspace = bundle["spec"].get("workspace", {})
    claims = {}

    def add(scope, key, opts):
        for domain, prefix, reference, collaborative in (
            ("shared", "ws", "reference", "collaborative"),
            ("skills", "skills", "skillsReference", "skillsCollaborative"),
        ):
            root = f"{prefix}-org" if scope == "organization" else f"{prefix}-{scope}-{key}"
            for flag, mode, suffix in ((reference, "reference", "ref"),
                                       (collaborative, "collaborative", "rw")):
                if opts.get(flag, False):
                    name = f"{root}-{suffix}"
                    if name in claims:
                        raise ValueError(f"duplicate workspace claim {name}")
                    claims[name] = {"domain": domain, "scope": scope, "key": key, "mode": mode}
    add("organization", "organization", workspace.get("organization") or {})
    for scope, plural in (("group", "groups"), ("user", "users")):
        for item in workspace.get(plural) or []:
            add(scope, item["name"], item)
    return claims


def kube_get(kind, namespace=None):
    command = ["kubectl"]
    if namespace:
        command += ["-n", namespace]
    command += ["get", kind, "-o", "json"]
    result = subprocess.run(command, capture_output=True, text=True, check=True)
    return json.loads(result.stdout)["items"]


def live_snapshot(tenants):
    # Only Kubernetes GET operations; no secrets, exec, logs or write verbs.
    return {
        "pvcs": {name: kube_get("pvc", f"tenant-{name}") for name in tenants},
        "pods": {name: kube_get("pods", f"tenant-{name}") for name in tenants},
        "pvs": kube_get("pv"),
    }


def evaluate(bundles, snapshot):
    """Pure structural audit. Never conclude data bytes/functional access PASS."""
    findings = []
    report = {"claims": [], "findings": findings, "physical_data_proof": "PENDING"}

    def add(level, code, namespace, claim, detail=""):
        findings.append({"level": level, "code": code, "namespace": namespace,
                         "claim": claim, "detail": detail})

    pvs = {item["metadata"]["name"]: item for item in snapshot.get("pvs", [])}
    for bundle in bundles:
        tenant = bundle["metadata"]["name"]
        ns = "tenant-" + tenant
        declared = expected_claims(bundle)
        pvcs = {p["metadata"]["name"]: p for p in snapshot.get("pvcs", {}).get(tenant, [])}
        pods = snapshot.get("pods", {}).get(tenant, [])
        usage = {name: 0 for name in declared}
        for pod in pods:
            for volume in pod.get("spec", {}).get("volumes", []):
                claim = (volume.get("persistentVolumeClaim") or {}).get("claimName", "")
                if not (claim.startswith("ws-") or claim.startswith("skills-")):
                    continue
                if claim.endswith("-synelia"):
                    add("FAIL", "LEGACY_MOUNT", ns, claim, pod["metadata"]["name"])
                if claim not in declared:
                    add("FAIL", "UNDECLARED_WORKSPACE_MOUNT", ns, claim,
                        pod["metadata"]["name"])
                else:
                    usage[claim] += 1
                    if declared[claim]["mode"] == "reference":
                        for container in (pod["spec"].get("containers") or []):
                            for mount in container.get("volumeMounts") or []:
                                if mount.get("name") == volume["name"] and not mount.get("readOnly", False):
                                    add("FAIL", "REFERENCE_WRITABLE", ns, claim,
                                        pod["metadata"]["name"])
        for name, scope in declared.items():
            pvc = pvcs.get(name)
            entry = {"namespace": ns, "claim": name, "uid": None,
                     "pv": None, "consumers": usage[name], "phase": "Missing"}
            report["claims"].append(entry)
            if pvc is None:
                add("FAIL", "MISSING_CANONICAL_PVC", ns, name)
                continue
            metadata = pvc["metadata"]
            entry["uid"] = metadata.get("uid")
            entry["phase"] = pvc.get("status", {}).get("phase", "Unknown")
            entry["pv"] = pvc.get("spec", {}).get("volumeName")
            labels = metadata.get("labels") or {}
            expected_labels = {
                LABEL + "tenant-name": tenant,
                LABEL + "tenant-id": bundle["spec"]["tenantId"],
                LABEL + "workspace-domain": scope["domain"],
                LABEL + "workspace-scope": scope["scope"],
                LABEL + "workspace-key": scope["key"],
                LABEL + "workspace-mode": scope["mode"],
                LABEL + "workspace-retention": "Retain",
            }
            for key, value in expected_labels.items():
                if labels.get(key) != value:
                    add("FAIL", "WORKSPACE_OWNERSHIP_MISMATCH", ns, name, key)
            if entry["phase"] != "Bound":
                add("FAIL", "PVC_NOT_BOUND", ns, name, entry["phase"])
            if pvc.get("spec", {}).get("storageClassName") != STORAGE_CLASS:
                add("FAIL", "PVC_WRONG_CLASS", ns, name)
            if "ReadWriteMany" not in pvc.get("spec", {}).get("accessModes", []):
                add("FAIL", "PVC_NOT_RWX", ns, name)
            pv = pvs.get(entry["pv"])
            if not pv:
                add("FAIL", "MISSING_PV", ns, name)
                continue
            ref = pv.get("spec", {}).get("claimRef") or {}
            if (not entry["uid"] or ref.get("uid") != entry["uid"]
                    or ref.get("name") != name or ref.get("namespace") != ns):
                add("FAIL", "PV_CLAIMREF_MISMATCH", ns, name)
            if pv.get("spec", {}).get("persistentVolumeReclaimPolicy") != "Retain":
                add("FAIL", "PV_NOT_RETAIN", ns, name)
            if pv.get("spec", {}).get("storageClassName") != STORAGE_CLASS:
                add("FAIL", "PV_WRONG_CLASS", ns, name)
            if not pv.get("spec", {}).get("csi", {}).get("driver"):
                add("FAIL", "PV_NOT_CSI", ns, name)
            if not usage[name]:
                add("PENDING", "NO_OBSERVED_CONSUMER", ns, name)
        for name in pvcs:
            if name.endswith("-synelia") and (name.startswith("ws-") or name.startswith("skills-")):
                add("INFO", "LEGACY_CLAIM_RETAINED_FOR_ROLLBACK", ns, name)
    if not bundles:
        add("FAIL", "NO_TENANT_MANIFESTS", "", "")
    report["status"] = "FAIL" if any(x["level"] == "FAIL" for x in findings) else "PENDING"
    return report


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tenant", action="append", choices=TENANTS,
                        help="Tenant(s) to inspect; default hAIrem + Indiba")
    parser.add_argument("--snapshot", type=Path,
                        help="Offline kubectl JSON inventory; without this, GET live resources")
    args = parser.parse_args(argv)
    tenants = args.tenant or list(TENANTS)
    bundles = []
    for tenant in tenants:
        path = ROOT / "apps/60-services/txo-fabric/tenants" / tenant / "tenantbundle.yaml"
        with path.open(encoding="utf-8") as source:
            bundles.append(yaml.safe_load(source))
    try:
        inventory = (json.loads(args.snapshot.read_text(encoding="utf-8"))
                     if args.snapshot else live_snapshot(tenants))
        result = evaluate(bundles, inventory)
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as exc:
        print(json.dumps({"status": "FAIL", "error": str(exc)}, indent=2))
        return 2
    print(json.dumps(result, indent=2, sort_keys=True))
    # PVC/PV binding and Pod mounts never attest byte integrity, access rights,
    # multi-writer behavior or two real reconcile loops.
    return 2 if result["status"] == "FAIL" else 3


if __name__ == "__main__":
    sys.exit(main())
