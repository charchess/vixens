# TXO Fabric — tenant-neutral management plane

This directory defines the tenant-neutral TXO Fabric management-plane bootstrap,
its authoritative Fabric API, and the operator that reconciles tenant cells.

A fresh deployment remains healthy with **zero tenants**: no customer namespace,
Hermes runtime, Hindsight bank, Paperclip or Valkey workload is created until a
`TenantBundle` or `AgentIdentity` is declared.

## Ownership model

Git and Argo CD own the high-level desired state:

- `TenantBundle` declares one Fabric Cell and its requested capabilities;
- `AgentIdentity` declares one tenant-local agent identity;
- `AgentRuntimeProfile` declares the platform-owned runtime implementation;
- `PostgreSQLProfile` declares how the platform satisfies tenant persistence;
- `HindsightProfile` declares how the platform satisfies tenant memory;
- `IntegrationConnection` and `IntegrationBinding` declare logical external
  integration metadata and per-agent authorization without embedding credential
  values.

The TXO Fabric operator owns the Kubernetes resources derived from that intent.
It reconciles continuously instead of writing generated manifests back to Git.
Kyverno remains available for admission, validation and generic security policy;
it is not a Fabric lifecycle engine.

## API contracts

### TenantBundle

`TenantBundle` is cluster-scoped. `metadata.name` is the canonical tenant slug and
`spec.tenantId` is the immutable business identifier. The operator reconciles the
tenant namespace, its default-deny network baseline and the mandatory tenant AI
plane for every Active tenant.

`spec.lifecycle.mode` defaults to `Active`. Active tenants automatically receive
tenant LiteLLM + tenant CPA; `spec.aiGateway` and `spec.aiCredentialBroker`
remain only as v1alpha1 profile overrides and are not enable/disable switches.
`Parked` is the recovery-shell state: AI-plane compute is stopped while durable
OAuth/database state is preserved for reactivation.

PostgreSQL, Hindsight and optional modules remain explicit API capabilities. A
TenantBundle selects platform-owned persistence and memory implementation profiles
through `profileRef`; it does not contain database credentials, provider secrets,
or provider-specific connection strings.

Shared PostgreSQL and tenant-scoped Hindsight are reconciled by the operator.
Optional modules remain pending until their lifecycle reconcilers are implemented
rather than being silently treated as ready.

### AgentIdentity

`AgentIdentity` is cluster-scoped and separates global Fabric identity from the
stable tenant-local runtime key. Canonical populated agents use stable keys such as
`usr000001-agt00012`; derived runtime resources therefore use names such as
`hermes-usr000001-agt00012`, `hermes-usr000001-agt00012-data`, and the
corresponding agent-scoped network resources.

Different tenants may use the same tenant-local `agentKey` while globally scoped
Fabric CR names and tenant ownership keep those identities distinct. Do not derive
identity from mutable display names, Pod names or group membership.

### AgentFunctionalProfile (v0.1 instructions-only)

`AgentFunctionalProfile` is a **tenant-owned, cluster-scoped, non-secret**
business-instruction baseline independent of `AgentRuntimeProfile` and
`AgentIdentity`. One agent may opt into one profile using
`AgentIdentity.spec.functional.profileRef`. The profile's immutable
`spec.tenantRef.name` must match the agent's tenant. A missing or foreign
profile is rejected and the previous agent runtime/human route is withdrawn
without deleting the agent's retained private state. An unbound agent retains
the existing behavior.

The first implementation slice intentionally supports **non-secret
`spec.instructions` only**. It carries no credentials, IAM grants,
workspace mounts, toolset approvals, model credentials or IntegrationBinding
authorization. Approved instructions are passed to the pinned Hermes gateway
via `TXO_FUNCTIONAL_SYSTEM_PROMPT` in the certified **TXO-patched pinned Hermes runtime**.
The managed layer is composed separately from the user's `/personality` and
channel overrides on every gateway turn. This requires
`AgentRuntimeProfile.spec.compatibility.functionalPromptOverlay=true` on a
verified compatible runtime image (default **false**, fail closed). The role
overlay does not write the
agent's private `SOUL.md` or skills. The effective revision is recorded on the
Hermes pod template and changes roll only the relevant agents. Role instructions
are readable in the profile and Pod environment, so **do not put secrets in
the role text**.

```yaml
# Example only; do not apply to production before acceptance of #3934.
apiVersion: fabric.truxonline.io/v1alpha1
kind: AgentFunctionalProfile
metadata:
  name: indiba-sales
spec:
  tenantRef:
    name: indiba
  instructions: |
    Apply the tenant-approved commercial playbook.
    Personalize the interaction for each user's context.
# Existing Sam and Alex would each opt in with:
# spec.functional.profileRef: indiba-sales
```

This opt-in is **not added** to the production Indiba manifests by #3934.
The UX and existing-session prompt compatibility of the pinned Hermes runtime
require a separate physical acceptance, especially on profile updates and
withdrawal. The broader skills/integrations/model-precondition composition
remains a proposed extension described in
[FUNCTIONAL-CONFIGURATION.md](FUNCTIONAL-CONFIGURATION.md) and proposed ADR-038.

### AgentRuntimeProfile

`AgentRuntimeProfile` separates identity from infrastructure policy. The profile
selects the Hermes image, storage class and size, resource envelope, scheduling
priority, compatibility settings and platform-owned executable toolset policy.
Tenant and agent manifests do not embed those implementation details. The
three-state Hermes capability contract and pinned-runtime inventory are documented
in [CAPABILITIES.md](CAPABILITIES.md).

### IntegrationConnection / IntegrationBinding

The v0 integration contract keeps logical connection metadata and per-agent
authorization in Fabric rather than in writable Hermes `/opt/data`. Credential
values remain outside the CRDs and Git. The current v0 adapter, temporary broad
POC egress, authenticated `fabric-smoke` canary, revocation semantics and v0.2
broker migration boundary are documented in [INTEGRATIONS.md](INTEGRATIONS.md).

### PostgreSQLProfile

`PostgreSQLProfile` is cluster-scoped and describes platform persistence policy.
The initial API deliberately supports only `SharedCluster`. The concrete
`postgresql-shared` profile selects the GitOps-owned Vixens CloudNativePG cluster
`databases/postgresql-shared` and establishes the contract: one logical database
and one login role per tenant, required database extensions, and explicit reclaim
policy.

The Fabric operator does not own or mutate the referenced CloudNativePG `Cluster`.
Its RBAC access to that dependency is read-only. It owns only the tenant logical
`Database`, `DatabaseRole` and generated credential Secret derived from the
profile. A dedicated per-tenant cluster topology is deferred until the Shared path
has been validated end-to-end; that investigation is tracked separately in #3573.

The controller derives stable PostgreSQL identifiers from the immutable tenant ID,
creates a `kubernetes.io/basic-auth` credential Secret outside Git, then waits for
CloudNativePG to apply the login role before creating the database. Required
extensions such as `vector` are declared through `Database.spec.extensions`, so
TXO does not connect directly to PostgreSQL to execute lifecycle SQL.

Reclaim is explicit rather than relying on ownerReferences. With `Retain`, deleting
the Fabric CR removes the CloudNativePG management CR while CloudNativePG retains
the logical database/role; the generated credential Secret is retained with them.
This prevents Kubernetes garbage collection from bypassing the profile's reclaim
contract.

Passwords, connection credentials and physical connection strings are never stored
in the Fabric CRD or Git.

### HindsightProfile

`HindsightProfile` is cluster-scoped and describes the tenant-scoped Hindsight API
runtime contract: immutable API image, API port, inbound API-auth mechanism,
resource envelope, optional model cache, scheduling policy and the platform
LLM-auth mode. The profile selects an auth mechanism but never embeds the API key
itself.

The initial `hindsight-standard` profile uses the upstream API image and requires
API-key authentication. Hindsight generative/reflection LLM processing remains
disabled with `HINDSIGHT_API_LLM_PROVIDER=none`. The canonical embedding path is
the tenant-local LiteLLM facade using the logical `txo-embedding` model and a
Hindsight-scoped virtual key. The OpenRouter provider credential exists only on
the platform-owned tenant gateway runtime; it never enters the Hindsight Pod.

Existing tenants may temporarily remain on the historical shared gateway while
their tenant OpenRouter credential and `txo-embedding` route are prepared. The
operator performs a failure-safe shared -> tenant cutover: it prepares the
destination key first, rolls Hindsight onto the tenant gateway, and revokes the
shared key only after the new runtime is Available. A tenant that has completed
the cutover never silently falls back to the shared gateway.

The operator consumes the tenant PostgreSQL binding, creates tenant-local runtime
Secrets, and reconciles one Hindsight API Deployment and Service per tenant.
Database credentials, the Hindsight tenant API key and the scoped embedding-gateway
credential remain Secret-backed rather than being embedded in the profile or
TenantBundle. AgentIdentity resources resolve deterministic bank IDs against their
tenant's Hindsight service.

When a Hindsight-enabled TenantBundle explicitly sets
`memory.hindsight.humanAccess: true` and declares `humanAccess.web`, Fabric
additionally reconciles the paired upstream Hindsight Control Plane image, a private
Service, default-deny-compatible NetworkPolicies and a stable authenticated route:

`https://hindsight-<tenant>.<domainSuffix>`

The Control Plane receives the tenant API key only through a Secret-backed
server-side environment variable and talks to the private Hindsight API inside the
tenant namespace. Browsers never receive the API key, the memory API is not routed
publicly, and PostgreSQL remains hidden behind Hindsight.

For tenants declaring `humanAccess.web`, the Fabric operator also publishes the
structural Authentik desired state through the aggregate
`auth/txo-fabric-authentik-blueprints` ConfigMap. That generated blueprint owns
the tenant structural groups, public OIDC provider/application and policy bindings,
plus the Hindsight Proxy Provider/application when Hindsight human access is
requested. It also publishes the deterministic provider union consumed by the
embedded Authentik outpost, avoiding per-tenant outpost-provider clobbering.

Fabric owns this **structural IAM intent only**. Authentik remains the live source
for human users, passwords, MFA and group membership. The separate
`/outpost.goauthentik.io` route is sent directly to the embedded Authentik outpost
so the sign-in flow itself is not recursively protected.

`TenantBundle.status.memory.hindsight.endpoint` remains the private API endpoint,
while `humanEndpoint` reports the stable authenticated WebUI URL.

A shared multi-tenant Hindsight service remains deferred until its authentication
and database/schema isolation model is proven; that investigation is tracked
separately in #3574.

## Lifecycle and deletion

The operator uses explicit finalizers for destructive tenant and agent lifecycle.
Tenant Namespace and tenant default-deny NetworkPolicy are intentionally not
owned through Kubernetes garbage-collection ownerReferences: a TenantBundle
cannot delete its tenant cell while AgentIdentity resources still reference it.
The Namespace is removed only after the tenant finalizer has observed zero
remaining agents and the Fabric-owned PostgreSQL management resources have been
released according to their reclaim policy.

Agent runtime resources are reconciled from `AgentIdentity`. Existing compatible
PVCs can be adopted without rewriting their storage contract, preserving their
UID and data. Runtime workspace deletion is explicit through
`spec.runtime.storage.retentionPolicy`, with values `Retain` or `Delete`. The safe
default is `Retain`; `Delete` is an immutable opt-in for disposable identities.
When an AgentIdentity using `Retain` is deleted, the operator removes the PVC's
AgentIdentity ownerReference and leaves tenant/agent retention labels in place so
a replacement identity with the same tenant and `agentKey` can re-adopt it.

For a bound `Retain` runtime PVC, the operator also promotes the concrete
PersistentVolume reclaim policy to `Retain`. This is the backend-safety contract:
it protects durable state even when a brownfield PVC was originally provisioned
from a StorageClass whose default reclaim policy was `Delete`. The controller only
promotes to the safer policy; an AgentIdentity using `Delete` does not
automatically downgrade an already-retained PV. Disposable profiles should
therefore continue to use a Delete StorageClass when backend deletion is desired.

A TenantBundle cannot delete its tenant Namespace while retained Fabric agent PVCs
remain inside it. Intentional destructive tenant removal must therefore release
those workspaces explicitly first, for example by using an AgentIdentity declared
with `retentionPolicy: Delete`. The `fabric-smoke` fixture opts into `Delete` so it
remains suitable for destructive lifecycle tests; customer identities inherit
`Retain` unless they explicitly choose otherwise.

The scoped LiteLLM virtual key is revoked by deterministic alias on a best-effort
basis and its tenant-local model-access Secret is deleted with the agent lifecycle.

## Hermes runtime boundary

Hermes runs from the TXO-owned immutable runtime image derived from a reviewed
upstream `nousresearch/hermes-agent` release. TXO Fabric does not copy upstream
provider credentials or legacy OAuth state into generated runtimes.

For every Active tenant, model access is mediated by that tenant's LiteLLM
facade. The operator provisions one LiteLLM virtual key per `AgentIdentity`,
restricted to the logical `txo-agent` model and tagged with tenant/agent
metadata. Only that scoped key is written to the tenant namespace.

The platform-owned Hermes managed scope pins the behavioral model route in
`/etc/hermes/config.yaml`, so the pinned Hermes gateway runtime sees the same
route on every surface and the user-owned `/opt/data/config.yaml` cannot replace
it:

- `model.default=txo-agent`;
- `model.provider=custom`;
- `model.base_url=http://txo-ai-gateway.tenant-<tenant>.svc:4000/v1`.

Hermes additionally receives the secret/runtime bridge:

- `TXO_LLM_AUTH_MODE=gateway`;
- `OPENAI_BASE_URL=http://txo-ai-gateway.tenant-<tenant>.svc:4000/v1`;
- `OPENAI_API_KEY` from `Secret/hermes-<agentKey>-model-access`;
- `HERMES_MODEL=txo-agent`.

Upstream provider/OAuth credentials remain behind LiteLLM/CPA. Hermes network
egress permits DNS, its tenant-local LiteLLM, and its tenant Hindsight service
when configured; it has no direct CPA path and no shared-LiteLLM steady-state
route. The shared gateway is retained only as a bounded legacy credential source
for failure-safe cutover cleanup.

`ModelAccessReady=True` records successful scoped credential reconciliation.
A Parked tenant cannot resolve a model backend for an AgentIdentity and therefore
fails closed instead of falling back to shared or direct provider access.

The operator starts the Hermes host gateway with `gateway run --replace` and uses
semantic process probes. Each AgentIdentity uses its pod-level
`HERMES_HOME=/opt/data` as the single private Hermes home; TXO Fabric does not add
a second named-profile layer inside that runtime.

Hindsight bank identity is part of the AgentIdentity contract. When tenant memory
is configured, the operator exposes the resolved bank through the tenant-scoped
Hindsight service while PostgreSQL remains hidden behind Hindsight from the agent.

## Historical brownfield cutover

The now-retired hAIrem sandbox was the first brownfield validation path. The
temporary Kyverno provisioning policies were retired first, leaving the historical
Tesla, Tina and Tiffa runtime resources ownerless so the authoritative operator
could prove adoption through the new CR contract.

This section records that migration mechanism; it is not the current Client0
population or an onboarding recipe for new tenants.

PVC adoption is deliberately non-destructive. Deployment adoption may cause one
controlled `Recreate` rollout when the operator normalizes the old POC Deployment
spec, but the existing workspace PVC must retain its identity and data.

A retained brownfield PVC that still uses Hermes' former named-profile layout can
opt into `runtime.storage.adoptLegacyProfile: true`. In that mode the operator
validates `profiles/<agentKey>` from the full retained volume during bootstrap,
then mounts that existing directory with Kubernetes `subPath` as the runtime's
single `/opt/data`. The profile is neither copied nor flattened. Hermes still sees
`HERMES_HOME=/opt/data`, so this is a storage-layout adoption mechanism rather
than a return to named-profile runtime multiplexing. The opt-in requires retained
storage and must be checkpointed before production cutover. New identities use the normal flat runtime home and do not need this brownfield-only flag.

## Zero-tenant acceptance

Without tenant declarations:

```console
kubectl get tenantbundles
No resources found

kubectl get agentidentities
No resources found
```

The following must also be true:

- no `tenant-*` namespace exists because of TXO Fabric;
- no Hermes runtime exists because of TXO Fabric;
- no tenant database, memory bank, repository, Paperclip or Valkey instance is
  created implicitly.

`hAIrem` receives no special platform treatment. Test and production tenant cells
use the same API contracts as future external customers.

## Immutable Hermes releases and stable profile pointers (#3944)

**Separation of responsibilities:** `HermesRuntimeRelease` is the immutable
software bundle (Hermes image digest + optional independent Hindsight plugin
OCI digest); `AgentRuntimeProfile` is a stable **mutable pointer** plus
platform-owned storage/resources/capability rules; `AgentIdentity` retains
its own `runtime.profileRef`, private PVC, `SOUL.md`, sessions, skills and
Hindsight bank. The Kubernetes CRD enforces release immutability using CEL
`self == oldSelf` on its spec. Release image tags are not permitted: every
release artifact uses a full OCI `@sha256:` digest.

New API example (**illustrative, not a manifest to apply**; replace digest
placeholders with measured, trusted 64-digit SHA256 digests):

```yaml
apiVersion: fabric.truxonline.io/v1alpha1
kind: HermesRuntimeRelease
metadata:
  name: hermes-2026-09-24-a
spec:
  image: nousresearch/hermes-agent@sha256:<64-hex-digest>
  hindsightPluginImage: ghcr.io/charchess/txo-hermes-hindsight-plugin@sha256:<64-hex-digest>
---
apiVersion: fabric.truxonline.io/v1alpha1
kind: AgentRuntimeProfile
metadata:
  name: hermes-upgrade-canary
spec:
  releaseRef: hermes-2026-09-24-a
  # Retain current storage, sizing, resource ceilings, toolset policy and
  # compatibility sections in the actual profile.
```

`hermes-default` and `hermes-upgrade-canary` keep their stable object names.
They can reference different immutable releases while preserving their own
security and storage policy. Each pointer transition is Git-reviewed and
recorded in immutable `dev-v*` / `prod-v*` snapshots. A new release is
added; existing release content is **never** edited. Changing a profile
pointer updates **only agents referencing that profile**, not all agents
or tenants. Since `hermes-default` may be shared by many clients, its
pointer must **not** move without physical canary and retained-state
acceptance (#3688).

For compatibility, old profiles with inline `spec.image` and
`spec.bootstrap.hindsightPluginImage` continue to work during transition.
A profile must choose **exactly one** image source: inline image or
`spec.releaseRef`. Mixing releaseRef with legacy image/bootstrap fields,
pointing at missing releases, or using mutable release image tags fails
closed; any previously exposed runtime for that agent is withdrawn, with
its private PVC retained. The operator constructs a local effective
runtime profile, never copies the release image back into the GitOps CR.
The agent's `SOUL.md` remains agent-owned and is never regenerated by a
pointer change.

**Migration order:** first merge and pin the new operator image, then
introduce approved release CRs and update only the test canary profile in a
separate GitOps PR coordinated with #3941. #3941 currently modifies the
canary's *legacy inline* image; do not merge overlapping profile edits
from both implementations. A later reviewed PR may convert the unchanged
legacy `hermes-default` to a releaseRef only after a verified immutable
digest for the currently deployed runtime has been recorded, and only
with explicit operator authorization. No production agent is migrated
by this source/API PR. The native functional-role skills in draft #3938
remain a separate change to be rebased/reviewed.

---

## Official Hermes + independent Hindsight OCI extension (candidate #3939)

The supported legacy inline form uses **unmodified**
`nousresearch/hermes-agent:v2026.9.24` in `AgentRuntimeProfile.spec.image`
with optional `spec.bootstrap.hindsightPluginImage`. New pointer-based
profiles instead select a `HermesRuntimeRelease` that pins **both** upstream
Hermes and the independently built Hindsight plugin by exact OCI digests.
The plugin artifact uses `ghcr.io/charchess/txo-hermes-hindsight-plugin@sha256:...`.

When both the tenant has Hindsight configured and this image is set, Fabric
renders two initContainers: `prepare-hindsight-extension` copies a pinned
provider and its hash-locked Python wheels from the artifact into a
Pod-local `emptyDir`, and `bootstrap-profile` configures the private
Hermes profile, Hindsight bank and managed settings after the artifact exists.
The official Hermes gateway then reads the plugin at
`$HERMES_HOME/plugins/hindsight` and gets the two additional Python wheels
from `PYTHONPATH=/opt/txo-hindsight/python`. The plugin and dependency
volume mounts are read-only, no vendor or app files under
`/opt/hermes/.venv` are modified, and no GitHub/PyPI calls occur on
tenant Pod startup.

**Private identity:** `SOUL.md` is not touched by Fabric, even on first
boot; Hermes' own first-run seed (only if missing) is authoritative.
Managed Hindsight configuration is merged into
`$HERMES_HOME/hindsight/config.json` without discarding unknown personal
fields and rewritten atomically only when effective managed data changes.
The per-agent `HINDSIGHT_API_KEY` remains secret-backed. Recreating a
Pod re-copies only ephemeral dependency bytes and keeps the existing private
state and independent Hindsight bank untouched.

**Rollout safety:** this field is *opt-in*, and default/production profiles
keep their existing derived Hermes image until an approved migration.
A separate GitHub workflow builds the plugin artifact and generates a
review-only PR creating a **new immutable HermesRuntimeRelease** (engine and
Hindsight digests) and moving only `hermes-upgrade-canary.releaseRef`.
It never updates `hermes-default` automatically. The legacy derived-image pipeline no
longer publishes or auto-updates runtime pins. No prod promotion is implied
by a generated PR; physical image/s6/import/memory/PVC tests under #3688 are
required before changing `hermes-default`. During this migration,
the draft functional-role PR #3938 must be rebased against #3939; it
continues to supply native Hermes role skills, not a gateway patch.


The optional `bootstrap.hindsightPluginImage` field strictly requires an OCI
`@sha256:<64-hex>` digest. Floating tags are rejected at reconciliation time.
