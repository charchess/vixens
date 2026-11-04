"""Guard the declarative Fabric tenant baseline against obsolete CSI cutover inputs.

The API may still decode stored legacy CRs for compatibility, but GitOps must
not request a historical copy/cutover or legacy-profile adoption, even as
an explicit false compatibility field. This is a
source-only contract and never contacts a Kubernetes cluster.
"""

from pathlib import Path
import unittest

import yaml


ROOT = Path(__file__).resolve().parents[3]
TENANTS = ROOT / "apps/60-services/txo-fabric/tenants"
CONTROLLERS = ROOT / "apps/60-services/txo-fabric/operator/internal/controller"
PROFILES = ROOT / "apps/60-services/txo-fabric/operator/config/profiles"
NATIVE_PROFILE_CLASSES = {
    "AgentRuntimeProfile": {"storage-rwo-retain", "storage-rwo-delete"},
    "SharedWorkspaceProfile": {"storage-rwx-retain", "storage-rwx-delete"},
}

AGENT_CRD = ROOT / "apps/60-services/txo-fabric/operator/config/crd/bases/fabric.truxonline.io_agentidentities.yaml"
TENANT_CRD = ROOT / "apps/60-services/txo-fabric/operator/config/crd/bases/fabric.truxonline.io_tenantbundles.yaml"

ARCHIVED_CSI_STATUS_REFERENCES = (
    "RuntimeStorageMigration",
    "WorkspaceMigration",
    "LegacyRecovered",
    "LegacyTargetBinding",
)


def assert_native_storage_declaration(document: dict) -> None:
    kind = document.get("kind")
    spec = document.get("spec") or {}
    if kind == "TenantBundle":
        workspace = spec.get("workspace") or {}
        forbidden = {"claimSuffix", "migration"} & workspace.keys()
        if forbidden:
            raise AssertionError(f"TenantBundle workspace has historical CSI fields: {sorted(forbidden)}")
        # New GitOps tenant declarations must not quietly fall back to the
        # Hermes release profile's storage class. Existing stored CRs are
        # unaffected; the operator's compatibility fallback is not changed.
        policy = spec.get("agentStorage")
        expected = {
            "retainedStorageClassName": "storage-rwo-retain",
            "disposableStorageClassName": "storage-rwo-delete",
        }
        if not isinstance(policy, dict) or any(
            policy.get(key) != value for key, value in expected.items()
        ):
            raise AssertionError(
                "TenantBundle must declare provider-neutral RWO Retain/Delete classes "
                "in spec.agentStorage; profile-class fallback is compatibility-only"
            )
    elif kind == "AgentIdentity":
        storage = ((spec.get("runtime") or {}).get("storage") or {})
        forbidden = {"migration"} & storage.keys()
        if forbidden or "adoptLegacyProfile" in storage:
            raise AssertionError("AgentIdentity runtime storage requests historical CSI cutover/adoption")


def assert_native_storage_profile(document: dict) -> None:
    """GitOps profiles select stable RWO/RWX capabilities, not CSI backends."""
    kind = document.get("kind")
    if kind not in NATIVE_PROFILE_CLASSES:
        return
    spec = document.get("spec") or {}
    storage_class = (
        (spec.get("storage") or {}).get("storageClassName")
        if kind == "AgentRuntimeProfile"
        else spec.get("storageClassName")
    )
    if not isinstance(storage_class, str) or storage_class not in NATIVE_PROFILE_CLASSES[kind]:
        raise AssertionError(
            f"{kind} declares unsupported {storage_class!r}; "
            "Fabric profiles must use the native RWO/RWX capability classes"
        )


def assert_native_csi_crd_compatibility(agent_crd: dict, tenant_crd: dict) -> None:
    """Archived CSI status must survive API upgrades, never return as spec intent."""
    def version_schema(document: dict) -> dict:
        versions = document.get("spec", {}).get("versions", [])
        matches = [v for v in versions if v.get("name") == "v1alpha1" and v.get("served")]
        if len(matches) != 1:
            raise AssertionError("expected exactly one served Fabric v1alpha1 schema")
        return matches[0]["schema"]["openAPIV3Schema"]["properties"]

    agent_schema = version_schema(agent_crd)
    tenant_schema = version_schema(tenant_crd)
    agent_spec = agent_schema["spec"]["properties"]
    tenant_spec = tenant_schema["spec"]["properties"]
    agent_storage = agent_spec["runtime"]["properties"]["storage"]["properties"]
    workspace = tenant_spec["workspace"]["properties"]
    if {"migration", "claimSuffix", "targetClaimSuffix", "copyJobName"} & set(agent_storage):
        raise AssertionError("retired CSI agent storage fields returned to active CRD spec")
    if {"migration", "claimSuffix", "targetClaimSuffix", "copyJobPrefix"} & set(workspace):
        raise AssertionError("retired CSI workspace fields returned to active CRD spec")
    if "adoptLegacyProfile" not in agent_storage:
        raise AssertionError("stored AgentIdentity adoptLegacyProfile sentinel was pruned")

    agent_status = agent_schema["status"]["properties"].get("runtimeStorageMigration")
    workspace_status = tenant_schema["status"]["properties"].get("workspaceMigration")
    if not isinstance(agent_status, dict) or not isinstance(workspace_status, dict):
        raise AssertionError("archived CSI status disappeared from Kubernetes CRD schema")
    required_binding = {
        "sourceClaimName", "sourcePVCUID", "targetClaimName",
        "targetPVCUID", "phase", "verificationResult",
    }
    for name, node in (
        ("runtimeStorageMigration", agent_status),
        ("workspaceMigration", workspace_status.get("items", {})),
    ):
        fields = node.get("properties", {})
        if not required_binding <= set(fields) or not required_binding <= set(node.get("required", [])):
            raise AssertionError(f"{name} lost required source/target UID or verification fields")
        if "LegacyRecovered" not in fields["phase"].get("enum", []):
            raise AssertionError(f"{name} lost archived LegacyRecovered phase")
    if not {"Job", "LegacyTargetBinding"} <= set(
        agent_status["properties"].get("evidenceSource", {}).get("enum", [])
    ):
        raise AssertionError("archived RWO evidence-source enum was pruned")
    if workspace_status.get("type") != "array" or workspace_status.get(
        "x-kubernetes-list-map-keys"
    ) != ["sourceClaimName"]:
        raise AssertionError("workspace archive list identity was changed")


def assert_no_archived_csi_status_in_controller(code: str) -> None:
    """Archived CSI status remains decodable, never a live routing authority."""
    import re

    for identifier in ARCHIVED_CSI_STATUS_REFERENCES:
        if re.search(r"\b" + identifier + r"\b", code):
            raise AssertionError(
                "controller must not consult archived CSI evidence: " + identifier
            )




class NativeFabricStorageSourceTests(unittest.TestCase):
    def test_executable_controller_does_not_consume_archived_csi_status(self):
        files = sorted(CONTROLLERS.glob("*.go"))
        files = [path for path in files if not path.name.endswith("_test.go")]
        self.assertTrue(files, "no executable Fabric controller sources found")
        for path in files:
            with self.subTest(path=str(path.relative_to(ROOT))):
                assert_no_archived_csi_status_in_controller(
                    path.read_text(encoding="utf-8")
                )

    def test_archived_csi_status_cannot_reenter_runtime_routing(self):
        for name in ARCHIVED_CSI_STATUS_REFERENCES:
            with self.subTest(name=name), self.assertRaises(AssertionError):
                assert_no_archived_csi_status_in_controller(
                    "func reconcile() { _ = bundle.Status." + name + " }"
                )
        assert_no_archived_csi_status_in_controller(
            'func reconcile() { ensureNativePVC(); ensureTenantWorkspace() }'
        )

    def test_gitops_storage_profiles_use_native_classes(self):
        manifests = sorted(PROFILES.glob("*.yaml"))
        self.assertTrue(manifests, "Fabric platform profiles missing")
        found = set()
        for path in manifests:
            for document in yaml.safe_load_all(path.read_text(encoding="utf-8")):
                if not isinstance(document, dict):
                    continue
                kind = document.get("kind")
                if kind not in NATIVE_PROFILE_CLASSES:
                    continue
                found.add(kind)
                with self.subTest(path=str(path.relative_to(ROOT))):
                    assert_native_storage_profile(document)
        self.assertEqual(found, set(NATIVE_PROFILE_CLASSES))

    def test_native_profile_policy_rejects_retired_or_wrong_access_classes(self):
        for kind, location in (
            ("AgentRuntimeProfile", "storage"),
            ("SharedWorkspaceProfile", "workspace"),
        ):
            for bad in (
                None, "", "synelia-iscsi-retain", "truenas-nfs-delete",
                "retired-csi-rwo-retain", "storage-rwo-xfs-retain",
                "storage-rwx-retain" if kind == "AgentRuntimeProfile" else "storage-rwo-retain",
            ):
                spec = (
                    {"storage": {"storageClassName": bad}}
                    if location == "storage" else {"storageClassName": bad}
                )
                with self.subTest(kind=kind, storageClass=bad), self.assertRaises(AssertionError):
                    assert_native_storage_profile({"kind": kind, "spec": spec})
            valid = (
                {"storage": {"storageClassName": "storage-rwo-delete"}}
                if location == "storage" else {"storageClassName": "storage-rwx-retain"}
            )
            assert_native_storage_profile({"kind": kind, "spec": valid})

    def test_generated_crds_preserve_archived_status_without_cutover_spec(self):
        agent_crd = yaml.safe_load(AGENT_CRD.read_text(encoding="utf-8"))
        tenant_crd = yaml.safe_load(TENANT_CRD.read_text(encoding="utf-8"))
        assert_native_csi_crd_compatibility(agent_crd, tenant_crd)

    def test_crd_archive_fail_closed_mutations(self):
        from copy import deepcopy

        agent = yaml.safe_load(AGENT_CRD.read_text(encoding="utf-8"))
        tenant = yaml.safe_load(TENANT_CRD.read_text(encoding="utf-8"))
        def properties(crd):
            return crd["spec"]["versions"][0]["schema"]["openAPIV3Schema"]["properties"]

        mutations = (
            ("missing agent archive", lambda a, t: properties(a)["status"]["properties"].pop("runtimeStorageMigration")),
            ("missing workspace archive", lambda a, t: properties(t)["status"]["properties"].pop("workspaceMigration")),
            ("missing agent target UID", lambda a, t: properties(a)["status"]["properties"]["runtimeStorageMigration"]["properties"].pop("targetPVCUID")),
            ("missing workspace source UID", lambda a, t: properties(t)["status"]["properties"]["workspaceMigration"]["items"]["properties"].pop("sourcePVCUID")),
            ("lost workspace map key", lambda a, t: properties(t)["status"]["properties"]["workspaceMigration"].pop("x-kubernetes-list-map-keys")),
            ("lost legacy phase", lambda a, t: properties(a)["status"]["properties"]["runtimeStorageMigration"]["properties"]["phase"].update(enum=["Succeeded"])),
            ("lost identity sentinel", lambda a, t: properties(a)["spec"]["properties"]["runtime"]["properties"]["storage"]["properties"].pop("adoptLegacyProfile")),
            ("reintroduced agent cutover", lambda a, t: properties(a)["spec"]["properties"]["runtime"]["properties"]["storage"]["properties"].update(migration={"type": "object"})),
            ("reintroduced workspace suffix", lambda a, t: properties(t)["spec"]["properties"]["workspace"]["properties"].update(claimSuffix={"type": "string"})),
        )
        for label, change in mutations:
            mutated_agent, mutated_tenant = deepcopy(agent), deepcopy(tenant)
            change(mutated_agent, mutated_tenant)
            with self.subTest(label=label), self.assertRaises(AssertionError):
                assert_native_csi_crd_compatibility(mutated_agent, mutated_tenant)

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

    def test_rejects_missing_or_non_native_tenant_storage_policy(self):
        valid = {
            "kind": "TenantBundle",
            "metadata": {"name": "arbitrary-third-tenant"},
            "spec": {
                "workspace": {"profileRef": "workspace-shared"},
                "agentStorage": {
                    "retainedStorageClassName": "storage-rwo-retain",
                    "disposableStorageClassName": "storage-rwo-delete",
                },
            },
        }
        assert_native_storage_declaration(valid)
        invalid_policies = (
            None,
            {},
            {"retainedStorageClassName": "storage-rwo-retain"},
            {"disposableStorageClassName": "storage-rwo-delete"},
            {"retainedStorageClassName": "storage-rwo-retain",
             "disposableStorageClassName": "storage-rwo-retain"},
            {"retainedStorageClassName": "synelia-iscsi-stateful-retain",
             "disposableStorageClassName": "storage-rwo-delete"},
            {"retainedStorageClassName": "storage-rwo-retain",
             "disposableStorageClassName": "synelia-iscsi-delete"},
        )
        for policy in invalid_policies:
            with self.subTest(policy=policy), self.assertRaises(AssertionError):
                document = {
                    "kind": "TenantBundle",
                    "metadata": {"name": "arbitrary-third-tenant"},
                    "spec": {"agentStorage": policy},
                }
                assert_native_storage_declaration(document)

    def test_rejects_old_workspace_cutover_and_claim_alias(self):
        native_policy = {
            "retainedStorageClassName": "storage-rwo-retain",
            "disposableStorageClassName": "storage-rwo-delete",
        }
        for field, value in (("claimSuffix", "synelia"), ("migration", {"cutover": True})):
            with self.subTest(field=field), self.assertRaises(AssertionError):
                assert_native_storage_declaration({
                    "kind": "TenantBundle",
                    "spec": {"workspace": {field: value}, "agentStorage": native_policy},
                })

    def test_rejects_old_agent_cutover_and_profile_adoption(self):
        for storage in ({"migration": {"cutover": True}}, {"adoptLegacyProfile": True}, {"adoptLegacyProfile": False}):
            with self.subTest(storage=storage), self.assertRaises(AssertionError):
                assert_native_storage_declaration({"kind": "AgentIdentity", "spec": {"runtime": {"storage": storage}}})

    def test_native_storage_and_independent_sql_migrations_are_allowed(self):
        assert_native_storage_declaration({"kind": "TenantBundle", "spec": {
            "workspace": {"profileRef": "shared-nfs"},
            "agentStorage": {
                "retainedStorageClassName": "storage-rwo-retain",
                "disposableStorageClassName": "storage-rwo-delete",
            },
            "aiGateway": {"migrationRetry": 1},
        }})
        assert_native_storage_declaration({"kind": "AgentIdentity", "spec": {
            "runtime": {"storage": {"retentionPolicy": "Retain"}}
        }})


if __name__ == "__main__":
    unittest.main()
