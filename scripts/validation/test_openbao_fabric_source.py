"""Tests for #4046 empty private OpenBao Fabric POC guard (no secrets)."""

from __future__ import annotations

import copy
import importlib.util
from pathlib import Path
import tempfile
import unittest

import yaml


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

    def test_exact_storage_migration_target_is_the_only_additional_manifest(self):
        f = "apps/60-services/txo-fabric/openbao-fabric/overlays/prod/migration/synelia-target-pvc.yaml"
        self.assertEqual(guard.assert_file_safe(f, "kind: PersistentVolumeClaim\n"), [])
        self.assertTrue(guard.assert_file_safe(
            "apps/60-services/txo-fabric/openbao-fabric/overlays/prod/migration/other.yaml",
            "kind: PersistentVolumeClaim\n",
        ))

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

    def test_cert_manager_private_key_schema_not_secret_value(self):
        path = "apps/60-services/txo-fabric/openbao-fabric/base/certificate.yaml"
        self.assertEqual(guard.assert_file_safe(
            path, "privateKey:\\n  algorithm: ECDSA\\n  size: 256\\n",
        ), [])
        # Same-line credential-like assignments remain prohibited.
        self.assertTrue(guard.ASSIGNMENT.search(
            "private_" + "key: " + ("unsafe" * 3)
        ))

    def test_positive_poc_selection_marker(self):
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


class TransitSealPreflightTests(unittest.TestCase):
    """Tests only an unselected identity; NO fake TLS or Transit authority."""

    @staticmethod
    def staged():
        base = Path(__file__).resolve().parents[2] / (
            "apps/60-services/txo-fabric/openbao-fabric/transit-staged"
        )
        return list(yaml.safe_load_all(
            (base / "serviceaccount.yaml").read_text(encoding="utf-8")
        ))

    def test_exact_dedicated_sa_never_automounts(self):
        docs = self.staged()
        self.assertEqual(guard.validate_staged_seal_identity(docs), [])
        self.assertEqual(guard.SEAL_NAMESPACE, "txo-fabric-system")
        self.assertEqual(guard.SEAL_SA, "openbao-fabric-seal-bootstrap")
        self.assertEqual(guard.SEAL_AUDIENCE, "txo-fabric-infra-seal")
        self.assertEqual(guard.SEAL_TOKEN_TTL, 600)
        self.assertEqual(guard.SEAL_KEY, "txo-fabric-seal")

    def test_extra_secret_token_esa_or_runtime_object_denied(self):
        docs = self.staged()
        for mutation in (
            lambda obj: obj["metadata"].update(name="default"),
            lambda obj: obj["metadata"].update(namespace="tenant-hairem"),
            lambda obj: obj.update(automountServiceAccountToken=True),
            lambda obj: obj.update(secrets=[{"name": "privileged-token"}]),
            lambda obj: obj["metadata"].update(annotations={"allow":"privileged"}),
        ):
            altered = copy.deepcopy(docs)
            mutation(altered[0])
            self.assertTrue(guard.validate_staged_seal_identity(altered))
        self.assertTrue(guard.validate_staged_seal_identity(docs + docs))
        self.assertTrue(guard.validate_staged_seal_identity(
            docs + [{"kind":"ExternalSecret"}],
        ))

    def test_preflight_incomplete_false_or_bogus_claim_never_authorizes(self):
        facts = {
            "infraTLS": True,
            "infraCA": True,
            "infraSNI": True,
            "tokenReview": True,
            "saAudience": True,
            "transitKeyPolicy": True,
            "periodicTokenRenewal": True,
            "exactPrivateEgress": True,
            "raftStorageContinuity": True,
            "originalShamirCustody": True,
            "explicitPhysicalApproval": True,
        }
        # This is a schema test only, NEVER evidence those facts exist.
        self.assertEqual(guard.require_seal_migration_preflight(facts), [])
        for key in facts:
            modified = dict(facts)
            modified[key] = False
            self.assertTrue(guard.require_seal_migration_preflight(modified))
            modified[key] = "true"
            self.assertTrue(guard.require_seal_migration_preflight(modified))
            modified.pop(key)
            self.assertTrue(guard.require_seal_migration_preflight(modified))
        self.assertTrue(guard.require_seal_migration_preflight({}))

    def test_staged_sa_not_selected_by_any_base_or_txo_overlay(self):
        root = Path(__file__).resolve().parents[2] / (
            "apps/60-services/txo-fabric"
        )
        for f in (
            "openbao-fabric/base/kustomization.yaml",
            "openbao-fabric/overlays/dev/kustomization.yaml",
            "openbao-fabric/overlays/prod/kustomization.yaml",
            "overlays/dev/kustomization.yaml",
            "overlays/prod/kustomization.yaml",
        ):
            contents = (root / f).read_text(encoding="utf-8")
            self.assertNotIn("transit-staged", contents)
        stage = (root / "openbao-fabric/transit-staged/kustomization.yaml").read_text(
            encoding="utf-8"
        )
        self.assertIn("serviceaccount.yaml", stage)
        self.assertNotIn("../../base", stage)


class InfraTransitSourceContractTests(unittest.TestCase):
    @staticmethod
    def policy_text():
        path = Path(__file__).resolve().parents[2] / guard.INFRA_POLICY
        return path.read_text(encoding="utf-8")

    def test_exact_narrow_non_executable_policy_example(self):
        self.assertEqual(guard.validate_infra_transit_policy(self.policy_text()), [])
        self.assertEqual(guard.SEAL_SA, "openbao-fabric-seal-bootstrap")
        self.assertEqual(guard.SEAL_NAMESPACE, "txo-fabric-system")
        self.assertEqual(guard.SEAL_AUDIENCE, "txo-fabric-infra-seal")
        self.assertEqual(guard.SEAL_TOKEN_TTL, 600)
        self.assertEqual(guard.SEAL_KEY, "txo-fabric-seal")

    def test_broad_or_wrong_transit_policy_is_denied(self):
        for mutation in (
            ('capabilities = ["update"]', 'capabilities = ["read"]', 1),
            ('transit/decrypt/txo-fabric-seal', 'transit/decrypt/other-key', 1),
            ('\n', '\npath "sys/policies" {\n  capabilities = ["update"]\n}\n', 1),
        ):
            text = self.policy_text()
            old, new, count = mutation
            if old == '\n':
                text += new
            else:
                text = text.replace(old, new, count)
            self.assertTrue(guard.validate_infra_transit_policy(text))

    def test_evidence_and_example_reject_material_bypass_and_resources(self):
        examples = (
            'token = "not-allowed"',
            'tls_skip_verify = true',
            'endpoint = "http://fallback.invalid"',
            'cidr = "0.0.0.0/0"',
            'kind: Deployment\napiVersion: apps/v1',
        )
        for content in examples:
            with self.subTest(content=content.splitlines()[0]):
                self.assertTrue(guard.assert_infra_evidence_safe(content))
        self.assertTrue(guard.assert_infra_contract_safe(
            guard.INFRA_POLICY, 'path "transit/*" {\n capabilities = ["update"]\n}\n'
        ))

    def test_legacy_eso_binding_has_a_dedicated_unchanged_target(self):
        self.assertEqual(
            guard.LEGACY_INFRA_STORE,
            "apps/00-infra/openbao/base/cluster-secret-store.yaml",
        )


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

    def test_unhashable_action_and_unknown_payloads_fail_closed(self):
        request, canonical = self.fixture()
        request["action"] = ["Ensure"]
        self.assertEqual(
            guard.validate_provisioning_request(request, canonical),
            ["InvalidAction"],
        )
        request["action"] = {"unexpected": "Ensure"}
        self.assertEqual(
            guard.validate_provisioning_request(request, canonical),
            ["InvalidAction"],
        )
        self.assertEqual(
            guard.validate_provisioning_request("browser-input", canonical),
            ["InvalidContract"],
        )

    def test_new_infra_eso_binding_is_still_prohibited(self):
        self.assertTrue(guard.INFRA_STORE_ADDITION.search(
            "+        name: openbao\n",
        ))
        self.assertFalse(guard.INFRA_STORE_ADDITION.search(
            "+        name: openbao-fabric-store\n",
        ))



POC_FILE = Path(__file__).with_name("validate-openbao-fabric-poc.py")
POC_SPEC = importlib.util.spec_from_file_location("openbao_fabric_poc", POC_FILE)
assert POC_SPEC and POC_SPEC.loader
poc = importlib.util.module_from_spec(POC_SPEC)
POC_SPEC.loader.exec_module(poc)


class PrivatePOCTests(unittest.TestCase):
    @staticmethod
    def fixture():
        base = POC_FILE.parents[2] / "apps/60-services/txo-fabric/openbao-fabric/base"
        files = ("configmap.yaml", "statefulset.yaml", "service.yaml", "networkpolicy.yaml",
                 "certificate.yaml")
        return [resource for file in files
                for resource in yaml.safe_load_all((base / file).read_text(encoding="utf-8"))]

    def test_exact_private_poc(self):
        self.assertEqual(poc.check_render(self.fixture()), [])

    def test_image_proven_nonroot_identity_and_immutable_hardening(self):
        manifests = self.fixture()
        sts = next(d for d in manifests if d["kind"] == "StatefulSet")
        pod = sts["spec"]["template"]["spec"]
        self.assertEqual(pod["securityContext"]["runAsNonRoot"], True)
        self.assertEqual(pod["securityContext"]["runAsUser"], 100)
        self.assertEqual(pod["securityContext"]["runAsGroup"], 1000)
        self.assertEqual(pod["securityContext"]["fsGroup"], 1000)
        self.assertEqual(pod["securityContext"]["fsGroupChangePolicy"], "OnRootMismatch")
        self.assertEqual(pod["securityContext"]["seccompProfile"], {"type": "RuntimeDefault"})
        container = pod["containers"][0]
        self.assertFalse(container["securityContext"]["allowPrivilegeEscalation"])
        self.assertTrue(container["securityContext"]["readOnlyRootFilesystem"])
        self.assertEqual(container["securityContext"]["capabilities"]["drop"], ["ALL"])
        self.assertIn({"name": "scratch", "mountPath": "/tmp"}, container["volumeMounts"])
        self.assertEqual(poc.check_render(manifests), [])

    def test_missing_non_numeric_or_wrong_image_identity_fails_closed(self):
        for key, invalid in (
            ("runAsUser", None),
            ("runAsUser", "openbao"),
            ("runAsUser", "100"),
            ("runAsUser", 0),
            ("runAsUser", 101),
            ("runAsUser", True),
            ("runAsGroup", None),
            ("runAsGroup", "1000"),
            ("runAsGroup", 0),
            ("runAsGroup", 1001),
            ("fsGroup", None),
            ("fsGroup", "1000"),
            ("fsGroup", 0),
        ):
            with self.subTest(field=key, value=invalid):
                manifests = copy.deepcopy(self.fixture())
                sts = next(d for d in manifests if d["kind"] == "StatefulSet")
                context = sts["spec"]["template"]["spec"]["securityContext"]
                if invalid is None:
                    del context[key]
                else:
                    context[key] = invalid
                errors = poc.check_render(manifests)
                self.assertTrue(any("numeric UID 100" in msg for msg in errors), errors)

    def test_internal_certificate_chain_and_https_tls_mount(self):
        manifests = self.fixture()
        self.assertEqual(len(manifests), 8)
        self.assertEqual(poc.check_render(manifests), [])
        chain = {(d["kind"], d["metadata"]["name"]): d for d in manifests}
        leaf = chain[("Certificate", "openbao-fabric-tls")]
        self.assertEqual(leaf["spec"]["dnsNames"], poc.SERVICE_DNS)
        self.assertEqual(leaf["spec"]["secretName"], "openbao-fabric-tls")
        sts = chain[("StatefulSet", "openbao-fabric")]
        pod = sts["spec"]["template"]["spec"]
        self.assertEqual(pod["containers"][0]["readinessProbe"]["httpGet"]["scheme"], "HTTPS")
        self.assertEqual(
            next(v for v in pod["volumes"] if v["name"] == "tls")["secret"]["secretName"],
            "openbao-fabric-tls",
        )

    def test_invalid_certificate_sans_issuer_and_inline_material_fail_closed(self):
        for mutate in (
            lambda c: c["spec"]["dnsNames"].append("external.example.invalid"),
            lambda c: c["spec"]["dnsNames"].pop(),
            lambda c: c["spec"]["issuerRef"].update(name="letsencrypt-prod"),
            lambda c: c["spec"].update({"data": {"tls.crt": "inline-not-allowed"}}),
            lambda c: c["spec"].update(secretName="wrong-secret"),
        ):
            manifests = copy.deepcopy(self.fixture())
            leaf = next(d for d in manifests
                        if (d["kind"], d["metadata"]["name"]) ==
                        ("Certificate", "openbao-fabric-tls"))
            mutate(leaf)
            self.assertTrue(poc.check_render(manifests))
        manifests = copy.deepcopy(self.fixture())
        issuer = next(d for d in manifests if d["metadata"]["name"] ==
                      "openbao-fabric-selfsigned")
        issuer["spec"] = {"acme": {"server": "https://public.example.invalid"}}
        self.assertTrue(poc.check_render(manifests))

    def test_runtime_auth_env_sa_privileges_and_bypass_annotations_denied(self):
        for mutation in ("env", "envFrom", "privileged", "root", "seccomp",
                         "pod-annotation", "host-network", "service-links"):
            docs = copy.deepcopy(self.fixture())
            sts = next(d for d in docs if d["kind"] == "StatefulSet")
            pod = sts["spec"]["template"]["spec"]
            container = pod["containers"][0]
            if mutation == "env":
                container["env"] = [{"name": "WORKLOAD_JWT", "value": "unused"}]
            elif mutation == "envFrom":
                container["envFrom"] = [{"secretRef": {"name": "unapproved"}}]
            elif mutation == "privileged":
                container["securityContext"]["privileged"] = True
            elif mutation == "root":
                pod["securityContext"]["runAsUser"] = 0
            elif mutation == "seccomp":
                pod["securityContext"]["seccompProfile"]["type"] = "Unconfined"
            elif mutation == "pod-annotation":
                sts["spec"]["template"]["metadata"]["annotations"] = {"bypass": "true"}
            elif mutation == "host-network":
                pod["hostNetwork"] = True
            else:
                pod["enableServiceLinks"] = True
            with self.subTest(mutation=mutation):
                self.assertTrue(poc.check_render(docs))

    def test_http_disabled_tls_and_inline_config_rejected(self):
        for extra in ("tls_disable = 1", "disable_mlock = true",
                      'api_addr = "http://127.0.0.1:8200"',
                      "-----BEGIN CERTIFICATE-----",
                      "private_key ="):
            manifests = copy.deepcopy(self.fixture())
            cm = next(d for d in manifests if d["kind"] == "ConfigMap")
            cm["data"]["server.hcl"] += "\n    " + extra
            self.assertTrue(poc.check_render(manifests), extra)
        manifests = copy.deepcopy(self.fixture())
        sts = next(d for d in manifests if d["kind"] == "StatefulSet")
        sts["spec"]["template"]["spec"]["containers"][0]["readinessProbe"]["httpGet"]["scheme"] = "HTTP"
        self.assertTrue(poc.check_render(manifests))

    def test_tls_secret_volume_restrictions_and_no_new_client(self):
        for change in ("writable", "wrong-secret", "hostpath", "sa", "new-secret", "job"):
            manifests = copy.deepcopy(self.fixture())
            sts = next(d for d in manifests if d["kind"] == "StatefulSet")
            pod = sts["spec"]["template"]["spec"]
            if change == "writable":
                next(v for v in pod["containers"][0]["volumeMounts"]
                     if v["name"] == "tls")["readOnly"] = False
            elif change == "wrong-secret":
                next(v for v in pod["volumes"]
                     if v["name"] == "tls")["secret"]["secretName"] = "openbao-infra"
            elif change == "hostpath":
                pod["volumes"].append({"name": "insecure", "hostPath": {"path": "/tmp"}})
            elif change == "sa":
                pod["serviceAccountName"] = "workload-auth"
            elif change == "new-secret":
                manifests.append({"apiVersion": "v1", "kind": "Secret",
                                  "metadata": {"name": "forbidden", "namespace": "txo-fabric-system"}})
            else:
                manifests.append({"apiVersion": "batch/v1", "kind": "Job",
                                  "metadata": {"name": "bootstrap", "namespace": "txo-fabric-system"}})
            with self.subTest(change=change):
                self.assertTrue(poc.check_render(manifests))

    def test_wrong_service_type_and_client_ingress_denied(self):
        for public in ("NodePort", "LoadBalancer"):
            manifests = copy.deepcopy(self.fixture())
            next(d for d in manifests if d["kind"] == "Service")["spec"]["type"] = public
            self.assertTrue(poc.check_render(manifests))
        manifests = copy.deepcopy(self.fixture())
        next(d for d in manifests if d["kind"] == "NetworkPolicy")["spec"]["ingress"] = [
            {"from": [{"podSelector": {}}]}
        ]
        self.assertTrue(poc.check_render(manifests))

    def test_delete_storage_or_multiple_replicas_denied(self):
        manifests = copy.deepcopy(self.fixture())
        sts = next(d for d in manifests if d["kind"] == "StatefulSet")
        sts["spec"]["volumeClaimTemplates"][0]["spec"]["storageClassName"] = "storage-rwo-delete"
        self.assertTrue(poc.check_render(manifests))
        manifests = copy.deepcopy(self.fixture())
        sts = next(d for d in manifests if d["kind"] == "StatefulSet")
        sts["spec"]["replicas"] = 2
        self.assertTrue(poc.check_render(manifests))

    def test_bootstrap_and_infra_credentials_denied(self):
        manifests = copy.deepcopy(self.fixture())
        sts = next(d for d in manifests if d["kind"] == "StatefulSet")
        sts["spec"]["template"]["spec"]["initContainers"] = [{"name": "bootstrap"}]
        self.assertTrue(poc.check_render(manifests))
        manifests = copy.deepcopy(self.fixture())
        sts = next(d for d in manifests if d["kind"] == "StatefulSet")
        sts["spec"]["template"]["spec"]["automountServiceAccountToken"] = True
        self.assertTrue(poc.check_render(manifests))
        manifests = copy.deepcopy(self.fixture())
        cm = next(d for d in manifests if d["kind"] == "ConfigMap")
        cm["data"]["server.hcl"] += "\nroot_" + "token = fake-example-value\n"
        self.assertTrue(poc.check_render(manifests))
        manifests = copy.deepcopy(self.fixture())
        manifests.append({"kind": "ExternalSecret", "metadata": {
            "name": "openbao-fabric-store", "namespace": "txo-fabric-system",
            "labels": {"app.kubernetes.io/name": "openbao-fabric"}}})
        self.assertTrue(poc.check_render(manifests))

    def test_txo_selected_requires_default_deny(self):
        self.assertTrue(poc.check_render(self.fixture(), selected=True))
        manifests = copy.deepcopy(self.fixture())
        manifests.append({"kind": "NetworkPolicy", "metadata": {
            "name": "default-deny", "namespace": "txo-fabric-system"},
            "spec": {"podSelector": {}, "policyTypes": ["Ingress", "Egress"]}})
        self.assertEqual(poc.check_render(manifests, selected=True), [])


if __name__ == "__main__":
    unittest.main()
