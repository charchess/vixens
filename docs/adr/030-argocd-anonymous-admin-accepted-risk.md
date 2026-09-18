# ADR-030: ArgoCD anonymous administrative access — accepted homelab risk

**Date:** 2026-09-18  
**Status:** Accepted  
**Deciders:** Repository owner  
**Tags:** argocd, security, risk-acceptance

## Context

The ArgoCD overlays intentionally configure the server with authentication disabled while Traefik provides the ingress/TLS path.

A repository review identified this as a high-impact security exposure if the endpoint is reachable by an untrusted party. The configuration is not accidental: the repository owner explicitly accepts this risk for the current homelab operating model.

## Decision

Keep the current ArgoCD unauthenticated administrative-access configuration.

Do not “repair” `server.disable.auth: "true"` automatically or treat it as configuration drift while this ADR is active.

## Consequences

- Anyone who can reach the ArgoCD endpoint may obtain administrative capabilities provided by that unauthenticated server configuration.
- TLS does not mitigate authorization risk; it protects transport only.
- Network/perimeter reachability therefore becomes part of the security boundary even though those controls may live outside this repository.
- Any change that broadens exposure or changes the trust model should trigger a new ADR revisiting this decision.
- Security tooling may report the condition, but repository validation must not fail solely because authentication is intentionally disabled.
