#!/usr/bin/env python3
"""#4185 UMI TrueNAS iSCSI PV re-IP: offline, fail-closed review preparation.

Accepts a read-only Kubernetes PV JSON export; performs NO Kubernetes/TrueNAS
calls or mutations. Candidate manifests are NOT installable migration steps.
"""

from __future__ import annotations

import argparse
from copy import deepcopy
import json
from pathlib import Path
import re
import sys

import yaml

OLD_PORTAL = "192.168.200.244:3260"
NEW_PORTAL = "nas.truxonline.com:3260"
DRIVER = "org.democratic-csi.truenas-iscsi"
# Exact four UMI PVs observed by the operator on 2026-10-24. No discovery guesses.
EXPECTED = {
    "pvc-5cb237dc-4a12-49ec-80c9-29477d89fa77": ("services", "hermes-data-truenas", "Bound", "Retain"),
    "pvc-71395a3d-7d40-4303-bff3-0f3c23849fbf": ("hindsight", "data-hindsight-postgresql-0", "Bound", "Retain"),
    "pvc-c09f3044-a015-4be7-8f9d-49698dbc2fe1": ("homeassistant", "homeassistant-config", "Released", "Retain"),
    "pvc-c5b1130c-d400-4f45-9cc8-bd1a25cf8e43": ("hindsight", "hindsight-api-model-cache", "Bound", "Delete"),
}


class CutoverError(ValueError):
    """Nonconforming source is never allowed into a candidate manifest."""


def require(ok: bool, message: str) -> None:
    if not ok:
        raise CutoverError(message)


def prepare(pv_list: dict) -> list[dict]:
    """Prepare four offline review-only PV candidates without touching a cluster."""
    require(isinstance(pv_list, dict) and isinstance(pv_list.get("items"), list),
            "expected kubectl get pv -o json object with an items array")
    existing = {}
    for pv in pv_list["items"]:
        require(isinstance(pv, dict), "invalid PV item")
        name = (pv.get("metadata") or {}).get("name")
        require(isinstance(name, str) and name, "PV missing metadata.name")
        require(name not in existing, f"duplicate PV {name}")
        existing[name] = pv
        portal = (((pv.get("spec") or {}).get("csi") or {}).get("volumeAttributes") or {}).get("portal")
        require(not (portal == OLD_PORTAL and name not in EXPECTED),
                f"unexpected old-portal PV {name}; inventory needs review")

    require(all(name in existing for name in EXPECTED),
            "required UMI PV absent from inventory: " + ", ".join(sorted(set(EXPECTED) - set(existing))))
    output = []
    for name, (namespace, claim, phase, reclaim) in EXPECTED.items():
        pv = existing[name]
        spec = pv.get("spec") or {}
        csi = spec.get("csi") or {}
        attributes = csi.get("volumeAttributes") or {}
        ref = spec.get("claimRef") or {}
        actual_phase = (pv.get("status") or {}).get("phase")
        require(actual_phase == phase, f"{name}: expected {phase} status, got {actual_phase!r}")
        require(spec.get("persistentVolumeReclaimPolicy") == reclaim,
                f"{name}: reclaim policy differs from the recorded inventory; reassess")
        require(ref.get("name") == claim and ref.get("namespace") == namespace,
                f"{name}: claimRef identity changed; refuse cross-tenant/claim rebind")
        require(isinstance(ref.get("uid"), str) and ref["uid"],
                f"{name}: missing source claimRef UID; refuse uncertain rebinding")
        require(csi.get("driver") == DRIVER, f"{name}: unexpected CSI driver")
        require(csi.get("volumeHandle") == name, f"{name}: volumeHandle mismatch")
        require(attributes.get("portal") == OLD_PORTAL,
                f"{name}: expected original portal {OLD_PORTAL}; source drift")
        target = f"iqn.2005-10.org.freenas.ctl:csi-{name}-umi"
        require(attributes.get("iqn") == target, f"{name}: target IQN mismatch")
        require(str(attributes.get("lun")) == "0", f"{name}: unexpected LUN")
        require(spec.get("accessModes") == ["ReadWriteOnce"], f"{name}: accessModes changed")
        require(isinstance((spec.get("capacity") or {}).get("storage"), str),
                f"{name}: storage capacity missing")
        require(isinstance(spec.get("storageClassName"), str) and spec["storageClassName"],
                f"{name}: storageClassName missing")
        # Refuse to print any possibly credential-bearing inline CSI attributes.
        require(not any(re.search(r"password|token|credential|secret|private.key", k, re.I)
                        for k in attributes), f"{name}: sensitive CSI attribute key present")
        candidate = {
            "apiVersion": "v1",
            "kind": "PersistentVolume",
            "metadata": {"name": name},
            "spec": deepcopy(spec),
        }
        candidate["spec"]["csi"]["volumeAttributes"]["portal"] = NEW_PORTAL
        # The cache is Delete in live API. Candidate must be Retain to prevent
        # accidental backend deletion during a *separately authorized* rebind.
        candidate["spec"]["persistentVolumeReclaimPolicy"] = "Retain"
        # The source object's UID, resourceVersion, status, finalizers and
        # controller annotations are deliberately NOT included; their treatment
        # is a separate, audited Kubernetes cutover problem.
        output.append(candidate)
    return output


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description="OFFLINE REVIEW ONLY: #4185 UMI PV portal re-IP")
    p.add_argument("--pv-json", type=Path, required=True,
                   help="local file from 'kubectl get pv -o json'; do not commit it")
    p.add_argument("--emit-review-yaml", action="store_true",
                   help="print candidate PV YAML to stdout (NEVER automatically apply)")
    args = p.parse_args(argv)
    try:
        candidates = prepare(json.loads(args.pv_json.read_text(encoding="utf-8")))
    except (OSError, json.JSONDecodeError, CutoverError) as exc:
        print(f"FAIL CLOSED: {exc}", file=sys.stderr)
        return 2
    if args.emit_review_yaml:
        print("# REVIEW ONLY — NOT an installation or in-place PV patch procedure.")
        print("# PV sources are immutable; live migration requires an approved, staged rebind.")
        print(yaml.safe_dump_all(candidates, sort_keys=False, explicit_start=True))
    else:
        print(f"PASS: {len(candidates)} exact UMI PV identities matched. No cluster calls or writes.")
        print(f"Review-only portal: {OLD_PORTAL} -> {NEW_PORTAL}")
        print("HOLD: 3 active VolumeAttachments last observed; confirm detached before cutover.")
        print("HOLD: verify all TrueNAS IQN/LUN/zvol mapping and protected backups first.")
        print("HOLD: kernel ext4 emergency_ro on Hermes and PostgreSQL requires separate recovery.")
        print("HOLD: original PVC UID/claimRef, controller annotations and finalizers require review.")
        print("NOTE: cache candidate reclaimPolicy=Retain; live cache remains Delete.")
        print("No kubectl apply/delete/patch, iscsiadm login/logout or filesystem operation performed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
