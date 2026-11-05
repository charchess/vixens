"""Fail-closed source regressions for the reviewed private Hermes Electra canary."""

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


def agent(name="ten00001-usr000001-agt00002", tenant="hairem", key="usr000001-agt00002",
          api=None):
    if api is None:
        api = {"enabled": True, "tenantId": "TEN00001"}
    return {
        "apiVersion": "fabric.truxonline.io/v1alpha1",
        "kind": "AgentIdentity",
        "metadata": {"name": name},
        "spec": {"tenantRef": {"name": tenant}, "agentKey": key,
                 "runtime": {"privateAPI": api}},
    }


def selected(docs):
    return CHECKER.check_rendered_private_api_contract(docs)


class PrivateHermesCanarySourceTests(unittest.TestCase):
    def test_reviewed_canary_and_unselected_runtime_pass(self):
        documents = [
            {"kind": "CustomResourceDefinition", "metadata": {"name": "agentidentities"},
             "spec": {"description": "privateAPI: API_SERVER_ENABLED 8642"}},
            agent(),
            agent(name="tenant-c-other", tenant="foreign", key="other",
                  api={"enabled": False}),
            {"kind": "Deployment", "metadata": {"name": "webui"},
             "spec": {"template": {"spec": {"containers": [{"name": "webui",
                     "env": [{"name": "TXO_WEBUI_BFF_BROWSER_ENABLED", "value": "false"}]}]}}}},
        ]
        self.assertEqual(selected(documents), [])
        explicit_credential = deepcopy(documents)
        explicit_credential[1]["spec"]["runtime"]["privateAPI"]["credentialRef"] = (
            "hermes-api-usr000001-agt00002"
        )
        self.assertEqual(selected(explicit_credential), [])

    def test_second_opt_in_foreign_tenant_agent_and_credential_fail(self):
        initial = [agent()]
        mutations = (
            ("second opt-in", lambda docs: docs.append(
                agent(name="other-tenant-agent", tenant="other", key="other",
                      api={"enabled": True, "tenantId": "TEN99999"}))),
            ("foreign tenant ref", lambda docs: docs[0]["spec"]["tenantRef"].update(name="foreign")),
            ("foreign tenant id", lambda docs: docs[0]["spec"]["runtime"]["privateAPI"].update(tenantId="TEN99999")),
            ("foreign agent key", lambda docs: docs[0]["spec"].update(agentKey="other")),
            ("foreign credential", lambda docs: docs[0]["spec"]["runtime"]["privateAPI"].update(credentialRef="hermes-api-other")),
            ("wrong enabled type", lambda docs: docs[0]["spec"]["runtime"]["privateAPI"].update(enabled="true")),
            ("unexpected API field", lambda docs: docs[0]["spec"]["runtime"]["privateAPI"].update(allowAnyTenant=True)),
            ("disabled with credential", lambda docs: docs[0]["spec"]["runtime"]["privateAPI"].update(enabled=False, credentialRef="hermes-api-other")),
            ("duplicate canary", lambda docs: docs.append(deepcopy(docs[0]))),
            ("no canary", lambda docs: docs.clear()),
        )
        for label, change in mutations:
            docs = deepcopy(initial)
            change(docs)
            with self.subTest(label=label):
                self.assertTrue(selected(docs), label)

    def test_directly_selected_api_and_chat_persistence_fail(self):
        for kind, content in (
            ("Deployment", {"spec": {"template": {"spec": {"containers": [{
                "name": "hermes", "env": [{"name": "API_SERVER_ENABLED", "value": "true"}]
            }]}}}}),
            ("Deployment", {"spec": {"template": {"spec": {"containers": [{
                "name": "hermes", "env": [{"name": "API_SERVER_KEY", "valueFrom": {
                    "secretKeyRef": {"name": "hermes-api-usr000001-agt00002", "key": "api-key"}
                }}]
            }]}}}}),
            ("Service", {"spec": {"ports": [{"port": 8642}]}}),
            ("Service", {"metadata": {"name": "hermes-api-server"}}),
            ("Secret", {"metadata": {"name": "txo-fabric-chat-runtime-credentials"}}),
            ("ConfigMap", {"data": {"schema": "003_fabric_chat_threads"}}),
            ("ConfigMap", {"data": {"sql": "txo_fabric_chat_turns"}}),
            ("ConfigMap", {"data": {"sql": "txo_fabric_chat_threads"}}),
            ("ConfigMap", {"data": {"contract": "privateAPI: enabled"}}),
        ):
            with self.subTest(kind=kind, content=content):
                self.assertTrue(selected([agent(), {"kind": kind, **content}]))

    def test_inert_second_agent_must_not_declare_bindings(self):
        for api in ({"tenantId": "TEN99999"}, {"enabled": False, "credentialRef": "hermes-api-foreign"}):
            with self.subTest(api=api):
                self.assertTrue(selected([
                    agent(),
                    agent(name="tenant-other", tenant="foreign", key="other", api=api),
                ]))


if __name__ == "__main__":
    unittest.main()
