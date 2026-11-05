#!/usr/bin/env python3
"""Source-only production render gate for the one reviewed private Hermes canary.

The operator may create runtime resources from AgentIdentity intent. This guard
does not attest live absence of credentials, listeners or Services.
"""

from __future__ import annotations

import argparse
from pathlib import Path
import re

import yaml

CANARY_NAME = "ten00001-usr000001-agt00002"
CANARY_TENANT = "hairem"
CANARY_TENANT_ID = "TEN00001"
CANARY_AGENT_KEY = "usr000001-agt00002"
CANARY_CREDENTIAL = "hermes-api-" + CANARY_AGENT_KEY

# Runtime resources selected directly through GitOps must remain free of an
# unrelated/private API activation or the dormant chat persistence schema.
FORBIDDEN_RUNTIME = re.compile(
    r"API_SERVER_ENABLED|API_SERVER_KEY|hermes-api-|privateAPI:"
    r"|(^|[^0-9])8642([^0-9]|$)"
    r"|003_fabric_chat_threads|txo_fabric_chat_threads|txo_fabric_chat_turns",
    re.MULTILINE,
)
FORBIDDEN_RESOURCE_NAMES = re.compile(
    r"name: (?:hermes-private-api|hermes-api-server|txo-fabric-chat-runtime-credentials)(?:\s|$)",
    re.MULTILINE,
)


def check_rendered_private_api_contract(documents: list[object]) -> list[str]:
    violations: list[str] = []
    found_canary = 0
    seen_agents: set[str] = set()

    for index, document in enumerate(documents, start=1):
        if document is None:
            continue
        if not isinstance(document, dict):
            violations.append(f"document {index}: expected a Kubernetes mapping")
            continue
        kind = document.get("kind")
        if kind == "CustomResourceDefinition":
            # CRD API vocabulary is a schema, not a selected runtime resource.
            continue
        if kind == "AgentIdentity":
            name = (document.get("metadata") or {}).get("name")
            if not isinstance(name, str) or not name:
                violations.append(f"document {index}: AgentIdentity has no name")
                continue
            if name in seen_agents:
                violations.append(f"AgentIdentity {name}: duplicate selected declaration")
            seen_agents.add(name)
            spec = document.get("spec") or {}
            if not isinstance(spec, dict):
                violations.append(f"AgentIdentity {name}: invalid spec")
                continue
            runtime = spec.get("runtime") or {}
            if not isinstance(runtime, dict):
                violations.append(f"AgentIdentity {name}: invalid runtime")
                continue
            api = runtime.get("privateAPI")
            if api is None:
                if name == CANARY_NAME:
                    violations.append(f"AgentIdentity {name}: reviewed canary opt-in missing")
                continue
            if not isinstance(api, dict):
                violations.append(f"AgentIdentity {name}: invalid privateAPI declaration")
                continue
            if set(api) - {"enabled", "tenantId", "credentialRef"}:
                violations.append(f"AgentIdentity {name}: unsupported privateAPI fields")
            enabled = api.get("enabled", False)
            if not isinstance(enabled, bool):
                violations.append(f"AgentIdentity {name}: privateAPI.enabled must be boolean")
                continue
            if not enabled:
                if api.get("tenantId") or api.get("credentialRef"):
                    violations.append(f"AgentIdentity {name}: disabled API has binding fields")
                if name == CANARY_NAME:
                    violations.append(f"AgentIdentity {name}: reviewed canary disabled")
                continue
            if name != CANARY_NAME:
                violations.append(f"AgentIdentity {name}: unreviewed privateAPI opt-in")
                continue
            found_canary += 1
            if (
                (spec.get("tenantRef") or {}).get("name") != CANARY_TENANT
                or spec.get("agentKey") != CANARY_AGENT_KEY
                or api.get("tenantId") != CANARY_TENANT_ID
                or api.get("credentialRef", "") not in ("", CANARY_CREDENTIAL)
            ):
                violations.append("reviewed Electra canary has noncanonical tenant, agent or credential binding")
            continue

        # The objects below are actually selected GitOps resources; the
        # operator-derived resources from the canary are NOT in this render.
        rendered = yaml.safe_dump(document, sort_keys=False)
        if FORBIDDEN_RUNTIME.search(rendered):
            violations.append(f"document {index} ({kind}): forbidden directly selected API/chat runtime")
        if FORBIDDEN_RESOURCE_NAMES.search(rendered):
            violations.append(f"document {index} ({kind}): forbidden private API/credential resource")

    if found_canary != 1:
        violations.append(f"expected exactly one enabled reviewed Electra canary, got {found_canary}")
    return violations


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("render", type=Path, help="Kustomize production YAML stream")
    args = parser.parse_args()
    with args.render.open(encoding="utf-8") as source:
        violations = check_rendered_private_api_contract(list(yaml.safe_load_all(source)))
    if violations:
        for violation in violations:
            print(f"::error::{violation}")
        return 1
    print("PASS: BFF/chat runtime remains unselected; Electra is the only private API opt-in")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
