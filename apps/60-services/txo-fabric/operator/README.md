# TXO Fabric Operator

`txo-fabric-operator` is the authoritative reconciler behind the Fabric CRDs.
It replaces the temporary Kyverno `generate` provisioning POC; Kyverno remains a
policy/admission engine, not the Fabric lifecycle engine.

## API ownership

- `TenantBundle` describes one Fabric Cell. `metadata.name` is the canonical
  tenant slug; `spec.tenantId` is the immutable business identifier.
- `AgentIdentity` describes who an agent is and references a `TenantBundle` by
  name. Because the CR is cluster-scoped, `metadata.name` is the globally unique
  Kubernetes object identity (recommended form: `<tenant>-<agentKey>`), while
  `spec.agentKey` is the immutable tenant-local machine identity used for Hermes
  profiles, runtime resource names and the default memory-bank key. This lets
  different tenants each have an agent called `sales`, `assistant`, etc. without
  colliding. `displayName` remains purely user-facing.
- `AgentRuntimeProfile` describes how an agent runs (image, storage, resources,
  Vixens scheduling and s6 compatibility). Runtime infrastructure is not part of
  the identity object.
- `PostgreSQLProfile` describes the platform-owned persistence implementation.
  The first supported topology is `SharedCluster`: TXO references a GitOps-owned
  CloudNativePG Cluster and owns only tenant `Database`, `DatabaseRole` and
  credential Secret resources.
- `AIGatewayProfile` describes the platform-owned tenant-facing inference gateway.
  The v0.1 target is one tenant-local LiteLLM facade per `TenantBundle`, backed by
  its own CloudNativePG `Database`/`DatabaseRole` binding for virtual keys,
  metering, budgets and proxy state. The gateway database is deliberately distinct
  from tenant application/Hindsight persistence.
- `AICredentialBrokerProfile` describes an internal provider-credential broker.
  The first implementation is a pinned CLIProxyAPI instance whose mutable OAuth
  state is isolated from every AgentIdentity PVC.

The operator rejects two live `AgentIdentity` resources that claim the same
`tenantRef.name + agentKey` pair instead of letting them fight over the same
Deployment/PVC/NetworkPolicy.

The current controller reconciles:

- tenant namespace + default-deny network baseline;
- isolated Hermes PVC, Deployment and egress policy per `AgentIdentity`;
- Shared PostgreSQL persistence through CloudNativePG `Database` and
  `DatabaseRole` resources, with generated secret-backed credentials and required
  extensions declared by `PostgreSQLProfile`;
- explicit reclaim semantics for tenant PostgreSQL resources;
- opt-in tenant-scoped CLIProxyAPI credential-broker substrate, including isolated
  OAuth-state storage, generated internal/management credentials, Service and
  NetworkPolicy;
- tenant LiteLLM PostgreSQL state reconciliation through a dedicated
  `postgresql-litellm` profile, generated database credentials, tenant-local
  runtime Secret, restricted migration egress and pinned-image Prisma migration Job;
- the LiteLLM serving/routing runtime remains deliberately fail-closed until the
  next #3885 slice;
- finalizers and standard Kubernetes status conditions;
- a semantic runtime probe that verifies a real `hermes gateway run` process,
  avoiding the prior `s6 + sleep infinity` false-positive readiness state.

TXO never owns or mutates the referenced CloudNativePG `Cluster`; RBAC grants the
operator read-only access to that dependency. Tenant database credentials are
generated outside Git and never copied into Fabric status.

Hindsight and its memory-bank lifecycle are reconciled by the current controller.
Optional Fabric modules remain represented in `TenantBundle` but are reported as
pending until their reconcilers are implemented. They must not be added as more
Kyverno generate rules.

The target tenant AI plane is tracked by #3885: tenant-local LiteLLM is the
consumer-facing policy/model/metering facade, while CLIProxyAPI is an internal
OAuth/subscription credential broker behind it. Existing tenants remain on the
shared LiteLLM/OpenRouter path until the tenant-local plane passes physical
acceptance. The direct-CPA smoke topology must not be promoted as the final
architecture.

## OpenFGA tenant store lifecycle — staged #3996

The private Fabric-wide OpenFGA service is deployed independently from the operator. **One service does not mean one store**: the target is one authorization store per `TenantBundle.spec.tenantId` plus a separate platform-control store. A store name is derived only from the **immutable business tenant ID** (`txo-fabric-tenant-ten00001`), never the mutable display name, AgentIdentity name, username or browser input.

`internal/openfga/stores.go` currently implements the platform-private store discovery/create/adoption primitive (including conflict detection, paginated discovery, deny-on-outage and no deletion on retries). Its tests use a fake HTTP transport to verify A/B tenant isolation, non-duplication and fail-closed behavior.

The store client was initially introduced by #3997 and connected to `TenantBundleReconciler` by #3998, **behind a disabled-by-default feature gate**. The operator reads its credential from the OpenBao-synced Secret and persists a private store binding; it does not manage human group membership in Authentik. Neither store creation nor model publication grants users any permissions. The full #3996 activation, tuple reconciliation and physical revocation tests remain pending.

Do not run manual per-client `fga store create` as a substitute for the remaining operator reconciliation. OpenFGA store names are not a uniqueness constraint: if duplicates already exist, the client refuses to select one, rather than guessing or deleting data.

### Guarded TenantBundle → OpenFGA store binding

The TenantBundle reconciler now has an intentionally **disabled-by-default** `TXO_FABRIC_OPENFGA_STORES_ENABLED=true` feature gate. This gate MUST NOT be enabled in the production operator Deployment until the same model and membership tuple reconciliation has been reviewed and physically proven. While disabled, `OpenFGAStoreReady=False/RolloutDisabled` is diagnostic and does not block the existing tenant's Ready status or create stores.

Once deliberately enabled through GitOps, `TenantBundleReconciler`:
- validates no two tenant objects share a `spec.tenantId`;
- reads the private OpenBao-synced `txo-fabric-system/txo-openfga-runtime` service key without persisting/logging it;
- adopts or provisions the uniquely named per-tenant OpenFGA store using `internal/openfga`;
- persists the association in a **platform-only ConfigMap** `txo-fabric-system/txo-openfga-tenant-<tenantId>` with immutable tenant ID, name, original Kubernetes UID, and store ID (never API token);
- verifies the association on repeated reconciliation; a deleted/recreated tenant UID, store change, or duplicate business ID is a **hard deny**, not an automatic cross-customer adoption;
- sets `OpenFGAStoreReady` independently of the **still absent model/tuple grants**; outages/identity conflicts degrade only when the rollout gate is enabled.

The binding is intentionally retained on tenant deletion so customer authorization history cannot be silently transferred to a new tenant. The final deletion/offboarding/revocation policy is tracked in #3996 and must be implemented *before* enabling the full integration. The operator NetworkPolicy admits only its outbound TCP 8080 path to the Fabric-owned OpenFGA pod; the existing OpenFGA ingress policy already only permits the operator.

**No production promotion is required for this source-only guarded slice.** Follow WORKFLOW.md: an operator code merge creates a source-only dev tag and a separate generated image pin PR; only the completed, physically validated immutable pin candidate may be promoted with explicit operator approval.

### Source-controlled OpenFGA model publication — #3996

`apps/60-services/txo-fabric/openfga/model/model.fga` remains the **canonical Fabric authorization model**. The compiled API JSON artifact is embedded inside the operator as `internal/openfga/model.json`; OpenFGA model CI runs the upstream `fga model transform` command and rejects drift from the DSL. The existing two-tenant FGA test suite continues to validate positive/negative access.

`internal/openfga/model.go` introduces an isolated model publication primitive:
- compares the *latest* store model against the exact approved compiled definition;
- returns the existing immutable model ID without writing another copy on retries;
- writes the approved model **only when no model exists**, then verifies it is visible and latest;
- rejects a foreign/newer model, model ID drift, API rejection/outage, or unsupported store ID, rather than automatically downgrading or upgrading an unknown authorization policy;
- exposes a deterministic content fingerprint for a future stable per-tenant store/model mapping.

### TenantBundle model ID + fingerprint binding — guarded #3996 slice

`TenantBundleReconciler` now uses the private model API after verifying the tenant's retained store binding. With `TXO_FABRIC_OPENFGA_STORES_ENABLED=true` **only**, it validates or publishes the exact approved model and records `modelID` and `modelFingerprint` **in one ConfigMap update**, alongside the existing verified `tenantID`, `tenantUID` and `storeID`. It never rewrites a different model ID or a changed content fingerprint silently.

Repeated reconciliation confirms the published latest model without unnecessary Kubernetes writes. A missing service credential, partial binding, fingerprint mismatch, foreign store, concurrent conflict, API outage or unknown model revision causes a hard failure and `OpenFGAModelReady=False`. No model exists in Fabric auth state until its ID and fingerprint are both verified. A crash between server publication and binding persistence is recoverable through verified adoption.

Conditions deliberately distinguish `OpenFGAStoreReady`, `OpenFGAModelReady`, and `OpenFGAAuthorizationReady`. Even with store and model ready, `OpenFGAAuthorizationReady=False/TupleSyncNotImplemented` and overall tenant `Ready=False/AuthorizationTupleSyncPending` while the flag is enabled: this stage provides **no actual grants or revocation**. With the flag disabled, existing tenant readiness semantics remain unchanged. Do not enable the flag yet; authoritative Authentik membership and Fabric/AgentIdentity tuple synchronization, deletion/revocation policy, runtime BFF checks and live two-tenant acceptance are still required.

The binding ConfigMap and FGA service key are platform-private. The binding resolver is **not** a browser API and does not by itself prove permission to chat. Model freshness must still be checked by trusted Fabric authorization callers before any sensitive operation. This source-only slice is not a production promotion candidate.


## Development

The production cluster is Kubernetes 1.34, so this module pins controller-runtime
0.22.x / k8s.io 0.34.x. Generate API/RBAC assets with controller-tools 0.19.x.

```sh
make generate manifests
make test
```

The operator image is built to GHCR after merge; a separate GitOps change enables
the manager only after an immutable image tag exists. This prevents a source PR
from deploying a non-existent image into `txo-fabric-system`.
