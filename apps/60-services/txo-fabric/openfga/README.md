# Fabric OpenFGA — central authorization service

**Approved architecture, implementation in progress. Not deployed.** #3988 and ADR-039.

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

## Next GitOps integration stages

1. Provision **one** private OpenFGA Deployment/Service as a Fabric infrastructure dependency, with dedicated CNPG PostgreSQL database/role, database schema migration Job, secret-backed service authentication and tight NetworkPolicies. Follow TXO Fabric's existing OpenBao/External Secrets bootstrap contract; never put a pre-shared API key or database password in Git.
2. Make Fabric the only OpenFGA API caller. Design read/write service credential scope; note: standard OpenFGA pre-shared keys do **not** by themselves implement per-tenant store authorization.
3. Implement versioned model/store provisioning and membership/ownership tuple reconciliation, including revocation from Authentik/Fabric authoritative data and safe tenant offboarding. Do **not** use a manual `fga store import` as an ongoing GitOps substitute.
4. Connect `webui/server/authorized-turn.mjs`, `chat-sessions.mjs` and authorized agent discovery to a real fail-closed Fabric authorization adapter. Check sessions on every access; restore after container restart must not bypass Check.
5. Prove hAIrem and Indiba isolation and permission revocation on live service. Do not silently share a user OAuth credential through a tenant pool.

### Promotion gate

This repository's production promotion is **manual per immutable candidate**. No promotion is needed for this *model-only* PR. Later, once runtime manifests and credential provisioning are reviewed, verify dev candidate `dev-vYYYY.MM.<PR>`, ArgoCD readiness and a scoped physical smoke test **before recommending**:

```sh
gh workflow run promote-prod.yaml -f version=vYYYY.MM.<PR>
```

The owner runs that exact command (or explicitly authorizes it) for the reviewed candidate. No promotion solely because CI is green.
