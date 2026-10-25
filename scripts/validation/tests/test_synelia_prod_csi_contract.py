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
EXPECTED = {
    "synelia-nfs-retain",
    "synelia-nfs-delete",
    "synelia-iscsi-retain",
    "synelia-iscsi-stateful-retain",
    "synelia-iscsi-delete",
}


class SyneliaProdCSIContractTest(unittest.TestCase):
    def test_storage_class_contract(self) -> None:
        values = yaml.safe_load(VALUES.read_text())
        classes = values["storageClasses"]
        self.assertEqual({item["name"] for item in classes}, EXPECTED)
        self.assertTrue(all(item["isDefault"] is False for item in classes))
        for item in classes:
            name = item["name"]
            self.assertEqual(item["reclaimPolicy"], name.rsplit("-", 1)[1].capitalize())
            parent = item["parameters"]["datasetPath"]
            self.assertNotIn("/vols", parent)
            self.assertNotIn("prod-staging", parent)
        stateful = next(item for item in classes if item["name"] == "synelia-iscsi-stateful-retain")
        self.assertEqual(stateful["parameters"]["datasetPath"], "k8s/vixens/synelia/stateful/iscsi")
        self.assertEqual(stateful["volumeBindingMode"], "WaitForFirstConsumer")

    def test_activation_and_credential_boundary(self) -> None:
        app = yaml.safe_load(APP.read_text())
        self.assertTrue(app["spec"]["syncPolicy"]["automated"]["selfHeal"])
        self.assertFalse(app["spec"]["syncPolicy"]["automated"]["prune"])
        self.assertIn("apps/synelia-prod-csi.yaml", ROOT_KUSTOMIZATION.read_text())
        eso = yaml.safe_load(ESO.read_text())
        self.assertEqual(eso["kind"], "ExternalSecret")
        self.assertEqual(eso["spec"]["data"][0]["remoteRef"]["key"], "vixens/prod/apps/01-storage/synelia-csi/api")


if __name__ == "__main__":
    unittest.main()
