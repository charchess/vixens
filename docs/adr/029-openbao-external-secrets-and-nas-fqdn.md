# ADR-029: OpenBao / External Secrets and NAS FQDN

**Date:** 2026-09-18  
**Status:** Accepted  
**Deciders:** Vixens maintainers  
**Tags:** security, secrets, storage, dns, portability  
**Supersedes:** ADR-011 and the historical duplicate-number record `018-openbao-external-secrets-and-nas-fqdn.md`

## Context

On 2026-09-12 the repository recorded the move away from the Infisical Operator toward External Secrets backed by OpenBao, together with the decision to use a stable NAS FQDN. That decision was accidentally stored under ADR number 018, which already belonged to the historical Netbird deployment decision.

ADR records are historical and must not be rewritten merely to repair numbering or reflect later state. The original duplicate-number file is therefore preserved verbatim.

## Decision

The canonical current architecture is:

- retire the Infisical Operator from desired state;
- use External Secrets Operator for Kubernetes secret materialization;
- use the `openbao` `ClusterSecretStore` for workload `ExternalSecret` resources;
- keep secret payloads outside Git;
- use the canonical NAS FQDN for mutable NAS-hosted endpoints rather than embedding a replaceable storage-server address in application manifests.

Legacy filenames containing `infisical` may remain temporarily when renaming would create noisy manifest churn. The resource API and behaviour, not the filename, determine the active architecture.

## Consequences

- New documentation and examples use External Secrets/OpenBao terminology.
- Active manifests must not contain `InfisicalSecret` resources or the retired `secrets.infisical.com` API.
- The original ADR-011 and duplicate ADR-018 remain historical records.
- A secret bootstrap mechanism is still required outside Git so External Secrets can authenticate to OpenBao.
