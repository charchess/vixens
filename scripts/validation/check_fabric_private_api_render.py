#!/usr/bin/env python3
"""Generic, fail-closed source-only private Hermes API production render contract.

An eligible AgentIdentity may cause the operator to provision a private API.
Only the operator can attest live Kubernetes ownership, credentials and access.
"""

from __future__ import annotations

import argparse
from pathlib import Path
import re

import yaml

TENANT_ID = re.compile(r"^TEN[0-9]{5,}$")
DNS_NAME = re.compile(r"^[a-z0-9]([a-z0-9-]*[a-z0-9])?$")

# These forms must not be selected *directly* as GitOps runtime resources.
# Operator-derived private API Services/credentials are not in a Kustomize render.
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


def valid_dns_name(value: object, max_length: int = 253) -> bool:
    return isinstance(value, str) and 0 < len(value) <= max_length and bool(DNS_NAME.fullmatch(value))


def check_rendered_private_api_contract(documents: list[object]) -> list[str]:
    violations: list[str] = []
    tenant_ids: dict[str, str] = {}
    seen_tenant_ids: set[str] = set()
    agents: list[tuple[int, dict]] = []
    seen_agents: set[str] = set()
    seen_agent_keys: set[tuple[str, str]] = set()

    for index, doc in enumerate(documents, start=1):
        if doc is None:
            continue
        if not isinstance(doc, dict):
            violations.append(f"document {index}: expected Kubernetes mapping")
            continue
        kind = doc.get("kind")
        if kind == "CustomResourceDefinition":
            # A CRD's declared vocabulary does not create runtime objects.
            continue
        if kind == "TenantBundle":
            metadata, spec = doc.get("metadata"), doc.get("spec")
            if not isinstance(metadata, dict) or not isinstance(spec, dict):
                violations.append(f"document {index}: invalid TenantBundle")
                continue
            name, tenant_id = metadata.get("name"), spec.get("tenantId")
            if not valid_dns_name(name) or not isinstance(tenant_id, str) or not TENANT_ID.fullmatch(tenant_id):
                violations.append(f"document {index}: invalid TenantBundle canonical identity")
                continue
            if name in tenant_ids or tenant_id in seen_tenant_ids:
                violations.append(f"document {index}: ambiguous TenantBundle identity")
            tenant_ids[name] = tenant_id
            seen_tenant_ids.add(tenant_id)
            continue
        if kind == "AgentIdentity":
            agents.append((index, doc))
            continue

        # Selected workloads/Secrets/Services must not bypass the operator.
        rendered = yaml.safe_dump(doc, sort_keys=False)
        if FORBIDDEN_RUNTIME.search(rendered):
            violations.append(f"document {index} ({kind}): forbidden directly selected API/chat runtime")
        if FORBIDDEN_RESOURCE_NAMES.search(rendered):
            violations.append(f"document {index} ({kind}): forbidden private API/credential resource")

    for index, agent in agents:
        metadata, spec = agent.get("metadata"), agent.get("spec")
        if not isinstance(metadata, dict) or not isinstance(spec, dict):
            violations.append(f"document {index}: invalid AgentIdentity")
            continue
        name = metadata.get("name")
        if not valid_dns_name(name):
            violations.append(f"document {index}: invalid AgentIdentity name")
            continue
        if name in seen_agents:
            violations.append(f"AgentIdentity {name}: duplicate selected declaration")
        seen_agents.add(name)
        tenant_ref = spec.get("tenantRef")
        tenant_name = tenant_ref.get("name") if isinstance(tenant_ref, dict) else None
        agent_key = spec.get("agentKey")
        if not valid_dns_name(tenant_name) or tenant_name not in tenant_ids:
            violations.append(f"AgentIdentity {name}: no canonical selected TenantBundle")
        if not valid_dns_name(agent_key, 63):
            violations.append(f"AgentIdentity {name}: invalid canonical agentKey")
            continue
        identity = (tenant_name, agent_key)
        if identity in seen_agent_keys:
            violations.append(f"AgentIdentity {name}: duplicate tenant-local agentKey")
        seen_agent_keys.add(identity)
        runtime = spec.get("runtime") or {}
        if not isinstance(runtime, dict):
            violations.append(f"AgentIdentity {name}: invalid runtime")
            continue
        api = runtime.get("privateAPI")
        if api is None:
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
                violations.append(f"AgentIdentity {name}: disabled API cannot have tenant/credential bindings")
            continue
        expected_id = tenant_ids.get(tenant_name)
        declared_id = api.get("tenantId")
        if expected_id is None or declared_id != expected_id:
            violations.append(f"AgentIdentity {name}: privateAPI tenantId differs from canonical TenantBundle")
        canonical_credential = "hermes-api-" + agent_key
        credential_ref = api.get("credentialRef", "")
        if (len(canonical_credential) > 63 or credential_ref not in ("", canonical_credential)):
            violations.append(f"AgentIdentity {name}: noncanonical privateAPI credentialRef")
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
    print("PASS: private API declarations follow tenant-agnostic contract; BFF/chat runtime unselected")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
