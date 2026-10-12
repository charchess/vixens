"""Tests for #4046 source-only OpenBao Fabric safety guard (no secrets)."""

from __future__ import annotations

import importlib.util
from pathlib import Path
import tempfile
import unittest


MODULE = Path(__file__).with_name("validate-openbao-fabric-source.py")
SPEC = importlib.util.spec_from_file_location("openbao_fabric_guard", MODULE)
assert SPEC and SPEC.loader
guard = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(guard)


class FoundationGuardTests(unittest.TestCase):
    def test_exact_platform_policy_allows_only_read(self):
        content = (
            "# not installed\n"
            'path "fabric/data/platform/txo-fabric/webui-bff" {\n'
            '  capabilities = ["read"]\n}\n'
        )
        self.assertEqual(
            guard.assert_file_safe(
                "apps/60-services/txo-fabric/openbao-fabric/policies/platform-webui-bff.hcl.example",
                content,
            ), [],
        )

    def test_unexpected_runnable_manifest_refused(self):
        f = "apps/60-services/txo-fabric/openbao-fabric/runtime/deployment.yaml"
        self.assertTrue(guard.assert_file_safe(f, "kind: Deployment\n"))

    def test_forged_admin_policy_or_all_tenant_wildcard_refused(self):
        f = "apps/60-services/txo-fabric/openbao-fabric/policies/tenant.hcl.example"
        self.assertTrue(guard.assert_file_safe(
            f, 'path "fabric/data/tenants/*" {\n capabilities = ["read"]\n}',
        ))
        self.assertTrue(guard.assert_file_safe(
            f, 'path "sys/policies" {\n capabilities = ["read"]\n}',
        ))
        self.assertTrue(guard.assert_file_safe(
            f, 'path "fabric/data/platform/" {\n capabilities = ["read", "create"]\n}',
        ))

    def test_literal_sensitive_values_denied_without_logging_them(self):
        f = "apps/60-services/txo-fabric/openbao-fabric/README.md"
        self.assertTrue(guard.assert_file_safe(f, "root_" + "token: " + ("sample" * 3)))
        self.assertTrue(guard.assert_file_safe(f, "pass" + "word=" + ("sample" * 3)))
        self.assertTrue(guard.assert_file_safe(
            f, "-----BEGIN " + "OPENSSH PRIVATE " + "KEY-----",
        ))

    def test_negative_render_marker(self):
        self.assertTrue(guard.PRODUCTION_MARKER.search(
            "kind: StatefulSet\nmetadata:\n  name: openbao-fabric",
        ))
        self.assertFalse(guard.PRODUCTION_MARKER.search(
            "kind: ClusterSecretStore\nmetadata:\n  name: openbao",
        ))

    def test_diff_infra_store_reference(self):
        self.assertTrue(guard.INFRA_STORE_ADDITION.search(
            "+    name: openbao\n",
        ))
        self.assertFalse(guard.INFRA_STORE_ADDITION.search(
            "+    name: openbao-fabric-store\n",
        ))


class CanonicalTenantProvisioningContractTests(unittest.TestCase):
    """Proof of rejection in the SOURCE oracle, not live OpenBao enforcement."""

    @staticmethod
    def fixture(action="Ensure"):
        canonical = {
            "source": {
                "apiVersion": "fabric.truxonline.io/v1alpha1",
                "kind": "TenantBundle",
                "name": "example-tenant",
                "uid": "fcba1212-1234-4123-8456-543210987654",
                "generation": 3,
            },
            "fabricTenantId": "TEN00001",
            "workload": {
                "namespace": "tenant-example-tenant",
                "serviceAccount": "example-agent",
                "audience": "reviewed-workload-audience",
            },
            "lifecycle": "Active",
            "operatorAuthenticated": True,
            "fabricAvailable": True,
        }
        request = {
            "contractVersion": "v1",
            "action": action,
            "source": dict(canonical["source"]),
            "fabricTenantId": canonical["fabricTenantId"],
            "workload": dict(canonical["workload"]),
            "pathTemplate": "tenant-runtime-v1",
            "idempotencyKey": guard.provisioning_idempotency_key(canonical, action),
        }
        return request, canonical

    def test_idempotent_canonical_tenant_request_only(self):
        request, canonical = self.fixture()
        self.assertEqual(guard.validate_provisioning_request(request, canonical), [])
        self.assertEqual(guard.validate_provisioning_request(request, canonical), [])
        self.assertEqual(
            request["idempotencyKey"],
            guard.provisioning_idempotency_key(canonical, "Ensure"),
        )

    def test_forged_policy_body_name_prefix_and_admin_scope_refused(self):
        for field, value in [
            ("hcl", 'path "sys/policies/*" { capabilities = ["create"] }'),
            ("policy", 'path "fabric/data/tenants/*" { capabilities = ["read"] }'),
            ("policyName", "tenant-TEN00001"),
            ("role", "all-tenants"),
            ("paths", ["fabric/tenants/TEN00001/runtime"]),
        ]:
            request, canonical = self.fixture()
            request[field] = value
            self.assertEqual(
                guard.validate_provisioning_request(request, canonical),
                ["ForbiddenFields"], field,
            )

    def test_wildcard_or_sibling_path_denied_without_installer(self):
        for path in [
            "fabric/tenants/*",
            "fabric/tenants/TEN00002/runtime",
            "fabric/tenants/TEN00001/../TEN00002",
        ]:
            request, canonical = self.fixture()
            request["paths"] = [path]
            self.assertEqual(
                guard.validate_provisioning_request(request, canonical),
                ["ForbiddenFields"],
            )
            request.pop("paths")
            request["pathTemplate"] = path
            self.assertEqual(
                guard.validate_provisioning_request(request, canonical),
                ["UnapprovedTemplate"],
            )
        request, canonical = self.fixture()
        request["fabricTenantId"] = "TEN00002"
        self.assertEqual(
            guard.validate_provisioning_request(request, canonical),
            ["UnknownOrStaleTenant"],
        )

    def test_group_user_browser_labels_never_act_as_secret_authority(self):
        for forged in [
            {"group": "fabric-admin"},
            {"user": "person"},
            {"oidcSub": "user-in-identity-provider"},
            {"tenantSlug": "someone-else"},
            {"browser": True},
            {"namespaceLabels": {"tenant": "example-tenant"}},
        ]:
            request, canonical = self.fixture()
            request.update(forged)
            self.assertEqual(
                guard.validate_provisioning_request(request, canonical),
                ["ForbiddenFields"],
            )
        request, canonical = self.fixture()
        request["source"]["kind"] = "Group"
        self.assertEqual(
            guard.validate_provisioning_request(request, canonical),
            ["InvalidSource"],
        )
        request, canonical = self.fixture()
        request["source"]["kind"] = "User"
        self.assertEqual(
            guard.validate_provisioning_request(request, canonical),
            ["InvalidSource"],
        )

    def test_exact_verified_workload_uid_and_generation_required(self):
        for key, value in [
            ("namespace", "tenant-other"),
            ("serviceAccount", "shared-admin"),
            ("audience", "wrong-audience"),
        ]:
            request, canonical = self.fixture()
            request["workload"][key] = value
            self.assertEqual(
                guard.validate_provisioning_request(request, canonical),
                ["InvalidWorkloadBinding"],
            )
        for key, value in [
            ("uid", "00000000-1111-4111-8111-111111111111"),
            ("generation", 2),
            ("name", "foreign-name"),
        ]:
            request, canonical = self.fixture()
            request["source"][key] = value
            self.assertEqual(
                guard.validate_provisioning_request(request, canonical),
                ["UnknownOrStaleTenant"],
            )
        request, canonical = self.fixture()
        canonical["operatorAuthenticated"] = False
        self.assertEqual(
            guard.validate_provisioning_request(request, canonical),
            ["UntrustedReconciler"],
        )

    def test_fail_closed_unknown_tenant_unavailable_and_revocation(self):
        request, canonical = self.fixture()
        canonical["fabricTenantId"] = "TEN99999"
        self.assertEqual(
            guard.validate_provisioning_request(request, canonical),
            ["UnknownOrStaleTenant"],
        )
        request, canonical = self.fixture()
        canonical["fabricAvailable"] = False
        self.assertEqual(
            guard.validate_provisioning_request(request, canonical),
            ["FabricUnavailable"],
        )
        request, canonical = self.fixture()
        canonical["lifecycle"] = "Deleting"
        self.assertEqual(
            guard.validate_provisioning_request(request, canonical),
            ["LifecycleDenied"],
        )
        request, canonical = self.fixture("Revoke")
        canonical["lifecycle"] = "Deleting"
        self.assertEqual(
            guard.validate_provisioning_request(request, canonical), [],
        )
        request["idempotencyKey"] = "forged-key"
        self.assertEqual(
            guard.validate_provisioning_request(request, canonical),
            ["IdempotencyMismatch"],
        )

    def test_new_infra_eso_binding_is_still_prohibited(self):
        self.assertTrue(guard.INFRA_STORE_ADDITION.search(
            "+        name: openbao\n",
        ))
        self.assertFalse(guard.INFRA_STORE_ADDITION.search(
            "+        name: openbao-fabric-store\n",
        ))


if __name__ == "__main__":
    unittest.main()
