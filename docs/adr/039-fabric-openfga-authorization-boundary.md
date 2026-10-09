# ADR-039 — OpenFGA as the Fabric-wide relational authorization boundary

**Date:** 2026-10-09  
**Status:** Accepted  
**Scope:** AIaaS  
**Related Project:** vixens roadmap  
**Related Issues:** #3988, #3981, #3980, #3979, #3852, #3856, #3802, #3803, #3729  
**Decider:** TXO Fabric product owner

## Context

TXO Fabric provides tenants, groups, individual and shared AgentIdentity, workspaces, skills, provider accounts and service agents. The first chat code under #3981 uses injected authorization callbacks; these are not an implemented source of grants. A hand-written per-endpoint RBAC system would fragment policy and make later revocation and delegated ownership difficult.

Authentik already authenticates humans. Credentials and LLM constraints have separate boundaries. TXO must prove real fine-grained access checks for a shared agent, including negative multi-user and multi-tenant scenarios, without leaking OpenFGA credentials to browsers or tenants.

## Decision

The owner **accepted OpenFGA on 2026-10-09** as the Fabric-wide relational authorization engine, deployed **once as a private Fabric platform service**, conceptually alongside the centralized Authentik service. **Not one OpenFGA runtime per tenant.** Implementation, store isolation and live acceptance are tracked by #3988; this ADR does not claim a deployed service.

Responsibilities:

- **Authentik:** human authentication, external subject and user/group source; Fabric maps those to immutable technical IDs (#3852).
- **Fabric authorization service backed by OpenFGA:** explicit subject-resource-action relations such as `can_chat`, `can_read`, `can_manage`, `can_use`; person, group, tenant and system ownership do not automatically mean permission.
- **Fabric effective policy:** hierarchical Fabric Admin → Tenant Admin → User values and locks (allowed providers/models, budgets, geographic constraints, BYOK, payment). `OpenFGA allow` does **not** bypass a locked parent or service secret/CPA scope.
- **Fabric APIs / credential broker:** check permissions and effective policy for every sensitive operation before calling Hermes/CPA/workspaces; enforce service-to-service identity and Kubernetes NetworkPolicies.
- **Kubernetes RBAC / NetworkPolicy / Secret policy:** transport and infrastructure boundary independent of OpenFGA.

### Deployment and isolation

One private shared OpenFGA control-plane service under Fabric ownership, durable PostgreSQL on the existing CNPG substrate with dedicated database and least-privileged role, schema migrations before serving, no public Ingress or playground. Use OpenBao/ESO-provisioned secret for service authentication and datastore credentials; never a shared hardcoded token in Git.

**Store topology:** Prefer **one OpenFGA store per tenant plus an isolated platform-control store**, operated by the central Fabric authorization service. This separates relation sets, supports explicit lifecycle/recovery and avoids an accidental cross-tenant union in queries; it does **not** by itself make OpenFGA's server API enforce tenant RBAC. Store identifiers are Fabric-private and resolved only from authenticated server-owned tenant IDs. The service credential can access all stores, so **only trusted Fabric backend workloads** may talk to the OpenFGA API. Do not expose its bearer key or direct API to tenant agents, plugins, SaaS consumers or browsers.

A single model revision is published and pinned by the control plane for each store. Tuple changes are reconciled from authoritative IAM/Fabric ownership state; self-service clients submit *intent* to Fabric and must never write tuples directly.

### Enforced invariant

```text
OIDC-authenticated principal -> stable Fabric subject + active tenant
  -> Fabric canAccess(principal, action, canonical resource) -> OpenFGA Check
  -> Fabric effective restrictions / locked values
  -> private authorized action (Hermes, CPA, filesystem, service)
```

- Default **deny** if identity, owner, store, model version, group membership or OpenFGA availability is missing/invalid.
- No durable auth grants synthesized from a user-controlled Host header, mutable login, browser JSON or a stale OIDC group claim.
- Revoking a membership prevents new chat calls and credential uses; cached/check sessions must not silently retain old authorization. Existing decrypted/derived data and already-running tool calls are handled separately in #3803.
- An agent owner (user/group/tenant/system) is **metadata**; `owner` does not imply `can_chat`, `can_manage` or `can_use` without an explicit granted relation.
- The FGA model must be tested with allow/deny assertions before model deployment. A model file in Git is **not** an automatically deployed model/tuple set.
- Cross-tenant boundary is enforced in the Fabric API (tenant-resolved store + resource ownership), not entrusted to bare OpenFGA direct access.

## Alternatives considered

- **Continue in-process custom RBAC only:** less operational overhead now, but duplicates IAM logic across chat, workspaces, skills, integrations and provider accounts; rejected as the long-term architecture.
- **Authentik groups/OIDC claims alone:** insufficient for resource-level and inherited relations and may remain stale across session life; retained as identity input, not as authorization source.
- **Separate OpenFGA for every tenant:** strong topology separation, but greater cost, schema/migration overhead and lifecycle complexity; rejected for v0.1 in favor of central service with backend-controlled store separation.
- **OPA/Cedar policy engine for everything:** useful for attribute/constraint policy but does not substitute for the selected relation graph; Fabric effective-policy contract remains distinct.

## Consequences and rollout gates

- A Fabric-wide shared **critical security dependency** is introduced; the failure mode is deny, with operational alerting, backups, PostgreSQL/PDB and verified recovery.
- Implement in phases: (1) model and negative tests (#3988), (2) service/DB/secret wiring (GitOps), (3) stable subjects/AuthN sync, FGA adapter and tuple reconcile, (4) live check through hAIrem chat then Indiba without borrowing credentials.
- Prepare GitOps candidate and tell the owner **the exact immutable `dev-vYYYY.MM.PR` candidate and promotion command** once build, secret prerequisites and validation have been inspected. **Never promote implicitly.** Documentation/model CI alone never proves a working authorization service.
- Prior production may rely on existing grants; cutover must be staged without allowing unknown access and without leaving agents unsafely locked out by surprise.

## References

- [OpenFGA](https://openfga.dev/)
- [OpenFGA model testing](https://github.com/openfga/cli)
- [Use cases](../../usecase/README.md), #3988, #3981, #3980.
