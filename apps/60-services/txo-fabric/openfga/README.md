# Fabric OpenFGA — central authorization service

**Central OpenFGA service deployed in production; tenant authorization integration not yet active.** #3988, #3990, #3996 and ADR-039. The operator/store client introduced in #3997 is staged code only, not connected to live TenantBundle reconciliation.

## Selected boundary

One OpenFGA **platform service**, not a runtime per tenant. Authentik authenticates humans and supplies external immutable subjects. Fabric maps them to canonical IDs; a Fabric-private authorization backend controls access to the OpenFGA service and selected store. A tenant user's browser, Hermes agent or tenant pod must **never** receive the OpenFGA service credential or store administration rights.

- Separate **authorization stores** (one per tenant plus one platform-control store) on the same centrally deployed OpenFGA server.
- Store and authorization model IDs are resolved server-side, not accepted from client input.
- Ownership is **not** implicit permission; a group-owned agent requires a `chatter` or `manager` grant, or an explicitly authorized tenant administrator.
- **`OpenFGA allowed` is necessary but not sufficient**: Fabric hierarchical constraints and CPA/service scopes must also allow an action.
- Existing file/skill mount isolation and Kubernetes network policies remain in force regardless of an FGA result.

## Current files

- `model/model.fga`: versioned authorization model for `tenant`, `group`, `agent`, `workspace`, `provider_account`.
- `model/contract.fga.yaml`: positive and negative two-tenant checks, group agent ownership, provider `use` vs `manage`, no implicit rights from `owner`, synthetic revocation.
- CI `txo-fabric-openfga-model-ci.yaml`: validates the DSL and exercises the test file using the upstream FGA test runner.

These model fixtures use **synthetic**, non-live IDs. A green CI does not populate any OpenFGA store or synchronize Authentik accounts.

## Next integration stages

1. **Done (infrastructure):** one Fabric-wide private OpenFGA service, dedicated CNPG database/role, OpenBao ExternalSecrets and GitOps runtime. The production deployment was reported Synced/Healthy, with DB/role Applied and server 1/1 available (PR #3994). These signals **do not prove that a tenant store, authorization model or grant exists**.
2. **#3996 in progress:** operator-owned per-tenant store reconciliation, durable mapping of immutable `TenantBundle.spec.tenantId` to private store/model IDs, versioned model publishing, and human group membership tuple reconciliation sourced from Authentik. The store HTTP client + tests from #3997 are the first **non-activated** slice. Do not run `fga store create` manually per tenant as a substitute for operator reconciliation.
3. **#3992 / #3981:** connect real Authentik OIDC verifier, Fabric principal mapping, `webui/server/authorized-turn.mjs`, `chat-sessions.mjs` and agent discovery to fail-closed OpenFGA `Check` through Fabric BFF; store ownership and revocation are rechecked on every operation.
4. **Acceptance:** prove hAIrem and Indiba isolation, group/agent grant revocation, OpenFGA outage deny, and no accidental use of another user's OAuth credentials. Add a Fabric-native read-only administration UI later (#3995), not as a prerequisite for this P0.

The OpenFGA preshared key is a **Fabric platform credential**, not a per-store tenant authorization mechanism. No browser, Hermes AgentIdentity, tenant Pod or client controls the store ID, model ID or raw API.

### Promotion gate

This repository's production promotion is **manual per immutable candidate**. No promotion is needed for this *model-only* PR. Later, once runtime manifests and credential provisioning are reviewed, verify dev candidate `dev-vYYYY.MM.<PR>`, ArgoCD readiness and a scoped physical smoke test **before recommending**:

```sh
gh workflow run promote-prod.yaml -f version=vYYYY.MM.<PR>
```

The owner runs that exact command (or explicitly authorizes it) for the reviewed candidate. No promotion solely because CI is green.
