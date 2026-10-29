"""#4185: Hermes UMI must not auto-sync while an isolated recovery PVC is staged.

This is a temporary cutover safety gate. Change/remove this guard only in
the separately reviewed PR restoring Argo automation after runtime signoff.
"""

from copy import deepcopy
from pathlib import Path
import unittest

import yaml


ROOT = Path(__file__).resolve().parents[3]
APP = ROOT / "argocd/overlays/dev/apps/hermes-umi.yaml"
DEPLOYMENT = ROOT / "apps/60-services/hermes/base/deployment.yaml"
LEGACY_PVC = ROOT / "apps/60-services/hermes/base/pvc.yaml"


def safe_hermes_manual_gate(app: dict) -> None:
    if app.get("kind") != "Application":
        raise AssertionError("Expected Argo Application")
    if app.get("metadata", {}).get("name") != "hermes-umi":
        raise AssertionError("Unexpected application identity")
    spec = app.get("spec", {})
    source = spec.get("source", {})
    if source.get("targetRevision") != "main" or source.get("path") != "apps/60-services/hermes/overlays/dev":
        raise AssertionError("UMI dev source/branch changed")
    if spec.get("destination", {}).get("namespace") != "services":
        raise AssertionError("Unexpected destination namespace")
    sync_policy = spec.get("syncPolicy", {})
    if "automated" in sync_policy:
        raise AssertionError("Hermes must have no automated sync or prune during #4185")
    if "Prune=true" in sync_policy.get("syncOptions", []):
        raise AssertionError("Forced prune prohibited during recovery")
    if "Force=true" in sync_policy.get("syncOptions", []):
        raise AssertionError("Forced resource replacement prohibited during recovery")


class HermesManualGateTests(unittest.TestCase):
    def setUp(self):
        self.app = yaml.safe_load(APP.read_text(encoding="utf-8"))

    def test_no_auto_sync_until_recovery_gate(self):
        safe_hermes_manual_gate(self.app)
        self.assertEqual(self.app["metadata"]["name"], "hermes-umi")

    def test_fail_closed_on_auto_sync_and_scope_drift(self):
        mutations = (
            ("auto-prune", lambda app: app["spec"]["syncPolicy"].update({"automated": {"prune": True, "selfHeal": True}})),
            ("auto-empty", lambda app: app["spec"]["syncPolicy"].update({"automated": {}})),
            ("prod-path", lambda app: app["spec"]["source"].update({"path": "apps/60-services/hermes/overlays/prod"})),
            ("wrong-namespace", lambda app: app["spec"]["destination"].update({"namespace": "default"})),
            ("force", lambda app: app["spec"]["syncPolicy"]["syncOptions"].append("Force=true")),
        )
        for label, mutate in mutations:
            with self.subTest(label=label):
                app = deepcopy(self.app)
                mutate(app)
                with self.assertRaises(AssertionError):
                    safe_hermes_manual_gate(app)

    def test_manual_gate_preserves_active_synelia_claim(self):
        deployment = yaml.safe_load(DEPLOYMENT.read_text(encoding="utf-8"))
        claims = [
            volume["persistentVolumeClaim"]["claimName"]
            for volume in deployment["spec"]["template"]["spec"]["volumes"]
            if volume["name"] == "data"
        ]
        self.assertEqual(claims, ["hermes-data-synelia-recovery"])
        active = yaml.safe_load(LEGACY_PVC.read_text(encoding="utf-8"))
        self.assertEqual(active["metadata"]["name"], "hermes-data-synelia-recovery")
        self.assertEqual(active["spec"]["storageClassName"], "synelia-iscsi-stateful-retain")
        self.assertEqual(active["spec"]["resources"]["requests"]["storage"], "80Gi")


if __name__ == "__main__":
    unittest.main()
