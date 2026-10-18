# Hindsight

> **Scope note:** the former standalone/laboratory Hindsight values have been retired.
> Hindsight is tenant-local and reconciled by TXO Fabric. In Fabric, the logical
> runtime identity is `AgentIdentity`; see
> [ADR-031](../../adr/031-agentidentity-is-the-fabric-runtime-identity.md). The durable
> memory-provider contract remains tracked by #3685 until its implementation/physical
> acceptance is complete.

## Purpose

Hindsight provides long-term memory services for Hermes agents: retain, recall,
consolidation, graph/entity resolution, and reflect synthesis.

## Runtime and inference boundary

Each tenant Hindsight API receives embeddings only through the tenant TXO AI
Gateway / llmproxy, using a Hindsight-scoped virtual credential. Provider
selection, OpenRouter credentials, metering, policy and failover remain owned by
the CPA/gateway; no direct provider endpoint or provider credential is permitted
in Hindsight configuration.

Generative and reflection processing stays disabled with
`HINDSIGHT_API_LLM_PROVIDER=none` until it can use the same CPA boundary. Hermes
uses Hindsight as an external durable-memory service; PostgreSQL credentials stay
inside Hindsight and are never injected into Hermes.

## Validation

Validate a change through GitOps: render the Fabric configuration, wait for the
tenant workload to become healthy, then confirm a retain/recall cycle and
cross-bank isolation. A provider credential or direct Internet egress in a
Hindsight Pod is a blocking architecture defect.
