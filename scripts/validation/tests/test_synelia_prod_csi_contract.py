"""Production Synelia CSI contract: durable names, parallel ownership, no default."""
from __future__ import annotations

from copy import deepcopy
import importlib.util
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
SPEC = importlib.util.spec_from_file_location(
    "check_synelia_prod_render",
    ROOT / "scripts/validation/check_synelia_prod_render.py",
)
CHECKER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CHECKER)

EXPECTED = {
    "storage-rwx-retain",
    "storage-rwx-delete",
    "storage-rwo-retain",
    "storage-rwo-delete",
    "storage-rwo-xfs-retain",
    "storage-rwo-xfs-delete",
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
                expected_fs = "xfs" if name.startswith("storage-rwo-xfs-") else "ext4"
                self.assertEqual(item["parameters"]["fsType"], expected_fs)
                self.assertEqual(item["volumeBindingMode"], "WaitForFirstConsumer")
                self.assertEqual(parent, "k8s/vixens/iscsi")
            else:
                self.assertEqual(item["parameters"]["protocol"], "nfs")
                self.assertIn("nfsvers=4.1", item["mountOptions"])
                self.assertEqual(item["volumeBindingMode"], "Immediate")
                self.assertEqual(parent, "k8s/vixens/nfs")

    def test_structured_helm_render_contract_rejects_foreign_provisioners(self) -> None:
        driver = {"kind": "CSIDriver", "metadata": {"name": CHECKER.EXPECTED_DRIVER}}
        classes = []
        for name in sorted(EXPECTED):
            rwo = name.startswith("storage-rwo-")
            params = {
                "protocol": "iscsi" if rwo else "nfs",
                "datasetPath": "k8s/vixens/iscsi" if rwo else "k8s/vixens/nfs",
            }
            if rwo:
                params["fsType"] = "xfs" if "-xfs-" in name else "ext4"
            classes.append({
                "kind": "StorageClass",
                "metadata": {"name": name, "annotations": {
                    "storageclass.kubernetes.io/is-default-class": "false",
                }},
                "provisioner": CHECKER.EXPECTED_DRIVER,
                "reclaimPolicy": name.rsplit("-", 1)[-1].capitalize(),
                "parameters": params,
                "volumeBindingMode": "WaitForFirstConsumer" if rwo else "Immediate",
                "mountOptions": [] if rwo else ["nfsvers=4.1"],
            })
        valid = [driver, *classes]
        CHECKER.assert_production_render(valid)

        mutations = {
            "foreign provisioner": lambda docs: docs[1].update(provisioner="retired.csi.example"),
            "missing driver": lambda docs: docs.pop(0),
            "duplicate class": lambda docs: docs.append(deepcopy(docs[1])),
            "unexpected class": lambda docs: docs[1]["metadata"].update(name="retired-class"),
            "default class": lambda docs: docs[1]["metadata"]["annotations"].update({
                "storageclass.kubernetes.io/is-default-class": "true",
            }),
            "missing non-default annotation": lambda docs: docs[1]["metadata"].pop("annotations"),
            "wrong reclaim": lambda docs: docs[1].update(reclaimPolicy="Retain"),
            "foreign class parent": lambda docs: docs[1]["parameters"].update(datasetPath="legacy/vols"),
            "injected Secret": lambda docs: docs.append({"kind": "Secret", "metadata": {"name": "credential"}}),
        }
        # The chosen class may already use Retain; choose the opposite value
        # to keep the reclaim-policy case a meaningful negative test.
        for label, mutate in mutations.items():
            docs = deepcopy(valid)
            if label == "wrong reclaim":
                docs[1]["reclaimPolicy"] = (
                    "Delete" if docs[1]["reclaimPolicy"] == "Retain" else "Retain"
                )
            else:
                mutate(docs)
            with self.subTest(label=label), self.assertRaises(AssertionError):
                CHECKER.assert_production_render(docs)

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
