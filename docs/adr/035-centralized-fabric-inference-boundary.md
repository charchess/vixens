# ADR-035: Centralize Fabric external inference behind the TXO AI Gateway

**Date:** 2026-10-02  
**Status:** Accepted  
**Scope:** AIaaS  
**Related Project:** vixens roadmap  
**Related Issues:** #3710, #3674, #3716, #3607, #3649, #3685  
**Deciders:** Vixens maintainers  
**Tags:** txo-fabric, inference, llm, embeddings, gateway, credentials, network-policy, retrospective

> **Retrospective ADR:** this records the centralized inference contract already implemented and physically accepted under #3607, #3649, #3685, #3674 and #3716 before this ADR was written.

## Context

TXO Fabric runs tenant workloads that may need externally hosted LLM or embedding inference. Giving every Hermes agent, tenant Hindsight instance or future module a provider master credential would distribute sensitive platform credentials into tenant namespaces, make revocation and provider changes difficult, and permit application-local configuration to bypass platform policy.

The accepted Fabric implementation instead introduced a proprietor-managed AI gateway and then proved the two active external-inference paths:

- Hermes chat/model inference;
- tenant Hindsight remote embeddings.

The implementation also established credential rotation/revocation, model allowlists, network-policy enforcement and non-secret observability. Those behaviors now form a durable platform boundary rather than an application-specific convention.

Concrete gateway software, upstream provider names, model versions and pricing remain mutable implementation details. This ADR records the ownership and security contract that those implementations must preserve.

## Decision

### One platform-owned external-inference boundary

Externally hosted LLM and embedding inference used by Fabric-managed tenant workloads goes through the TXO AI Gateway when the capability is supported by that boundary.

Tenant workloads do not receive upstream provider master credentials merely to perform external inference.

Explicit local/offline inference may remain outside the gateway when it is intentionally selected and documented as a different execution boundary. It must not become an implicit fallback for a failed gateway request.

### Stable logical model aliases

Fabric workloads address platform-owned logical capabilities rather than concrete provider/model identifiers.

The accepted v0 contract includes separate logical aliases for general model inference and embeddings. Concrete upstream routing is owned by gateway configuration so a provider/model change does not require editing every TenantBundle or AgentIdentity.

A future capability may add another purpose-specific logical alias when policy isolation requires it.

### Scoped workload credentials

Gateway access uses independently scoped and revocable workload credentials:

- Hermes receives a credential scoped to an AgentIdentity and its allowed model capability;
- tenant Hindsight receives a separate credential scoped to the TenantBundle embedding capability;
- future Fabric-managed external-inference consumers must receive an equivalently scoped authorization rather than a provider master credential.

Gateway administrative credentials and upstream provider credentials remain on the platform side of the boundary.

The Hindsight service API credential is a separate authorization domain from its gateway embedding credential. Rotating one must not implicitly rotate the other.

### Credential lifecycle is independent from private runtime state

Scoped inference authorization is platform lifecycle state, not agent-owned durable state under `/opt/data`.

The platform must be able to revoke or rotate gateway access without copying credentials through retained PVCs. Accepted rotation semantics are fail-closed and edge-triggered: the previous credential is revoked before the replacement becomes authoritative, and removing a completed rotation request does not rotate back or create another credential.

Deletion or credential replacement must not leave a deliberately reusable old credential under the same workload identity.

### No silent direct-provider bypass

A gateway-backed workload does not silently fall back to a direct provider credential or unrestricted provider route when the gateway or scoped credential is unavailable.

Network policy and runtime configuration should make the intended path difficult to bypass by construction. For the accepted Hermes and Hindsight paths, workload egress permits the internal gateway flows required for brokered inference rather than generic direct-provider access.

Future integration/egress policy may add other explicit Internet access for unrelated capabilities; that does not grant permission to bypass the inference boundary.

### Observable policy without secret disclosure

The effective inference binding must be inspectable without exposing credential material.

Runtime or Fabric status may expose non-secret information such as:

- gateway endpoint identity;
- logical model/capability;
- applied credential-rotation revision;
- readiness/reconciliation state.

Raw scoped credentials, gateway administrative credentials and provider credentials must not be copied into CRD status, labels, annotations, logs or Git-based acceptance evidence.

### Hindsight generative LLM remains disabled until governed

Tenant Hindsight generative/reflection LLM processing remains disabled by default until it has an explicitly implemented gateway-backed logical model and scoped credential contract.

Embedding access through the gateway does not implicitly authorize Hindsight generative model access.

### Budgets and rate limits remain platform policy

Budget, RPM and TPM controls belong to the platform inference boundary rather than arbitrary per-agent mutable configuration.

For the accepted Client 0 v0 contract, those numeric quotas are deliberately unset while model allowlists and independently revocable scoped credentials are enforced. Future quota policy can be added centrally without changing this ownership model.

## Consequences

### Positive

- upstream provider credentials stay outside tenant agent/memory workloads;
- credentials can be rotated or revoked independently per workload/capability;
- provider/model routing can change centrally behind stable workload-facing aliases;
- model allowlists prevent an embedding credential from becoming a general chat credential, and vice versa;
- retained private agent state is insufficient to recover revoked inference access;
- network and runtime configuration enforce the intended inference path rather than relying only on prompts or operator convention;
- status can show the effective binding without leaking secrets;
- future Fabric modules have a clear contract to join instead of inventing their own provider-auth pattern.

### Negative

- externally hosted inference depends on the availability and capacity of the central gateway;
- the platform owns gateway credential lifecycle, routing policy, metering and operational recovery;
- gateway-backed inference still sends applicable customer data to the selected upstream provider; centralization does not make offshore processing local or remove privacy/subprocessor obligations;
- fail-closed rotation may temporarily block inference when replacement provisioning fails;
- future consumers that cannot use the gateway contract need an explicit architectural exception rather than an application-local workaround.

## Alternatives considered

### Put provider API keys directly in each tenant workload

Rejected because it spreads provider credentials into tenant namespaces, couples workloads to concrete providers and weakens independent revocation and auditability.

### Share one gateway credential across every tenant/agent

Rejected because compromise or revocation would have an unnecessarily large blast radius and model/capability policy could not be attributed cleanly to one workload identity.

### Configure concrete provider/model names in TenantBundle or AgentIdentity

Rejected because provider selection is a platform policy concern and should be changeable centrally without rewriting tenant intent.

### Allow a direct-provider fallback when the gateway fails

Rejected because a fallback would bypass the same credential, routing, accounting and network policy that the gateway exists to enforce.

### Freeze arbitrary v0 budget/RPM/TPM numbers into the Fabric API

Rejected because no operating policy justified those values. Quotas remain platform-owned and can be introduced centrally when requirements are real.

## Physical acceptance evidence

The accepted contract was established incrementally:

- #3607 physically proved Hermes model inference through the local gateway with a scoped workload credential and metering;
- #3649 physically proved tenant Hindsight embeddings through the same platform boundary with scoped attribution and the steady-state embedding path;
- #3685 revalidated the Hindsight/Hermes service and credential boundary, including no PostgreSQL credential exposure to Hermes and generative LLM disabled;
- #3716 physically proved independent Hermes and Hindsight credential rotation/revocation, old-credential rejection, workload-specific model allowlists, successful post-rotation inference, non-secret status visibility and edge-trigger behavior;
- #3674 consolidated those implementation and physical proofs as the completed centralized inference objective.

Exact release tags, provider/model selections and operating commands remain living documentation rather than ADR content.

## References

- #3674 — centralized LLM and embedding governance
- #3716 — physical scoped credential rotation/revocation acceptance
- #3607 — TXO AI egress gateway implementation and initial physical acceptance
- #3649 — gateway-backed Hindsight embedding migration and acceptance
- #3685 — Hermes/Hindsight memory and credential boundary acceptance
- #3710 — v0 platform-contract ADR harvesting
- #3714 — scoped gateway credential rotation implementation
- `apps/60-services/txo-fabric/gateway/README.md` — living gateway contract and current implementation details
- `apps/60-services/txo-fabric/operator/` — current reconciliation and policy implementation
