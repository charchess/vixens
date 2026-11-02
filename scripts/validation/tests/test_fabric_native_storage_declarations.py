"""Guard the declarative Fabric tenant baseline against obsolete CSI cutover inputs.

The API may still decode stored legacy CRs for compatibility, but GitOps must
not request a historical copy/cutover or legacy-profile adoption. This is a
source-only contract and never contacts a Kubernetes cluster.
"""

from pathlib import Path
import unittest

import yaml


ROOT = Path(__file__).resolve().parents[3]
TENANTS = ROOT / "apps/60-services/txo-fabric/tenants"


def assert_native_storage_declaration(document: dict) -> None:
    kind = document.get("kind")
    spec = document.get("spec") or {}
    if kind == "TenantBundle":
        workspace = spec.get("workspace") or {}
        forbidden = {"claimSuffix", "migration"} & workspace.keys()
        if forbidden:
            raise AssertionError(f"TenantBundle workspace has historical CSI fields: {sorted(forbidden)}")
    elif kind == "AgentIdentity":
        storage = ((spec.get("runtime") or {}).get("storage") or {})
        forbidden = {"migration"} & storage.keys()
        if forbidden or storage.get("adoptLegacyProfile") is True:
            raise AssertionError("AgentIdentity runtime storage requests historical CSI cutover/adoption")


class NativeFabricStorageSourceTests(unittest.TestCase):
    def test_all_declared_tenants_have_no_csi_cutover_inputs(self):
        manifests = sorted(TENANTS.glob("*/**/*.yaml"))
        self.assertTrue(manifests, "Tenant manifests missing")
        seen = set()
        for path in manifests:
            for document in yaml.safe_load_all(path.read_text(encoding="utf-8")):
                if not isinstance(document, dict):
                    continue
                if document.get("kind") not in {"TenantBundle", "AgentIdentity"}:
                    continue
                seen.add(document["kind"])
                with self.subTest(path=str(path.relative_to(ROOT)), name=document.get("metadata", {}).get("name")):
                    assert_native_storage_declaration(document)
        self.assertEqual(seen, {"TenantBundle", "AgentIdentity"})

    def test_rejects_old_workspace_cutover_and_claim_alias(self):
        for field, value in (("claimSuffix", "synelia"), ("migration", {"cutover": True})):
            with self.subTest(field=field), self.assertRaises(AssertionError):
                assert_native_storage_declaration({"kind": "TenantBundle", "spec": {"workspace": {field: value}}})

    def test_rejects_old_agent_cutover_and_profile_adoption(self):
        for storage in ({"migration": {"cutover": True}}, {"adoptLegacyProfile": True}):
            with self.subTest(storage=storage), self.assertRaises(AssertionError):
                assert_native_storage_declaration({"kind": "AgentIdentity", "spec": {"runtime": {"storage": storage}}})

    def test_native_storage_and_independent_sql_migrations_are_allowed(self):
        assert_native_storage_declaration({"kind": "TenantBundle", "spec": {
            "workspace": {"profileRef": "shared-nfs"},
            "aiGateway": {"migrationRetry": 1},
        }})
        assert_native_storage_declaration({"kind": "AgentIdentity", "spec": {
            "runtime": {"storage": {"retentionPolicy": "Retain"}}
        }})


if __name__ == "__main__":
    unittest.main()
