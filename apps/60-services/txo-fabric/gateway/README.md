# TXO AI egress gateway

This directory contains the proprietor-managed AI egress boundary for TXO Fabric.
It is deliberately separate from tenant cells: tenant runtimes must not receive
upstream provider credentials.

## Architecture

The tenant-scoped multi-credential target contract for #3868 is documented in
[`CREDENTIAL_POOLS.md`](CREDENTIAL_POOLS.md). The target has moved to **one
CLIProxyAPI gateway per TenantBundle**. It is an incremental migration: the
current shared LiteLLM/OpenRouter route remains authoritative until the tenant CPA
path is explicitly enabled and physically validated.

The following section documents the **currently active compatibility path**, not
the #3868 target architecture.

TXO Fabric currently uses LiteLLM as an OpenAI-compatible gateway in
`txo-fabric-system`:

```text
TXO Fabric operator
    |
    | LiteLLM admin credential
    | /key/generate + /key/delete
    v
Service/txo-ai-gateway:4000
    |
    +--> PostgreSQL txo_ai_gateway (keys, spend, budgets/rate-limit state)
    |
    +--> OpenRouter (provider credential exists only here)

Hermes
    |
    | per-AgentIdentity scoped LiteLLM virtual key
    | model = txo-default
    v
Service/txo-ai-gateway:4000/v1

Tenant Hindsight
    |
    | per-TenantBundle scoped LiteLLM virtual key
    | model = txo-embedding
    v
Service/txo-ai-gateway:4000/v1/embeddings
```

The gateway exposes stable local aliases instead of leaking provider/model identifiers
to tenant workloads:

- `txo-default` -> initially `openrouter/openai/gpt-5.6-luna`;
- `txo-embedding` -> `openrouter/baai/bge-m3`.

Provider/model routing can therefore change centrally without rebuilding tenant
workloads or rewriting tenant intent. Provider-specific pricing belongs to the same
central model mapping: if LiteLLM's bundled cost map does not recognize the selected
provider/model identifier, the deployment must declare the current per-token rates
explicitly so persisted spend logs remain meaningful.

## AgentIdentity model access

Each `AgentIdentity` receives one LiteLLM virtual key restricted to model
`txo-default`. The operator creates the key with a deterministic alias and metadata
containing the tenant and agent identities, then stores only the returned virtual
key in the tenant namespace as `Secret/hermes-<agentKey>-model-access`.

Hermes receives:

- `OPENAI_BASE_URL=http://txo-ai-gateway.txo-fabric-system.svc:4000/v1`;
- `OPENAI_API_KEY` from the scoped tenant Secret;
- `HERMES_MODEL=txo-default`;
- `TXO_LLM_AUTH_MODE=gateway`.

The tenant runtime never receives the LiteLLM administrative credential or the
OpenRouter provider credential. Agent deletion attempts to revoke the LiteLLM key
by its deterministic alias before deleting the scoped Secret. Revocation is
best-effort so a temporary gateway outage cannot wedge the AgentIdentity finalizer.

`ModelAccessReady=True` means the scoped gateway credential has been reconciled.
Once the Hermes Deployment is also available, the AgentIdentity may transition to
`Ready` instead of the previous deliberate `AuthBlocked` state.

## Hindsight embedding access

`HindsightProfile.spec.llmAuthMode=PlatformGateway` activates the existing platform
model-credential boundary for tenant Hindsight. This does not enable Hindsight LLM
processing: `HINDSIGHT_API_LLM_PROVIDER` remains `none` so retain behavior does not
silently change. The scoped credential is used only for remote embeddings.

The operator stores the tenant-scoped LiteLLM key inside the existing
`Secret/hindsight-runtime` and configures Hindsight 0.10.1 with its supported
OpenAI-compatible embedding contract:

- `HINDSIGHT_API_EMBEDDINGS_PROVIDER=openai`;
- `HINDSIGHT_API_EMBEDDINGS_OPENAI_BASE_URL=http://txo-ai-gateway.txo-fabric-system.svc:4000/v1`;
- `HINDSIGHT_API_EMBEDDINGS_OPENAI_MODEL=txo-embedding`;
- `HINDSIGHT_API_EMBEDDINGS_OPENAI_API_KEY=<scoped LiteLLM virtual key>`;
- `HINDSIGHT_API_EMBEDDINGS_OPENAI_DIMENSIONS=` (empty), which Hindsight 0.10.1 resolves to `None` and therefore auto-detects the provider width.

The virtual key is restricted to `txo-embedding` and carries tenant metadata plus
`component=hindsight` and `capability=embeddings`. Hindsight therefore cannot use
this credential to call `txo-default`.

The steady-state embedding backend is `BAAI/bge-m3`, exposed through the stable
`txo-embedding` alias. BGE-M3 returns native 1024-dimensional vectors and Hindsight
is intentionally allowed to detect that width rather than pinning a platform
constant. Existing banks created with the former 384-dimensional path cannot be
switched in place: export/import must rebuild them against a fresh target bank so
Hindsight regenerates embeddings with the active model.

The local reranker remains in place and the baked-model offline flags remain
enabled; only embedding inference moves behind the gateway.

Tenant deletion revokes the deterministic Hindsight embedding-key alias on a
best-effort basis after Fabric-owned Hindsight resources have been removed. A
gateway outage must not wedge tenant cleanup.

## Inference policy and credential lifecycle

The active TXO Fabric external-inference consumers are deliberately small:

- Hermes uses the logical chat/model alias `txo-default` through a per-`AgentIdentity`
  LiteLLM virtual key;
- tenant Hindsight uses only the logical embedding alias `txo-embedding` through
  a per-`TenantBundle` LiteLLM virtual key;
- Hindsight generative/reflection LLM processing remains disabled with
  `HINDSIGHT_API_LLM_PROVIDER=none`;
- Paperclip is not yet an active Fabric-managed inference consumer (#3672). If an
  optional module later uses external LLM/embedding inference, it must join this
  gateway/scoped-credential contract instead of receiving an upstream provider key.

The model allowlists are enforced when each virtual key is created. Concrete
provider/model mappings remain solely in the gateway configuration, so changing
the backend for `txo-default` or `txo-embedding` does not require editing tenant
or agent intent.

### Scoped credential rotation

Credential rotation is a platform lifecycle operation and is independent from an
agent's retained `/opt/data`.

- changing `fabric.truxonline.io/model-access-rotation` on an `AgentIdentity`
  revokes its currently projected LiteLLM key, mints a replacement restricted to
  `txo-default`, updates the tenant-local Secret and rolls that Hermes runtime;
- changing `fabric.truxonline.io/hindsight-embedding-rotation` on a
  `TenantBundle` performs the equivalent replacement for the Hindsight
  `txo-embedding` key and rolls the tenant Hindsight workload;
- loss/recreation of a generated model-access Secret first clears any stale key
  under the deterministic LiteLLM alias before a replacement is minted;
- the applied rotation revision is a short hash of the requested nonce. It is
  non-secret diagnostic metadata; the raw virtual key is never copied into
  status, labels or annotations;
- rotation requests are edge-triggered: a non-empty nonce rotates only when its
  hash differs from the revision already applied to the generated Secret. If
  GitOps later removes an imperative request annotation, that means "no new
  rotation" and does not rotate the credential back to a baseline state.

Rotation is intentionally fail-closed: the old key is revoked before the
replacement becomes active. A failed replacement may temporarily block inference,
but it must not preserve an undisclosed old credential or silently bypass the
gateway.

### v0 budgets and rate limits

For the Client 0 POC, virtual-key `max_budget`, RPM and TPM limits are
**platform-owned but deliberately unset**. The enforced v0 control is the
workload-specific model allowlist plus independently revocable scoped credentials.
TenantBundle and AgentIdentity APIs do not own arbitrary quota values.

Future budget/rate-limit policy can be applied centrally when the operating policy
is known; choosing placeholder numbers now would turn an arbitrary POC value into
an accidental product contract.

There is no direct-provider fallback. If the TXO AI gateway or a scoped credential
is unavailable, the workload's external inference fails rather than switching to a
provider credential or generic Internet route.

## Secrets

No secret value is stored in Git. `ExternalSecret` resources read
`vixens/prod/apps/60-services/txo-ai-gateway` from `ClusterSecretStore/openbao`.
The OpenBao object must contain these properties:

- `postgres_username` — expected to be `txo_ai_gateway`;
- `postgres_password` — gateway database role password;
- `litellm_master_key` — LiteLLM administrative key (must use LiteLLM's `sk-` form);
- `litellm_salt_key` — persistent LiteLLM encryption/hash salt;
- `openrouter_api_key` — upstream OpenRouter credential.

Three platform Kubernetes Secrets are projected with distinct scope:

- `databases/txo-ai-gateway-postgresql` contains only `username` and `password` for CloudNativePG;
- `txo-fabric-system/txo-ai-gateway-runtime` contains the DB, LiteLLM and provider variables consumed by the gateway runtime;
- `txo-fabric-system/txo-ai-gateway-admin` contains only the LiteLLM administrative token consumed by the TXO Fabric operator.

Per-agent model-access Secrets live in their tenant namespace and contain only the
scoped LiteLLM virtual key. Tenant Hindsight reuses its existing `hindsight-runtime`
Secret and adds only its scoped embedding virtual key; it never receives the
OpenRouter provider credential or LiteLLM administrative credential.

## Persistence

LiteLLM uses logical database and role `txo_ai_gateway` on the existing
`databases/postgresql-shared` cluster. The CloudNativePG `Database` and
`DatabaseRole` both use `retain` reclaim policy so usage/accounting and virtual-key
state survive gateway workload replacement.

The gateway role is the owner of its database but is not superuser and cannot
create databases or roles.

## Database schema lifecycle

LiteLLM database migrations are owned by a dedicated Argo CD Sync hook Job rather
than by the serving Deployment. `Job/txo-ai-gateway-migrations` runs with the same
pinned LiteLLM image as the gateway and applies that image's bundled Prisma
migrations before the serving workload is allowed to start.

The ordering is explicit:

```text
wave 1  DatabaseRole
wave 2  Database
wave 3  txo-ai-gateway-migrations
wave 4  txo-ai-gateway Deployment
```

The serving Deployment sets `DISABLE_SCHEMA_UPDATE=true`; this does not freeze the
database schema. On every LiteLLM image upgrade the migration Job is recreated and
runs the migrations bundled with the new image before that version of the gateway
starts. A successful migration Job is deleted by Argo CD as soon as the hook succeeds, so
later syncs do not depend on cleaning up a retained successful hook. `BeforeHookCreation`
remains as a defensive rerun policy for a fixed-name Job, while failed Jobs are retained
for diagnostics.

The migration pod receives only the PostgreSQL username/password from the runtime
Secret. Provider credentials and the LiteLLM master key are not projected into the
migration pod.

## Network boundary

`txo-fabric-system` is default-deny. `NetworkPolicy/txo-ai-gateway-access` allows:

- inbound TCP/4000 from Fabric-managed tenant namespaces, limited to Hermes and Hindsight pods;
- inbound TCP/4000 from the TXO Fabric operator for virtual-key reconciliation;
- DNS egress;
- PostgreSQL egress only to the `postgresql-shared` pods on TCP/5432;
- public HTTPS egress on TCP/443 while excluding private/link-local IPv4 ranges.

The operator egress policy permits only DNS, the cluster-wide kube-apiserver path,
and TCP/4000 to the local AI gateway. Each Hermes egress policy permits DNS,
TCP/4000 to the local AI gateway and, when configured, TCP/8888 to its tenant-local
Hindsight service. Hermes is not granted generic Internet egress for model access.

A gateway-backed Hindsight egress policy permits only DNS, its tenant PostgreSQL
binding, and TCP/4000 to `txo-ai-gateway`. Hindsight receives no generic provider
Internet egress.

`NetworkPolicy/txo-ai-gateway-migration-egress` separately limits the migration
Job to DNS plus TCP/5432 toward `postgresql-shared`. The migration pod uses a
distinct `app.kubernetes.io/name` label and is never selected by
`Service/txo-ai-gateway`.

## Privacy / RGPD boundary

Brokered embeddings change the data boundary: the text sent for embedding leaves
the tenant pod and cluster through the proprietor gateway and is then sent to the
configured upstream provider. The gateway centralizes the provider credential,
routing and accounting, but it does not make that upstream processing local.

Deployment/contract decisions must therefore treat embedding input as customer
data and document the selected provider/subprocessor, processing location,
retention/logging behavior and any required data-processing agreement. The stable
`txo-embedding` alias intentionally keeps those provider choices on the platform
side so a local/on-cluster embedding backend can replace offshore routing later
without changing tenant workloads.

## Physical validation

The gateway and AgentIdentity LLM path were physically validated on grenat for
#3607:

1. the dedicated Prisma migration Job applied all 171 bundled migrations successfully;
2. `Deployment/txo-ai-gateway` became Ready and `/health/readiness` returned 200;
3. a scoped `fabric-smoke-probe` virtual key restricted to `txo-default` was reconciled;
4. the real Hermes pod called `txo-default` through the local gateway and returned `TXO_AGENT_GATEWAY_OK` with HTTP 200;
5. the response reported token usage and cost (`1.52e-05` for the validation call);
6. the AgentIdentity reached `ModelAccessReady=True`, `RuntimeReady=True` and `Ready=True`;
7. the operator no longer requires cluster-wide Secret list/watch access.

The initial Hindsight gateway path was then physically exercised on `fabric-smoke`
with the `hindsight-gateway` profile using the temporary 384-dimensional
`text-embedding-3-small` compatibility route. Retain/recall succeeded, existing
memories remained accessible, scoped key attribution was proven, and #3644 added an
explicit provider rate so persisted embedding spend became positive.

For #3649, the BGE-M3 route was independently proven before changing the stable
alias. A temporary key restricted to `txo-embedding-next` saw only that model;
LiteLLM `/v1/embeddings` returned HTTP 200 with a native 1024-dimensional vector,
17 input tokens and persisted spend `1.7e-07` against upstream
`openrouter/baai/bge-m3`. `/key/info` preserved the expected tenant-independent
validation metadata and the temporary key was deleted afterwards.

The remaining physical cutover is deliberately destructive only for the disposable
`fabric-smoke` canary: export the existing bank, reset that tenant's Hindsight
database, deploy the stable `txo-embedding -> BGE-M3` mapping with automatic
embedding-width detection, import the bank archive, and prove that the known
`quokka`, `axolotl`, and `capybara` memories are recalled after re-embedding.
The hAIrem `hindsight-standard` local path remains a non-regression boundary.
