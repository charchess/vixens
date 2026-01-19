# ADR-021: Netbird Migration to Native Manifests

**Date:** 2026-01-17
**Status:** Accepted
**Deciders:** User, Coding Agent
**Tags:** networking, netbird, kustomize, native

---

## Context
ADR-018 proposed using the `totmicro/netbird` community Helm chart. However, initial deployment attempts encountered stability issues, complexity in patching OIDC dynamically, and difficulties managing specific gRPC/h2c requirements through Helm abstractions.

## Decision
Abandon the community Helm chart in favor of **Native Kubernetes Manifests (Kustomize)**.

1. **Source:** Native manifests manually authored based on official Netbird components.
2. **Persistence:** PostgreSQL (CloudNativePG) for Management backend.
3. **SSO:** Authentik OIDC integration.
4. **Ingress:** Traefik with `h2c` annotation for gRPC support.
5. **Dynamic Config:** Init-container used to generate `management.json` dynamically from environment variables to handle Dev/Prod differences properly.

## Consequences
### Positives ✅
- Total control over resource manifests.
- Easier troubleshooting (no Helm abstraction layer).
- Robust dynamic configuration for OIDC and paths.
- Better alignment with Vixens "Elite" standard.
### Négatives ⚠️
- Increased maintenance burden (no automatic upstream chart updates).

## References
- Supersedes: [ADR-018](018-netbird-deployment-architecture.md)
- Ticket Beads: vixens-yufn
