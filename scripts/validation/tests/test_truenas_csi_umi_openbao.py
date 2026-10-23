"""Regression guard for #4185: UMI CSI secrets use canonical OpenBao, never openbao-umi.

This source-level guard complements the Kustomize build in validate.yaml.
It does not validate live OpenBao credentials or existing iSCSI PV portals.
"""

from __future__ import annotations

from copy import deepcopy
from pathlib import Path
import unittest

import yaml


ROOT = Path(__file__).resolve().parents[3]
CSI_BASE = "apps/01-storage/truenas-csi/openbao/base/truenas-csi-externalsecrets.yaml"
CSI_UMI = "apps/01-storage/truenas-csi/openbao/overlays/umi/kustomization.yaml"
INFRA_STORE = "apps/00-infra/openbao/base/cluster-secret-store.yaml"


def read_yaml(path: str):
    return list(yaml.safe_load_all((ROOT / path).read_text(encoding="utf-8")))


def effective_external_secrets(overlay: dict) -> dict[str, dict]:
    """Apply the narrowly scoped JSON6902 overrides from the UMI source overlay."""
    external_secrets = {
        obj["metadata"]["name"]: deepcopy(obj)
        for obj in read_yaml(CSI_BASE)
        if obj and obj["kind"] == "ExternalSecret"
    }
    for item in overlay["patches"]:
        target = item["target"]
        if target["kind"] != "ExternalSecret":
            raise AssertionError("UMI CSI overlay patches an unexpected kind")
        name = target["name"]
        if name not in external_secrets:
            raise AssertionError(f"UMI CSI overlay patches unknown ExternalSecret: {name}")
        resource = external_secrets[name]
        for operation in yaml.safe_load(item["patch"]):
            if operation["op"] != "replace":
                raise AssertionError("unexpected UMI CSI patch operation")
            nodes = operation["path"].strip("/").split("/")
            current = resource
            for node in nodes[:-1]:
                current = current[int(node)] if isinstance(current, list) else current[node]
            last = nodes[-1]
            if isinstance(current, list):
                current[int(last)] = operation["value"]
            else:
                if last not in current:
                    raise AssertionError(f"cannot replace missing field {last}")
                current[last] = operation["value"]
    return external_secrets


class UMICSIOpenBaoSourceTests(unittest.TestCase):
    def setUp(self):
        self.overlay = read_yaml(CSI_UMI)[0]

    def test_dev_iscsi_and_nfs_use_existing_canonical_store_and_dev_only_paths(self):
        available = {
            item["metadata"]["name"]
            for item in read_yaml(INFRA_STORE)
            if item and item.get("kind") == "ClusterSecretStore"
        }
        self.assertEqual(available, {"openbao"})

        rendered = effective_external_secrets(self.overlay)
        self.assertEqual(set(rendered), {
            "truenas-csi-iscsi-driver-config",
            "truenas-csi-nfs-driver-config",
        })
        for protocol in ("iscsi", "nfs"):
            resource = rendered[f"truenas-csi-{protocol}-driver-config"]
            self.assertEqual(resource["metadata"]["namespace"], "truenas-csi")
            self.assertEqual(
                resource["spec"]["secretStoreRef"],
                {"kind": "ClusterSecretStore", "name": "openbao"},
            )
            self.assertIn(resource["spec"]["secretStoreRef"]["name"], available)
            self.assertEqual(
                resource["spec"]["data"][0]["remoteRef"],
                {
                    "key": f"vixens/dev/apps/01-storage/truenas-csi/{protocol}",
                    "property": "driver-config-file.yaml",
                },
            )
            self.assertEqual(
                resource["spec"]["target"]["name"],
                f"truenas-csi-{protocol}-driver-config",
            )

    def test_legacy_store_override_would_fail_contract(self):
        mutated = deepcopy(self.overlay)
        mutated["patches"][0]["patch"] += (
            "\n- op: replace\n  path: /spec/secretStoreRef/name"
            "\n  value: openbao-umi"
        )
        bad = effective_external_secrets(mutated)
        self.assertNotEqual(
            bad["truenas-csi-iscsi-driver-config"]["spec"]["secretStoreRef"]["name"],
            "openbao",
        )

    def test_prod_application_stays_on_prod_stable_and_separate_overlay(self):
        prod = read_yaml("argocd/overlays/prod/apps/truenas-csi-secrets.yaml")[0]
        dev = read_yaml("argocd/overlays/dev/apps/truenas-csi-secrets-umi.yaml")[0]
        self.assertEqual(prod["spec"]["source"]["targetRevision"], "prod-stable")
        self.assertEqual(
            prod["spec"]["source"]["path"],
            "apps/01-storage/truenas-csi/bootstrap/overlays/prod",
        )
        self.assertEqual(dev["spec"]["source"]["targetRevision"], "main")
        self.assertEqual(
            dev["spec"]["source"]["path"],
            "apps/01-storage/truenas-csi/openbao/overlays/umi",
        )


if __name__ == "__main__":
    unittest.main()
