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


if __name__ == "__main__":
    unittest.main()
