# ADR-029: Align application maturity with the current platform

**Date:** 2026-09-25  
**Status:** Accepted  
**Deciders:** Vixens maintainers  
**Tags:** maturity, resources, secrets, openbao, external-secrets

## Context

ADR-023 remains the active definition of the seven application maturity tiers, but two implementation details in its original March 2026 wording no longer match the platform:

1. Silver names **Infisical** as the required secret backend.
2. Some derived documentation interpreted sizing labels as meaning that explicit Kubernetes `resources.requests` / `resources.limits` must never appear in manifests.

The current platform uses **OpenBao + External Secrets Operator (ESO)**. Current workload manifests and `RESOURCE_STANDARDS.md` also use explicit resource requests/limits as a bootstrap-safe baseline while sizing labels, Goldilocks and VPA provide policy integration and tuning.

Keeping obsolete implementation names inside active standards causes humans and agents to recreate retired patterns.

## Decision

### 1. Secret criterion

The ADR-023 Silver criterion:

```text
Secrets via Infisical
```

is amended to the implementation-independent criterion:

```text
Secrets are externally managed; no secret value is stored in Git.
```

The canonical Vixens implementation is currently:

```text
OpenBao
  ↓
ClusterSecretStore/openbao
  ↓
ExternalSecret (external-secrets.io/v1)
  ↓
Kubernetes Secret
  ↓
workload
```

The detailed secret architecture is documented in `docs/adr/018-openbao-external-secrets-and-nas-fqdn.md` and `docs/guides/secret-management.md`.

Historical Infisical ADRs, audits and incident reports remain valid historical records but are not implementation guidance.

### 2. Resource criterion

The maturity model continues to require:

- **Bronze:** explicit CPU/memory requests;
- **Silver:** explicit CPU/memory limits;
- higher tiers: sizing justification, observation and tuning as defined by ADR-023.

Sizing metadata such as `vixens.io/sizing.<container>` complements the Kubernetes resources block; it does not prohibit it.

The exact resource values and current tier conventions are defined by `docs/reference/RESOURCE_STANDARDS.md`, current shared policies/components, and comparable current workloads.

### 3. Sources of truth

For application work, use this precedence:

1. `AGENTS.md` / `WORKFLOW.md` for change workflow;
2. ADR-023 as amended by this ADR for maturity intent;
3. `docs/reference/RESOURCE_STANDARDS.md` for resource conventions;
4. `docs/guides/secret-management.md` for secret architecture;
5. `docs/procedures/deployment-standard.md` and current manifests for deployment patterns.

`apps/template-app/` is an example implementation, not an independent architecture authority.

## Consequences

### Positive

- no active standard instructs contributors to create new Infisical resources;
- resource fallback manifests and sizing policy no longer contradict each other;
- agent guidance can be smaller and refer to canonical docs;
- future secret backends can change without rewriting the maturity tier itself.

### Negative

- historical documents will still contain Infisical terminology by design;
- exact sizing values still require reading current policies/manifests because they may evolve.

## Compatibility

This ADR does **not** change maturity tier names, tier ordering, application runtime state, OpenBao paths, Secret names, Kubernetes policies or production deployment refs. It only amends the interpretation of two active documentation rules.
