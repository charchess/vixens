"""Production Synelia CSI contract: durable names, parallel ownership, no default."""
from __future__ import annotations

from pathlib import Path
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
VALUES = ROOT / "apps/01-storage/synelia-csi/prod/values.yaml"
APP = ROOT / "argocd/overlays/prod/apps/synelia-prod-csi.yaml"
ROOT_KUSTOMIZATION = ROOT / "argocd/overlays/prod/kustomization.yaml"
ESO = ROOT / "apps/01-storage/synelia-csi/prod/externalsecret.yaml"
NAMESPACE = ROOT / "apps/01-storage/synelia-csi/prod/namespace.yaml"
NETWORK_POLICY = ROOT / "apps/01-storage/synelia-csi/prod/networkpolicy.yaml"
EXPECTED = {
    "storage-rwx-retain",
    "storage-rwx-delete",
    "storage-rwo-retain",
    "storage-rwo-delete",
}


class SyneliaProdCSIContractTest(unittest.TestCase):
    def test_storage_class_contract(self) -> None:
        values = yaml.safe_load(VALUES.read_text())
        classes = values["storageClasses"]
        self.assertEqual({item["name"] for item in classes}, EXPECTED)
        self.assertTrue(all(item["isDefault"] is False for item in classes))
        for item in classes:
            name = item["name"]
            self.assertTrue(name.startswith("storage-rw"))
            self.assertEqual(item["reclaimPolicy"], name.rsplit("-", 1)[1].capitalize())
            parent = item["parameters"]["datasetPath"]
            self.assertNotIn("/vols", parent)
            self.assertNotIn("prod-staging", parent)
            if name.startswith("storage-rwo-"):
                self.assertEqual(item["parameters"]["protocol"], "iscsi")
                self.assertEqual(item["parameters"]["fsType"], "ext4")
                self.assertEqual(item["volumeBindingMode"], "WaitForFirstConsumer")
                self.assertEqual(parent, "k8s/vixens/synelia-test/iscsi")
            else:
                self.assertEqual(item["parameters"]["protocol"], "nfs")
                self.assertIn("nfsvers=4.1", item["mountOptions"])
                self.assertEqual(item["volumeBindingMode"], "Immediate")
                self.assertEqual(parent, "k8s/vixens/synelia-test/nfs")

    def test_activation_and_credential_boundary(self) -> None:
        app = yaml.safe_load(APP.read_text())
        self.assertTrue(app["spec"]["syncPolicy"]["automated"]["selfHeal"])
        self.assertFalse(app["spec"]["syncPolicy"]["automated"]["prune"])
        self.assertIn("apps/synelia-prod-csi.yaml", ROOT_KUSTOMIZATION.read_text())
        eso = yaml.safe_load(ESO.read_text())
        self.assertEqual(eso["kind"], "ExternalSecret")
        self.assertEqual(eso["spec"]["data"][0]["remoteRef"]["key"], "vixens/prod/apps/01-storage/synelia-csi/api")
        values = yaml.safe_load(VALUES.read_text())
        self.assertEqual(values["truenas"]["url"], "wss://nas.truxonline.com:8443/api/current")
        policy = NETWORK_POLICY.read_text()
        self.assertIn('port: "8443"', policy)
        self.assertNotIn('port: "443"', policy)
        namespace = yaml.safe_load(NAMESPACE.read_text())
        labels = namespace["metadata"]["labels"]
        self.assertEqual(
            {name: labels[name] for name in (
                "pod-security.kubernetes.io/enforce",
                "pod-security.kubernetes.io/audit",
                "pod-security.kubernetes.io/warn",
            )},
            {
                "pod-security.kubernetes.io/enforce": "privileged",
                "pod-security.kubernetes.io/audit": "privileged",
                "pod-security.kubernetes.io/warn": "privileged",
            },
        )


if __name__ == "__main__":
    unittest.main()
