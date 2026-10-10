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



### Scoped OpenFGA tuple reconciliation primitive — staged #3996

`internal/openfga/tuples.go` stages a **private, source-only** tuple reconciler for a single Fabric-owned `object + relation` at a time. It uses an immutable `storeID` and `modelID` supplied by trusted Fabric code, validates the approved model, lists the exact relation with bounded pagination, calculates the desired-minus-actual difference, revokes stale tuples **before** adding new tuples and verifies the exact resulting relation set and pinned model. OpenFGA's `on_duplicate=ignore` / `on_missing=ignore` options make retries tolerant of prior successful writes; no success is reported if a write or read-back fails.

The primitive allows only a deliberately restricted `group` and `agent` relation vocabulary. It cannot select a whole store for deletion, take browser-supplied store/model IDs or infer permissions from an agent owner, IAM group name or workspace scope. Each call represents an **authoritatively complete snapshot for that one relation**; it MUST NOT be passed a partial Authentik response as an empty membership set. Tenant ownership and serialized ownership of writes are responsibilities of the forthcoming Fabric controller integration.

**Not connected to `TenantBundleReconciler` yet:** this stage does not query Authentik, map OIDC subjects, materialize `AgentIdentity` grants, or activate any tuple writes in the cluster. Before connecting it, the controller must resolve immutable Fabric IDs from verified IAM subjects, distinguish authored grants from membership, detect partial/stale snapshots, set revocation/readiness conditions, handle deletion and concurrent generations, and validate real OpenFGA `Check` against representative tenant A/B identities. The existing feature gate remains off and production promotion remains prohibited without explicit owner approval.


### Authentik human-membership snapshot client — staged #3996

`internal/authentik/memberships.go` introduces a **read-only, operator-owned** Authentik 2026.8 API client pinned to the internal Service `authentik.auth.svc:9000`. It resolves the exact generated tenant IAM group (`txo-fabric-<tenant>-<group>`) by unique immutable Authentik group UUID, then enumerates members through `core/users/?groups_by_pk=...`. It verifies every page's count/cursor, checks that each user truly belongs to the selected UUID, excludes disabled users, rejects malformed or duplicate identity records, and repeats the complete snapshot before returning an apparently stable set. API credentials are never accepted from the browser, tenant payload or workspace declarations.

**This is not yet a grant source connected to OpenFGA.** The returned UUIDs are Authentik identities, *not* the canonical Fabric user IDs expected in `user:...` tuples. A trusted Authentik subject → Fabric stable user mapping, explicit agent permissions and ownership verification, runtime-only credential from OpenBao/External Secrets, dedicated operator ↔ Authentik NetworkPolicies, retries and freshness ceilings must be implemented before its results may flow to `ReconcileTupleScope`. Double reading detects common changes across pages but is **not an atomic IAM transaction** and must not be treated as one. During an IAM outage, the operator must report authorization not ready, never treat missing/partial data as an authorized empty user list, and the BFF must deny checks until revalidation.

The existing human workspace `access.userRef` and group paths/PVCs are deliberately unchanged. #3852 requires a separate identity/migration design decision before altering those durable resource names. There is no production IAM token, external secret or open network rule added by this source-only stage. The OpenFGA gate stays off and no production promotion is authorized.

### Tenant-scoped canonical Authentik / OIDC / Fabric identity projection — staged #3996

`internal/authentik/identity_registry.go` introduces a source-only, strictly tenant-scoped identity projection from **explicit, complete Fabric-owned bindings**: `(tenantID, trusted issuer, exact OIDC sub, Authentik user UUID, immutable fabricUserId)`. It never assumes that `sub == user UUID`, derives an identifier from a mutable login, or equates `workspace.users[].name` / `access.userRef` with an authorization subject. Repeated/ambiguous subject, UUID or Fabric ID bindings are rejected. The group translator accepts only `TenantBundle`-approved IAM group keys and fails closed on every unmapped or duplicated Authentik UUID instead of generating partial desired grants.

**Important upstream contract:** generated tenant Authentik OAuth2 providers currently omit `sub_mode`; the Authentik provider model defaults to `hashed_user_id`, which is **not** the `core/users` UUID. The server-side verification layer must map the *actually issued* opaque `sub` to the Authentik UUID and Fabric user ID via an approved persisted binding. Silently switching existing providers to `sub_mode: user_uuid` would be an identity migration affecting sessions and is **not performed here**. This source-only registry does not verify OIDC signatures, allocate IDs, create or mutate durable identity bindings, read an IAM API or write OpenFGA tuples.

**Not yet connected to TenantBundle or the web BFF.** The next implementation must define/prove the durable Fabric-owned binding lifecycle without per-tenant manual bootstrap, issuer/audience/server-session verification, Authentik group UUID continuity, freshness ceilings and revocation. Only a complete and verified IAM snapshot may feed the existing scoped tuple reconciler. This change adds no manifests, credentials, network access, PVC changes, user/workspace renames, grant defaults or production activation. `TXO_FABRIC_OPENFGA_STORES_ENABLED` stays disabled.

### Durable Authentik group UUID pins — operator-owned, gated #3996 slice

`internal/controller/tenantbundle_openfga_iam_group_pins.go` adds an **operator-owned, resourceVersion-checked IAM group UUID registry** on each tenant's retained private OpenFGA store/model ConfigMap. It records a versioned, canonical list of `{name, authentikGroupUUID}` for exactly the `TenantBundle.spec.humanAccess.web.iamGroups` groups. The operator initializes the pins automatically via the **read-only, double-read Authentik group membership API** after the blueprint and approved OpenFGA store/model + empty identity ledger have been reconciled. A valid empty-membership snapshot can pin an empty group; no human user or grant is created by this step.

Each subsequent guarded reconcile checks the actual **immutable Authentik group UUID**, not just the group name. Newly declared groups can be appended after verifying all groups first, but a recreated group, missing/incomplete API response, duplicate UUID, tampered JSON/schema, stale `TenantBundle` UID, model mismatch or previous pinned group removed from the declaration blocks reconciliation **without rewriting the pins**. Group removal/rename MUST wait for an explicit, verified `group#member` tuple revocation lifecycle before retiring a pin; otherwise old grants could survive invisibly. The current implementation deliberately fails closed on group shrink rather than silently garbage-collecting it.

This guarded branch resolves an operator-owned Authentik token only from the platform-private `txo-fabric-system/txo-authentik-runtime` Secret key `token`; a controller test seam can inject the read-only client. **The required OpenBao ExternalSecret and operator→Authentik NetworkPolicy are not provisioned in this PR.** Until they are reviewed and GitOps-deployed, the guarded reconciliation reports `OpenFGAAuthorizationReady=False/IAMGroupPinReconcileFailed` when the token is absent. The production OpenFGA feature flag remains OFF, so current hAIrem and Indiba readiness is unchanged. No browser-supplied credentials, group UUIDs or arbitrary FGA stores are accepted.

This does NOT automatically enroll hashed OIDC subjects or synchronize member tuples in the main loop. A staged `reconcileIAMGroupMembersFromPersistedPins` wrapper already loads these Fabric-retained pins before invoking the #4019 group tuple projection, so the activation path never accepts caller-supplied UUID maps; **the wrapper is still uncalled**. Verified subject/UUID enrollment, freshness/revocation policy, agent grants, BFF and physical A/B/C `Check` tests remain blockers for enabling authorization or promoting to production.

### Verified Authentik group membership → scoped OpenFGA tuple convergence (staged #3996)

`internal/controller/tenantbundle_openfga_iam_projection.go` connects the existing *trusted source-only* components without activating them: a double-read Authentik `SnapshotGroupMembership` reader, the retained tenant-scoped `IdentityRegistry`, and the already implemented `openfga.Client.ReconcileTupleScope`. The new operator method accepts only `TenantBundle`-declared group keys and **previously pinned immutable Authentik group-name → group-UUID associations** from a future Fabric-owned durable source (not an OIDC group claim). It chooses the store and model itself from retained, UID-bound Fabric state and verifies them through the existing ledger/model reader. Unmapped users, incomplete or inconsistent membership snapshots, changed group UUIDs, and foreign tenants fail before **any** FGA write. All group snapshots must be validated first; the only desired relations are `group:txo-fabric-<tenant>-<key>#member@user:<fabricUserId>`.

Once an authorized owner invokes the method, its underlying scoped writer **removes obsolete membership tuples before adding new ones**, reads back the exact resulting set and revalidates the model; the controller rechecks the Authentik snapshots and retained binding afterward. Unit tests exercise grant, membership removal, empty-group revocation, tenant A/B store separation, absent or changed group pin, unmapped/disabled-incompatible data, source outages and mid-sync membership changes. This establishes the **source-to-tuple reconciliation method**, *not* an operational membership synchronizer: even repeated snapshots are not transactional, and a source race/outage **must** leave the caller/serving BFF in a denied/stale state until the next successful fresh convergence.

**Deliberate gate:** This helper is **not called by the main TenantBundle loop**. Its trusted group-UUID pin registry, verified OIDC subject → Authentik UUID enrollment writer, operator Authentik service token from OpenBao, scoped NetworkPolicy, refresh/lease/freshness enforcement, AgentIdentity grants, BFF deny-on-stale, and physical A/B/C Check+revocation acceptance still require separate implementation and evidence. In particular, **never pass group UUIDs from the browser or auto-pin a recreated group from its name**. `TXO_FABRIC_OPENFGA_STORES_ENABLED` remains off and `OpenFGAAuthorizationReady` remains False. No new prod permission or rollout is implied by this PR.

### Retained per-tenant canonical human identity ledger — guarded #3996 slice

`internal/controller/tenantbundle_openfga_identities.go` now hooks into the **existing disabled-by-default OpenFGA store/model gate**, *after* the operator has durably verified its tenant store and model. For each `TenantBundle` with `humanAccess.web`, the operator initializes an **empty** and versioned `identityBindings: "[]"` ledger plus an issuer pin on the already retained, platform-private `txo-fabric-system/txo-openfga-tenant-<tenantId>` ConfigMap. The same protected binding persists `tenantID`, original `tenantUID`, `storeID`, `modelID` and model fingerprint: **no new tenant provisioning entry point or workspace/PVC migration** is introduced. No per-client manual store/identity-ledger bootstrap is needed once the gate is deliberately enabled in a reviewed later rollout.

The ledger is read with strict JSON field and trailing-data validation, issuer and tenant-UID binding, duplicate user/subject/UUID detection via `internal/authentik/identity_registry.go`, and a fresh OpenFGA model binding verification; malformed, partially removed or cross-tenant records fail closed. A version marker makes losing either issuer or records detectable. Reconciliation **does not replace or mutate existing human mappings**, authorize on ledger presence or silently switch issuer; a verified issuer migration and human offboarding need explicit lifecycle work. A future trusted Fabric API must validate the OIDC identity and its actual Authentik account UUID before writing via a resourceVersion/CAS-guarded enrollment path. It must not reconstruct OIDC `sub` from Authentik's username or assume `sub == UUID`.

**This is not a complete grant implementation.** The ledger contains **zero enrolled humans at creation**, does not discover users from `workspace.users`, Authentik groups or claims, and never calls the tuple writer. Membership sync, issuer-sub enrollment, group UUID continuity, periodic refresh, revocation/physical `Check`, BFF resolver and platform-control store provisioning are still required. `OpenFGAAuthorizationReady` remains **False / TupleSyncNotImplemented** even when the ledger is valid. The production `TXO_FABRIC_OPENFGA_STORES_ENABLED` flag is untouched/off. No deployment, secret, NetworkPolicy or production promotion change belongs to this PR.

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
