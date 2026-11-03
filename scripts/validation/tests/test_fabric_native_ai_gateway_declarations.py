"""Prevent one-off LiteLLM recovery requests in portable Fabric tenant GitOps.

The operator's UID-bound migration history lives in Kubernetes status. New
declarative tenants must start at schema attempt zero without historical Job IDs.
See #4387; the rollout of this guard depends on the matching operator/CRD PR.
"""

from pathlib import Path
import unittest

import yaml


ROOT = Path(__file__).resolve().parents[3]
TENANTS = ROOT / "apps/60-services/txo-fabric/tenants"
INCIDENT_FIELDS = frozenset({
    "migrationRetry",
    "migrationRecovery",
    "migrationRecoveryFailedGeneration",
    "migrationRecoveryTransitions",
})


def assert_native_gateway_declaration(document: dict) -> None:
    if document.get("kind") != "TenantBundle":
        return
    gateway = (document.get("spec") or {}).get("aiGateway") or {}
    forbidden = INCIDENT_FIELDS.intersection(gateway)
    if forbidden:
        raise AssertionError(
            "portable TenantBundle contains historical LiteLLM recovery inputs: "
            + ", ".join(sorted(forbidden))
        )


class NativeAIGatewayDeclarationsTests(unittest.TestCase):
    def test_all_active_tenant_bundles_start_without_incident_history(self):
        files = sorted(TENANTS.glob("*/**/*.yaml"))
        self.assertTrue(files, "tenant declarations missing")
        checked = 0
        for path in files:
            for document in yaml.safe_load_all(path.read_text(encoding="utf-8")):
                if not isinstance(document, dict):
                    continue
                with self.subTest(path=str(path.relative_to(ROOT))):
                    assert_native_gateway_declaration(document)
                if document.get("kind") == "TenantBundle":
                    checked += 1
        self.assertGreater(checked, 0, "no TenantBundle declarations checked")

    def test_rejects_each_incident_only_field_for_arbitrary_tenant(self):
        for field in INCIDENT_FIELDS:
            with self.subTest(field=field), self.assertRaises(AssertionError):
                assert_native_gateway_declaration({
                    "kind": "TenantBundle",
                    "metadata": {"name": "arbitrary-client"},
                    "spec": {"aiGateway": {field: 0 if field == "migrationRetry" else "legacy"}},
                })

    def test_normal_tenant_ai_gateway_remains_portable(self):
        assert_native_gateway_declaration({
            "kind": "TenantBundle",
            "metadata": {"name": "arbitrary-client"},
            "spec": {"aiGateway": {"profileRef": "litellm-standard"}},
        })
        # Other CRs and independent SQL migration resources are not historical
        # LiteLLM recovery *requests* and are outside this source guard.
        assert_native_gateway_declaration({
            "kind": "AgentIdentity", "spec": {"runtime": {"storage": {"retentionPolicy": "Retain"}}},
        })
        assert_native_gateway_declaration({"kind": "Job", "metadata": {"name": "db-schema"}})


if __name__ == "__main__":
    unittest.main()
