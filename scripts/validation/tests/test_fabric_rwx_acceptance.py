"""Contract tests for read-only Fabric RWX physical acceptance (#4400)."""
import importlib.util
from pathlib import Path
import unittest

SOURCE = Path(__file__).resolve().parents[1] / "fabric_rwx_acceptance.py"
SPEC = importlib.util.spec_from_file_location("fabric_rwx_acceptance", SOURCE)
mod = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(mod)


def bundle(name="tenant-a", tenant_id="TEN90000"):
    return {"kind": "TenantBundle", "metadata": {"name": name},
            "spec": {"tenantId": tenant_id, "workspace": {
                "profileRef": "shared-nfs", "retentionPolicy": "Retain",
                "groups": [{"name": "team", "collaborative": True,
                            "skillsCollaborative": True}],
                "users": [{"name": "reader", "reference": True}],
            }}}


def inventory(tenant):
    claims = mod.expected_claims(tenant)
    t = tenant["metadata"]["name"]
    ns = "tenant-" + t
    pvc_list, pv_list, volumes = [], [], []
    for name, scope in claims.items():
        uid, pv_name = "uid-" + name, "pv-" + name
        labels = {mod.LABEL + "tenant-name": t,
                  mod.LABEL + "tenant-id": tenant["spec"]["tenantId"],
                  mod.LABEL + "workspace-domain": scope["domain"],
                  mod.LABEL + "workspace-scope": scope["scope"],
                  mod.LABEL + "workspace-key": scope["key"],
                  mod.LABEL + "workspace-mode": scope["mode"],
                  mod.LABEL + "workspace-retention": "Retain"}
        pvc_list.append({"metadata": {"name": name, "uid": uid, "labels": labels},
                         "spec": {"volumeName": pv_name,
                                  "storageClassName": mod.STORAGE_CLASS,
                                  "accessModes": ["ReadWriteMany"]},
                         "status": {"phase": "Bound"}})
        pv_list.append({"metadata": {"name": pv_name},
                        "spec": {"claimRef": {"name": name, "namespace": ns, "uid": uid},
                                 "storageClassName": mod.STORAGE_CLASS,
                                 "persistentVolumeReclaimPolicy": "Retain",
                                 "csi": {"driver": "test.csi.example"}}})
        volumes.append({"name": name, "persistentVolumeClaim": {"claimName": name}})
    pod = {"metadata": {"name": "agent-one"},
           "spec": {"volumes": volumes,
                    "containers": [{"name": "agent", "volumeMounts": [
                        {"name": name, "readOnly": scope["mode"] == "reference"}
                        for name, scope in claims.items()]}]}}
    return {"pvcs": {t: pvc_list}, "pvs": pv_list, "pods": {t: [pod]}}


class FabricRWXAcceptanceTests(unittest.TestCase):
    def setUp(self):
        self.tenant = bundle()
        self.snapshot = inventory(self.tenant)

    def check(self, code):
        result = mod.evaluate([self.tenant], self.snapshot)
        self.assertIn(code, [f["code"] for f in result["findings"]])
        self.assertEqual(result["status"], "FAIL")

    def test_native_claim_generation(self):
        self.assertEqual(set(mod.expected_claims(self.tenant)),
                         {"ws-group-team-rw", "skills-group-team-rw", "ws-user-reader-ref"})

    def test_good_binding_is_still_pending_functional_proof(self):
        result = mod.evaluate([self.tenant], self.snapshot)
        self.assertEqual(result["status"], "PENDING")
        self.assertEqual(result["physical_data_proof"], "PENDING")
        self.assertFalse([x for x in result["findings"] if x["level"] == "FAIL"])

    def test_missing_canonical_claim_fails_closed(self):
        self.snapshot["pvcs"]["tenant-a"].pop()
        self.check("MISSING_CANONICAL_PVC")

    def test_uid_substitution_fails_closed(self):
        self.snapshot["pvs"][0]["spec"]["claimRef"]["uid"] = "different"
        self.check("PV_CLAIMREF_MISMATCH")

    def test_cross_tenant_ref_fails_closed(self):
        self.snapshot["pvs"][0]["spec"]["claimRef"]["namespace"] = "tenant-foreign"
        self.check("PV_CLAIMREF_MISMATCH")

    def test_foreign_ownership_fails_closed(self):
        self.snapshot["pvcs"]["tenant-a"][0]["metadata"]["labels"][mod.LABEL + "tenant-name"] = "foreign"
        self.check("WORKSPACE_OWNERSHIP_MISMATCH")

    def test_legacy_mount_fails_closed(self):
        self.snapshot["pods"]["tenant-a"][0]["spec"]["volumes"].append({
            "name": "old", "persistentVolumeClaim": {"claimName": "ws-group-team-rw-synelia"}})
        self.check("LEGACY_MOUNT")

    def test_readonly_scope_must_not_be_writable(self):
        mounts = self.snapshot["pods"]["tenant-a"][0]["spec"]["containers"][0]["volumeMounts"]
        next(m for m in mounts if m["name"] == "ws-user-reader-ref")["readOnly"] = False
        self.check("REFERENCE_WRITABLE")

    def test_missing_consumer_is_pending(self):
        self.snapshot["pods"]["tenant-a"] = []
        result = mod.evaluate([self.tenant], self.snapshot)
        self.assertEqual(result["status"], "PENDING")
        self.assertIn("NO_OBSERVED_CONSUMER", [f["code"] for f in result["findings"]])

    def test_legacy_pvc_is_not_selected_or_deleted(self):
        self.snapshot["pvcs"]["tenant-a"].append({
            "metadata": {"name": "ws-group-team-rw-synelia"}})
        result = mod.evaluate([self.tenant], self.snapshot)
        self.assertEqual(result["status"], "PENDING")
        self.assertIn("LEGACY_CLAIM_RETAINED_FOR_ROLLBACK",
                      [f["code"] for f in result["findings"]])


if __name__ == "__main__":
    unittest.main()
