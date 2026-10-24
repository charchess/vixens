"""Contract & regression tests for fail-closed UMI PV cutover planning (#4185)."""

from __future__ import annotations

from copy import deepcopy
import importlib.util
from pathlib import Path
import tempfile
import unittest
import json


SCRIPT = Path(__file__).resolve().parents[2] / "storage" / "umi_iscsi_pv_cutover.py"
spec = importlib.util.spec_from_file_location("umi_cutover", SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


def fixture() -> dict:
    items = []
    for name, (namespace, claim, phase, reclaim) in module.EXPECTED.items():
        items.append({
            "apiVersion": "v1", "kind": "PersistentVolume",
            "metadata": {"name": name, "uid": "pv-uid", "resourceVersion": "456",
                         "finalizers": ["kubernetes.io/pv-protection"]},
            "status": {"phase": phase},
            "spec": {
                "capacity": {"storage": "80Gi"},
                "accessModes": ["ReadWriteOnce"],
                "storageClassName": "truenas-iscsi-delete" if reclaim == "Delete" else "truenas-iscsi-retain",
                "persistentVolumeReclaimPolicy": reclaim,
                "claimRef": {"namespace": namespace, "name": claim, "uid": "uid-" + name},
                "csi": {"driver": module.DRIVER, "volumeHandle": name,
                        "volumeAttributes": {
                            "portal": module.OLD_PORTAL,
                            "iqn": f"iqn.2005-10.org.freenas.ctl:csi-{name}-umi",
                            "lun": 0, "customOpaqueSetting": "preserved"}},
            },
        })
    return {"apiVersion": "v1", "kind": "PersistentVolumeList", "items": items}


class OfflineCutoverTests(unittest.TestCase):
    def test_exact_ids_and_only_portal_and_cache_reclaim_change(self):
        source = fixture()
        before = deepcopy(source)
        out = module.prepare(source)
        self.assertEqual(source, before, "inputs must not mutate")
        self.assertEqual(len(out), 4)
        for old, candidate in zip(source["items"], out):
            expected = deepcopy(old["spec"])
            expected["csi"]["volumeAttributes"]["portal"] = module.NEW_PORTAL
            expected["persistentVolumeReclaimPolicy"] = "Retain"
            self.assertEqual(candidate["spec"], expected)
            self.assertEqual(candidate["metadata"], {"name": old["metadata"]["name"]})
            self.assertNotIn("status", candidate)
            self.assertNotIn("finalizers", candidate["metadata"])

    def assert_refused(self, mutate):
        source = fixture()
        mutate(source)
        with self.assertRaises(module.CutoverError):
            module.prepare(source)

    def test_wrong_tenant_or_pvc_denied(self):
        for field, value in (("namespace", "prod"), ("name", "other-pvc"), ("uid", "")):
            with self.subTest(field=field):
                self.assert_refused(lambda x: x["items"][0]["spec"]["claimRef"].__setitem__(field, value))

    def test_lun_volumehandle_target_driver_denied(self):
        for path, value in (("volumeHandle", "other-volume"), ("driver", "other.driver")):
            with self.subTest(field=path):
                self.assert_refused(lambda x: x["items"][0]["spec"]["csi"].__setitem__(path, value))
        for path, value in (("portal", module.NEW_PORTAL), ("iqn", "other"), ("lun", 2)):
            with self.subTest(field=path):
                self.assert_refused(lambda x: x["items"][0]["spec"]["csi"]["volumeAttributes"].__setitem__(path, value))

    def test_missing_duplicate_or_unexpected_old_portal_denied(self):
        self.assert_refused(lambda x: x["items"].pop())
        self.assert_refused(lambda x: x["items"].append(deepcopy(x["items"][0])))
        def foreign(x):
            extra = deepcopy(x["items"][0])
            extra["metadata"]["name"] = "unknown-prod-pv"
            x["items"].append(extra)
        self.assert_refused(foreign)

    def test_wrong_reclaim_status_class_or_capacity_denied(self):
        self.assert_refused(lambda x: x["items"][3]["spec"].__setitem__("persistentVolumeReclaimPolicy", "Retain"))
        self.assert_refused(lambda x: x["items"][0]["status"].__setitem__("phase", "Available"))
        self.assert_refused(lambda x: x["items"][0]["spec"].__setitem__("capacity", {}))
        self.assert_refused(lambda x: x["items"][0]["spec"].__setitem__("storageClassName", ""))

    def test_potential_inline_credential_attribute_denied(self):
        self.assert_refused(lambda x: x["items"][0]["spec"]["csi"]["volumeAttributes"].__setitem__("password", "DO-NOT-PRINT"))

    def test_cli_never_contacts_cluster_and_fails_closed(self):
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / "pv.json"
            path.write_text(json.dumps(fixture()), encoding="utf-8")
            self.assertEqual(module.main(["--pv-json", str(path)]), 0)
            path.write_text(json.dumps({"items": []}), encoding="utf-8")
            self.assertEqual(module.main(["--pv-json", str(path)]), 2)


if __name__ == "__main__":
    unittest.main()
