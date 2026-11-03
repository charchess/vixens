"""Hermes UMI stateful source contract while the retained data gate is active.

This is an application-safety test, not proof of filesystem content, CSI
volume health, or permission to re-enable Argo automation. The two concrete
existing PVC identifiers below are *physical binding pins*, not the native
Fabric operator's storage policy.
"""

from copy import deepcopy
from pathlib import Path
import unittest

import yaml


ROOT = Path(__file__).resolve().parents[3]
ARGO_APP = ROOT / "argocd/overlays/dev/apps/hermes-umi.yaml"
BASE = ROOT / "apps/60-services/hermes/base"
DEV = ROOT / "apps/60-services/hermes/overlays/dev/kustomization.yaml"
ACTIVE_DATA_CLAIM = "hermes-data-synelia-recovery"
ACTIVE_DATA_CLASS = "synelia-iscsi-stateful-retain"
ACTIVE_WORKSPACE_CLAIM = "hairem-files-synelia"


def check_stateful_source(app: dict, pvc: dict, deployment: dict, overlay: dict) -> None:
    """Fail closed if the declared running application changes data identity."""
    if (app.get("kind") != "Application"
            or app.get("metadata", {}).get("name") != "hermes-umi"):
        raise AssertionError("Hermes UMI Argo Application identity changed")
    source = app.get("spec", {}).get("source", {})
    dest = app.get("spec", {}).get("destination", {})
    if (source.get("targetRevision") != "main"
            or source.get("path") != "apps/60-services/hermes/overlays/dev"
            or dest.get("namespace") != "services"):
        raise AssertionError("Hermes UMI GitOps source or namespace changed")
    policy = app.get("spec", {}).get("syncPolicy", {})
    if "automated" in policy:
        raise AssertionError("automatic Hermes UMI sync requires physical acceptance")
    if any(
        option.lower() in {"prune=true", "force=true", "replace=true"}
        for option in policy.get("syncOptions", [])
    ):
        raise AssertionError("unsafe Argo prune/force/replace request")

    if overlay.get("resources") != ["../../base"]:
        raise AssertionError("unexpected Hermes dev PVC replacement overlay")
    if (pvc.get("kind") != "PersistentVolumeClaim"
            or pvc.get("metadata", {}).get("name") != ACTIVE_DATA_CLAIM
            or pvc.get("metadata", {}).get("namespace") != "services"):
        raise AssertionError("Hermes retained PVC physical identity changed")
    spec = pvc.get("spec", {})
    if (spec.get("accessModes") != ["ReadWriteOnce"]
            or spec.get("storageClassName") != ACTIVE_DATA_CLASS
            or spec.get("resources", {}).get("requests", {}).get("storage") != "80Gi"
            or spec.get("volumeName") or spec.get("dataSource")
            or spec.get("dataSourceRef")):
        raise AssertionError("Hermes existing-data PVC specification changed")

    if (deployment.get("kind") != "Deployment"
            or deployment.get("metadata", {}).get("name") != "hermes"
            or deployment.get("metadata", {}).get("namespace") != "services"):
        raise AssertionError("Hermes Deployment identity changed")
    template = deployment.get("spec", {}).get("template", {}).get("spec", {})
    claims = [
        (v.get("name"), v["persistentVolumeClaim"].get("claimName"))
        for v in template.get("volumes", [])
        if "persistentVolumeClaim" in v
    ]
    if sorted(claims) != sorted([
        ("data", ACTIVE_DATA_CLAIM),
        ("hairem-files", ACTIVE_WORKSPACE_CLAIM),
    ]):
        raise AssertionError("Hermes data or workspace PVC binding changed")
    containers = template.get("containers", []) + template.get("initContainers", [])
    if not any(
        m.get("name") == "data" and m.get("mountPath") == "/opt/data"
        for container in containers
        for m in container.get("volumeMounts", [])
    ):
        raise AssertionError("Hermes private home is not mounted at /opt/data")


# Historical recovery descriptions belong in the archived issue, not
# current deployable manifests that define the retained steady state.
OBSOLETE_CUTOVER_NARRATIVE = (
    "Incident #4185",
    "temporary data-copy Pod",
    "PVC is staged",
    "staged migrations",
    "copy-first migrations",
)


def assert_current_stateful_manifest_narrative(content: str) -> None:
    for phrase in OBSOLETE_CUTOVER_NARRATIVE:
        if phrase in content:
            raise AssertionError("active stateful manifests must not describe an ongoing CSI cutover")


class HermesUMIStatefulSourceTests(unittest.TestCase):
    def setUp(self):
        self.app = yaml.safe_load(ARGO_APP.read_text(encoding="utf-8"))
        self.pvc = yaml.safe_load((BASE / "pvc.yaml").read_text(encoding="utf-8"))
        self.deploy = yaml.safe_load((BASE / "deployment.yaml").read_text(encoding="utf-8"))
        self.overlay = yaml.safe_load(DEV.read_text(encoding="utf-8"))

    def assert_contract(self, app=None, pvc=None, deployment=None, overlay=None):
        check_stateful_source(
            app if app is not None else self.app,
            pvc if pvc is not None else self.pvc,
            deployment if deployment is not None else self.deploy,
            overlay if overlay is not None else self.overlay,
        )

    def test_exact_existing_retained_data_source(self):
        self.assert_contract()

    def test_active_manifests_describe_current_state_not_historical_cutover(self):
        for path in (ARGO_APP, BASE / "pvc.yaml"):
            with self.subTest(path=path.name):
                assert_current_stateful_manifest_narrative(path.read_text(encoding="utf-8"))

    def test_old_cutover_narrative_fails_closed(self):
        for phrase in OBSOLETE_CUTOVER_NARRATIVE:
            with self.subTest(phrase=phrase), self.assertRaises(AssertionError):
                assert_current_stateful_manifest_narrative("  # " + phrase + "\\n")
        assert_current_stateful_manifest_narrative(
            "  # Preserve active data and require physical acceptance before automation."
        )


    def test_arbitrary_automated_sync_and_destructive_options_fail_closed(self):
        for mutation in (
            lambda x: x["spec"]["syncPolicy"].update(automated={}),
            lambda x: x["spec"]["syncPolicy"].update(automated={"prune": False}),
            lambda x: x["spec"]["syncPolicy"]["syncOptions"].append("Prune=true"),
            lambda x: x["spec"]["syncPolicy"]["syncOptions"].append("Force=true"),
            lambda x: x["spec"]["syncPolicy"]["syncOptions"].append("Replace=true"),
            lambda x: x["spec"]["source"].update(path="apps/60-services/hermes/overlays/prod"),
            lambda x: x["spec"]["destination"].update(namespace="tenant-foreign"),
        ):
            app = deepcopy(self.app)
            mutation(app)
            with self.subTest(app=app), self.assertRaises(AssertionError):
                self.assert_contract(app=app)

    def test_retained_claim_must_not_silently_be_replaced_or_reprovisioned(self):
        for mutation in (
            lambda x: x["metadata"].update(name="fresh-empty-data"),
            lambda x: x["metadata"].update(namespace="tenant-foreign"),
            lambda x: x["spec"].update(storageClassName="storage-rwo-delete"),
            lambda x: x["spec"].update(accessModes=["ReadWriteMany"]),
            lambda x: x["spec"].update(resources={"requests": {"storage": "1Gi"}}),
            lambda x: x["spec"].update(dataSource={"name": "wrong", "kind": "VolumeSnapshot"}),
        ):
            pvc = deepcopy(self.pvc)
            mutation(pvc)
            with self.subTest(pvc=pvc), self.assertRaises(AssertionError):
                self.assert_contract(pvc=pvc)

    def test_retained_claim_deployment_routing_remains_exact(self):
        for name in ("fresh-empty-data", "foreign-tenant-claim"):
            deploy = deepcopy(self.deploy)
            volume = next(x for x in deploy["spec"]["template"]["spec"]["volumes"]
                          if x["name"] == "data")
            volume["persistentVolumeClaim"]["claimName"] = name
            with self.subTest(claim=name), self.assertRaises(AssertionError):
                self.assert_contract(deployment=deploy)
        overlay = deepcopy(self.overlay)
        overlay["resources"].append("pvc-replacement.yaml")
        with self.assertRaises(AssertionError):
            self.assert_contract(overlay=overlay)


if __name__ == "__main__":
    unittest.main()
