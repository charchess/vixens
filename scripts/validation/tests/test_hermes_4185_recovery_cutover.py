"""Contract for the active Hermes Synelia recovery claim (#4185)."""
from copy import deepcopy
from pathlib import Path
import unittest
import yaml

ROOT = Path(__file__).resolve().parents[3]
BASE = ROOT / "apps/60-services/hermes/base"
DEV = ROOT / "apps/60-services/hermes/overlays/dev"
TARGET_PVC = "hermes-data-synelia-recovery"


def read(path): return yaml.safe_load(path.read_text(encoding="utf-8"))

def assert_active_claim(claim):
    if claim.get("kind") != "PersistentVolumeClaim": raise AssertionError("expected PVC")
    md, spec = claim.get("metadata", {}), claim.get("spec", {})
    if md.get("name") != TARGET_PVC or md.get("namespace") != "services": raise AssertionError("active claim identity drift")
    if spec.get("accessModes") != ["ReadWriteOnce"]: raise AssertionError("active Hermes claim must be RWO")
    if spec.get("storageClassName") != "synelia-iscsi-stateful-retain": raise AssertionError("active Hermes claim must retain Synelia state")
    if spec.get("resources", {}).get("requests", {}).get("storage") != "80Gi": raise AssertionError("active Hermes claim size drift")

class HermesRecoveryCutoverTests(unittest.TestCase):
    def setUp(self):
        self.claim = read(BASE / "pvc.yaml")
        self.deployment = read(BASE / "deployment.yaml")
        self.overlay = read(DEV / "kustomization.yaml")
    def test_active_claim_is_retained_synelia_contract(self): assert_active_claim(self.claim)
    def test_active_claim_fails_closed_on_class_or_identity_drift(self):
        for mutate in (lambda x: x["metadata"].update({"name":"new-empty-pvc"}), lambda x: x["spec"].update({"storageClassName":"storage-rwo-delete"}), lambda x: x["spec"].update({"accessModes":["ReadWriteMany"]})):
            with self.assertRaises(AssertionError):
                x=deepcopy(self.claim); mutate(x); assert_active_claim(x)
    def test_rendered_base_and_dev_select_only_active_claim(self):
        claims=[v["persistentVolumeClaim"]["claimName"] for v in self.deployment["spec"]["template"]["spec"]["volumes"] if v["name"]=="data"]
        self.assertEqual(claims,[TARGET_PVC])
        self.assertEqual(self.overlay["resources"],["../../base"])
        self.assertNotIn("pvc-synelia-recovery.yaml", self.overlay.get("resources",[]))

if __name__ == "__main__": unittest.main()
