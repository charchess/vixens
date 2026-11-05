"""Generic private Hermes API declaration regressions, across arbitrary tenants."""

from copy import deepcopy
import importlib.util
from pathlib import Path
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
CHECKER_FILE = ROOT / "scripts/validation/check_fabric_private_api_render.py"
SPEC = importlib.util.spec_from_file_location("fabric_private_api_render", CHECKER_FILE)
CHECKER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CHECKER)


def tenant(name, tenant_id):
    return {"kind": "TenantBundle", "metadata": {"name": name},
            "spec": {"tenantId": tenant_id}}


def agent(name, tenant_name, agent_key, api=None):
    runtime = {}
    if api is not None:
        runtime["privateAPI"] = api
    return {"kind": "AgentIdentity", "metadata": {"name": name},
            "spec": {"tenantRef": {"name": tenant_name},
                     "agentKey": agent_key, "runtime": runtime}}


def check(documents):
    return CHECKER.check_rendered_private_api_contract(documents)


class PrivateAPISourceContractTests(unittest.TestCase):
    def setUp(self):
        self.tenant_a = tenant("alpha", "TEN12345")
        self.tenant_b = tenant("beta", "TEN67890")
        self.agent_a = agent("alpha-robot", "alpha", "robot",
                             {"enabled": True, "tenantId": "TEN12345"})
        self.agent_b = agent("beta-helper", "beta", "helper",
                             {"enabled": True, "tenantId": "TEN67890",
                              "credentialRef": "hermes-api-helper"})

    def test_zero_one_or_multiple_opt_ins_use_the_same_contract(self):
        self.assertEqual(check([self.tenant_a, self.tenant_b]), [])
        self.assertEqual(check([self.tenant_a, self.agent_a]), [])
        self.assertEqual(check([self.tenant_a, self.tenant_b, self.agent_a, self.agent_b]), [])
        # A second agent within one tenant does not require a special CI exception.
        other = agent("alpha-second", "alpha", "second",
                      {"enabled": True, "tenantId": "TEN12345"})
        self.assertEqual(check([self.tenant_a, self.agent_a, other]), [])
        # A future tenant must work without knowing any existing customer name.
        self.assertEqual(check([tenant("gamma", "TEN54321"),
                                agent("gamma-assistant", "gamma", "assistant",
                                      {"enabled": True, "tenantId": "TEN54321"})]), [])

    def test_disabled_and_omitted_intent_are_allowed_without_bindings(self):
        self.assertEqual(check([self.tenant_a,
                                agent("alpha-one", "alpha", "one"),
                                agent("alpha-two", "alpha", "two", {"enabled": False})]), [])

    def test_foreign_missing_and_ambiguous_tenants_fail_closed(self):
        initial = [self.tenant_a, self.tenant_b, self.agent_a]
        mutations = [
            ("unknown tenant ref", lambda docs: docs[2]["spec"]["tenantRef"].update(name="gamma")),
            ("foreign tenant ID", lambda docs: docs[2]["spec"]["runtime"]["privateAPI"].update(tenantId="TEN67890")),
            ("missing tenant ID", lambda docs: docs[2]["spec"]["runtime"]["privateAPI"].pop("tenantId")),
            ("duplicated tenant name", lambda docs: docs.append(deepcopy(docs[0]))),
            ("duplicated tenant ID", lambda docs: docs.append(tenant("gamma", "TEN12345"))),
            ("missing TenantBundle", lambda docs: docs.pop(0)),
        ]
        for label, mutation in mutations:
            docs = deepcopy(initial)
            mutation(docs)
            with self.subTest(label=label):
                self.assertTrue(check(docs), label)

    def test_noncanonical_credential_or_agent_identity_fails_closed(self):
        initial = [self.tenant_a, self.agent_a]
        mutations = [
            ("wrong credential", lambda docs: docs[1]["spec"]["runtime"]["privateAPI"].update(
                credentialRef="hermes-api-helper")),
            ("foreign agent key", lambda docs: docs[1]["spec"].update(agentKey="OTHER")),
            ("wrong enabled type", lambda docs: docs[1]["spec"]["runtime"]["privateAPI"].update(enabled="true")),
            ("unsupported API field", lambda docs: docs[1]["spec"]["runtime"]["privateAPI"].update(allowUnscoped=True)),
            ("too long credential ref", lambda docs: docs[1]["spec"].update(agentKey="a" * 55)),
            ("duplicate agent name", lambda docs: docs.append(deepcopy(docs[1]))),
            ("same tenant agent key", lambda docs: docs.append(agent("alpha-other", "alpha", "robot"))),
            ("disabled with binding", lambda docs: docs[1]["spec"]["runtime"]["privateAPI"].update(enabled=False)),
            ("invalid API shape", lambda docs: docs[1]["spec"]["runtime"].update(privateAPI="true")),
        ]
        for label, mutation in mutations:
            docs = deepcopy(initial)
            mutation(docs)
            with self.subTest(label=label):
                self.assertTrue(check(docs), label)

    def test_directly_selected_API_server_service_secret_and_database_are_denied(self):
        for kind, content in (
            ("Deployment", {"spec": {"template": {"spec": {"containers": [
                {"name": "hermes", "env": [{"name": "API_SERVER_ENABLED", "value": "true"}]}]}}}}),
            ("Deployment", {"spec": {"template": {"spec": {"containers": [
                {"name": "hermes", "env": [{"name": "API_SERVER_KEY",
                                             "valueFrom": {"secretKeyRef": {"name": "hermes-api-robot",
                                                                             "key": "api-key"}}}]}]}}}}),
            ("Service", {"spec": {"ports": [{"port": 8642}]}}),
            ("Service", {"metadata": {"name": "hermes-api-server"}}),
            ("Secret", {"metadata": {"name": "txo-fabric-chat-runtime-credentials"}}),
            ("ConfigMap", {"data": {"migration": "003_fabric_chat_threads"}}),
            ("ConfigMap", {"data": {"schema": "txo_fabric_chat_threads"}}),
            ("ConfigMap", {"data": {"schema": "txo_fabric_chat_turns"}}),
            ("ConfigMap", {"data": {"schema": "privateAPI: true"}}),
        ):
            with self.subTest(kind=kind, content=content):
                self.assertTrue(check([self.tenant_a, self.agent_a,
                                       {"kind": kind, **content}]))

    def test_operator_ci_invokes_generic_contract_not_pinned_canary(self):
        """The operator gate must not regress to a named-client allowlist."""
        workflow_path = ROOT / ".github/workflows/txo-fabric-operator-ci.yaml"
        workflow = yaml.safe_load(workflow_path.read_text(encoding="utf-8"))
        steps = workflow["jobs"]["test"]["steps"]
        commands = "\\n".join(str(step.get("run", "")) for step in steps)
        self.assertIn(
            "python3 scripts/validation/check_fabric_private_api_render.py ",
            commands,
        )
        self.assertIn("kustomize build apps/60-services/txo-fabric/overlays/prod", commands)
        self.assertIn(
            "python3 -m unittest discover -s scripts/validation/tests "
            "-p test_fabric_private_api_render.py -v",
            commands,
        )
        self.assertIn("go test -race", commands)
        for retired in (
            "expected exactly three canary API documents",
            "only the approved internal Hermes API canary",
            "ten00001-usr000001-agt00002",
            "hairem-openwebui",
        ):
            with self.subTest(retired=retired):
                self.assertNotIn(retired, commands)


    def test_crd_vocabulary_is_not_runtime_selection(self):
        self.assertEqual(check([
            {"kind": "CustomResourceDefinition", "metadata": {"name": "agents"},
             "spec": {"description": "API_SERVER_ENABLED privateAPI: 8642"}},
            self.tenant_a, self.agent_a,
        ]), [])


if __name__ == "__main__":
    unittest.main()
