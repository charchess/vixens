"""#4185: fail-closed UMI-only Synelia recovered Hermes PV cutover.

This is a source contract; actual bound PVC UID, PV identity, storage mount,
sampled content acceptance and paused live Argo autosync must be checked on UMI
prior to an authorized manual application sync.
"""

from copy import deepcopy
from pathlib import Path
import unittest
import yaml


ROOT = Path(__file__).resolve().parents[3]
BASE = ROOT / "apps/60-services/hermes/base"
DEV = ROOT / "apps/60-services/hermes/overlays/dev"
TARGET_PVC = "hermes-data-synelia-recovery"
EXPECTED_PV = "pvc-66d68670-a146-4ae3-87f6-421860f4554e"


def read(path: Path) -> dict:
    return yaml.safe_load(path.read_text(encoding="utf-8"))


def assert_safe_recovered_claim(claim: dict) -> None:
    if claim.get("kind") != "PersistentVolumeClaim":
        raise AssertionError("Expected PVC, not another resource")
    md = claim.get("metadata", {})
    spec = claim.get("spec", {})
    if md.get("name") != TARGET_PVC or md.get("namespace") != "services":
        raise AssertionError("Recovered PVC identity must be exact")
    opts = set(md.get("annotations", {}).get(
        "argocd.argoproj.io/sync-options", ""
    ).split(","))
    if not {"Prune=false", "Delete=false"}.issubset(opts):
        raise AssertionError("Retained recovered PVC cannot be Argo pruned/deleted")
    if spec.get("accessModes") != ["ReadWriteOnce"]:
        raise AssertionError("Recovered PV requires RWO")
    if spec.get("storageClassName") != "synelia-iscsi-stateful-retain":
        raise AssertionError("Recovered class must be the durable Synelia Retain class")
    if spec.get("volumeName") != EXPECTED_PV:
        raise AssertionError("PVC MUST explicitly prebind to existing recovered PV")
    if spec.get("resources", {}).get("requests", {}).get("storage") != "80Gi":
        raise AssertionError("Recovery PVC storage request drift")


class HermesRecoveryCutoverTests(unittest.TestCase):
    def setUp(self):
        self.claim = read(DEV / "pvc-synelia-recovery.yaml")
        self.overlay = read(DEV / "kustomization.yaml")

    def test_recovery_pvc_is_prebound_and_protected(self):
        assert_safe_recovered_claim(self.claim)

    def test_recovery_claim_fail_closed(self):
        mutations = (
            ("volumeName-missing", lambda x: x["spec"].pop("volumeName")),
            ("volumeName-changed", lambda x: x["spec"].update({"volumeName": "pvc-other"})),
            ("class-test", lambda x: x["spec"].update({"storageClassName": "synelia-iscsi-delete"})),
            ("mode-rwx", lambda x: x["spec"].update({"accessModes": ["ReadWriteMany"]})),
            ("prune-enabled", lambda x: x["metadata"]["annotations"].update({"argocd.argoproj.io/sync-options": "Prune=true"})),
            ("new-namespace", lambda x: x["metadata"].update({"namespace": "default"})),
        )
        for name, mutate in mutations:
            with self.subTest(name=name):
                claim = deepcopy(self.claim)
                mutate(claim)
                with self.assertRaises(AssertionError):
                    assert_safe_recovered_claim(claim)

    def test_dev_overlay_changes_only_data_volume_and_keeps_legacy_pvc(self):
        self.assertEqual(self.overlay["resources"], ["../../base", "pvc-synelia-recovery.yaml"])
        self.assertEqual(self.overlay["namespace"], "services")
        self.assertEqual(self.overlay["commonLabels"], {"environment": "dev"})
        patches = self.overlay.get("patches", [])
        self.assertEqual(len(patches), 1)
        self.assertEqual(patches[0]["target"], {
            "group": "apps", "version": "v1", "kind": "Deployment", "name": "hermes"
        })
        patch = yaml.safe_load(patches[0]["patch"])
        self.assertEqual(patch, {
            "apiVersion": "apps/v1",
            "kind": "Deployment",
            "metadata": {"name": "hermes", "namespace": "services"},
            "spec": {
                "template": {
                    "spec": {
                        "volumes": [{
                            "name": "data",
                            "persistentVolumeClaim": {"claimName": TARGET_PVC},
                        }]
                    }
                }
            },
        })
        base = read(BASE / "deployment.yaml")
        self.assertEqual(
            [v["persistentVolumeClaim"]["claimName"]
             for v in base["spec"]["template"]["spec"]["volumes"]
             if v["name"] == "data"],
            ["hermes-data-truenas"],
        )
        old = read(BASE / "pvc.yaml")
        self.assertEqual(old["metadata"]["name"], "hermes-data-truenas")
        self.assertEqual(old["spec"]["storageClassName"], "truenas-iscsi-retain")


if __name__ == "__main__":
    unittest.main()
