"""#4193 source safety guard for a manually gated official UMI Synelia CSI.

This proves manifest invariants only; physical TrueNAS 27 / legacy migration
remain separate gates. Run: python3 -m unittest discover -s
scripts/validation/tests -p 'test_synelia_csi_source.py' -v
"""

from __future__ import annotations

from copy import deepcopy
from pathlib import Path
import unittest

import yaml


ROOT = Path(__file__).resolve().parents[3]
BASE = "apps/01-storage/synelia-csi/umi"
APP = "argocd/overlays/dev/apps/synelia-csi-umi.yaml"


def read_one(path: str) -> dict:
    with (ROOT / path).open(encoding="utf-8") as handle:
        return yaml.safe_load(handle)


def verify_classes(values: dict) -> None:
    classes = values["storageClasses"]
    names = [sc["name"] for sc in classes]
    expected = {
        "synelia-iscsi-delete",
        "synelia-iscsi-retain",
        "synelia-nfs-delete",
        "synelia-nfs-retain",
    }
    if len(names) != len(set(names)) or set(names) != expected:
        raise AssertionError(f"Unexpected classes or class collision: {names}")
    for sc in classes:
        name = sc["name"]
        if sc.get("isDefault") is not False:
            raise AssertionError(f"Default class not permitted: {name}")
        if sc["reclaimPolicy"] != name.rsplit("-", 1)[-1].capitalize():
            raise AssertionError(f"Wrong reclaim policy: {name}")
        if sc["parameters"].get("protocol") not in ("nfs", "iscsi"):
            raise AssertionError(f"Unexpected protocol: {name}")
        if sc["parameters"]["pool"] != "tank":
            raise AssertionError(f"Unexpected pool: {name}")
        parent = sc["parameters"]["datasetPath"]
        if not parent.startswith("k8s/vixens/synelia-umi-test/"):
            raise AssertionError(f"Unsafe parent: {parent}")
        if any(old in parent for old in ("nfs/vols", "iscsi/vols")):
            raise AssertionError(f"Legacy dataset collision: {parent}")
        if parent.endswith("/" + sc["parameters"]["protocol"]) is False:
            raise AssertionError(f"Protocol path mismatch: {name}")


class SyneliaCSISourceTests(unittest.TestCase):
    def setUp(self):
        self.values = read_one(f"{BASE}/values.yaml")
        self.app = read_one(APP)

    def test_manual_umi_only_pinned_release_and_identity(self):
        self.assertEqual(self.app["metadata"]["name"], "synelia-csi-umi")
        self.assertEqual(self.app["spec"]["destination"]["namespace"], "synelia-csi")
        self.assertNotIn("automated", self.app["spec"]["syncPolicy"])
        sources = self.app["spec"]["sources"]
        self.assertEqual(sources[0]["chart"], "truenas-csi")
        self.assertEqual(sources[0]["targetRevision"], "1.4.0")
        self.assertEqual(sources[0]["helm"]["releaseName"], "synelia-csi")
        self.assertEqual(
            sources[0]["helm"]["valueFiles"], [f"$values/{BASE}/values.yaml"]
        )
        self.assertEqual(sources[1]["ref"], "values")
        self.assertEqual(sources[2]["path"], BASE)
        self.assertEqual(self.values["driverName"], "synelia.csi.truenas.io")
        self.assertEqual(self.values["image"]["driver"]["tag"], "v1.4.0")
        self.assertEqual(self.values["kubeletDir"], "/var/lib/kubelet")
        self.assertEqual(self.values["volumeSnapshotClass"]["enabled"], False)
        self.assertNotIn(
            "apps/synelia-csi-umi.yaml",
            read_one("argocd/overlays/prod/kustomization.yaml")["resources"],
        )
        self.assertIn(
            "apps/synelia-csi-umi.yaml",
            read_one("argocd/overlays/dev/kustomization.yaml")["resources"],
        )

    def test_classes_isolated_not_default_and_reclaim_compatible(self):
        verify_classes(self.values)
        # A regression would fail if someone reused a legacy class name.
        legacy_nfs = read_one(
            "apps/01-storage/truenas-csi/values/nfs-umi.yaml"
        )["storageClasses"]
        legacy_iscsi = read_one(
            "apps/01-storage/truenas-csi/values/iscsi-umi.yaml"
        )["storageClasses"]
        legacy_names = {s["name"] for s in legacy_nfs + legacy_iscsi}
        self.assertFalse(
            legacy_names.intersection({s["name"] for s in self.values["storageClasses"]})
        )

    def test_safety_regressions_fail_closed(self):
        mutated = deepcopy(self.values)
        mutated["storageClasses"][0]["isDefault"] = True
        with self.assertRaises(AssertionError):
            verify_classes(mutated)
        mutated = deepcopy(self.values)
        mutated["storageClasses"][0]["parameters"]["datasetPath"] = (
            "k8s/vixens/nfs/vols"
        )
        with self.assertRaises(AssertionError):
            verify_classes(mutated)
        mutated = deepcopy(self.values)
        mutated["storageClasses"][0]["reclaimPolicy"] = "Delete"
        with self.assertRaises(AssertionError):
            verify_classes(mutated)

    def test_tls_and_openbao_infra_secret_contract(self):
        cfg = self.values["truenas"]
        self.assertTrue(cfg["url"].startswith("wss://nas.truxonline.com/"))
        self.assertIs(cfg["insecureSkipTLS"], False)
        self.assertEqual(cfg["existingSecret"], "synelia-csi-credentials")
        self.assertEqual(cfg["existingSecretKey"], "api-key")
        self.assertNotIn("apiKey", cfg)
        self.assertEqual(cfg["iscsiPortal"], "192.168.111.69:3260")
        eso = read_one(f"{BASE}/externalsecret.yaml")
        self.assertEqual(eso["metadata"]["namespace"], "synelia-csi")
        self.assertEqual(
            eso["spec"]["secretStoreRef"],
            {"name": "openbao", "kind": "ClusterSecretStore"},
        )
        self.assertEqual(eso["spec"]["target"]["deletionPolicy"], "Retain")
        self.assertEqual(
            eso["spec"]["data"][0],
            {
                "secretKey": "api-key",
                "remoteRef": {
                    "key": "vixens/dev/apps/01-storage/synelia-csi/api",
                    "property": "api-key",
                },
            },
        )

    def test_cilium_policy_is_narrow_and_hostnetwork_limit_documented(self):
        docs = [
            x for x in yaml.safe_load_all(
                (ROOT / BASE / "cilium-networkpolicy.yaml").read_text(encoding="utf-8")
            ) if x
        ]
        self.assertEqual(len(docs), 2)
        self.assertTrue(all(x["kind"] == "CiliumNetworkPolicy" for x in docs))
        self.assertTrue(
            all(x["metadata"]["namespace"] == "synelia-csi" for x in docs)
        )
        self.assertEqual(
            docs[0]["spec"]["endpointSelector"]["matchLabels"]["app"],
            "synelia-csi-controller",
        )
        egresses = docs[0]["spec"]["egress"] + docs[1]["spec"]["egress"]
        cidrs = [
            cidr for rule in egresses for cidr in rule.get("toCIDR", [])
        ]
        self.assertEqual(set(cidrs), {"192.168.111.69/32"})
        ports = {
            item["port"]
            for rule in egresses
            for group in rule.get("toPorts", [])
            for item in group["ports"]
        }
        self.assertTrue({"443", "2049", "3260", "53"}.issubset(ports))


if __name__ == "__main__":
    unittest.main()
