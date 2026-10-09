# TXO AI egress gateway

This directory contains the proprietor-managed AI egress boundary for TXO Fabric.
It is deliberately separate from tenant cells: tenant runtimes must not receive
upstream provider credentials.

## Architecture

The canonical #3885 topology is one tenant-local LiteLLM facade per Active
TenantBundle. CLIProxyAPI (CPA) is the **internal broker for all Hermes
conversation providers** (Codex OAuth and other provider accounts/API keys
only where supported by the pinned CPA version). It is not the consumer-facing
inference endpoint.

```text
Hermes -----------------------+
                              |
Tenant Hindsight -------------+--> tenant LiteLLM
                                   +-- txo-agent ----> tenant CPA ----> Codex OAuth
                                   |                        +-> other approved chat provider*
                                   +-- txo-embedding -> OpenRouter/BGE-M3 (Hindsight)

* Other chat providers (including OpenRouter) must be connected via CPA and
  tested against its pinned version; they are not direct LiteLLM chat routes.
```

Consumers see logical capabilities and scoped LiteLLM virtual keys. They never
receive the CPA management credential or an upstream provider API key.

The historical `txo-fabric-system/txo-ai-gateway` remains only as a bounded
migration source for Hindsight instances that have not yet received a
tenant-specific OpenRouter credential. It is not the canonical AgentIdentity path
and is not a fallback after a Hindsight instance has completed tenant-local
cutover.

### Provider credential boundary

OpenRouter credentials are tenant-specific platform secrets. Fabric watches:

```text
txo-fabric-system/txo-ai-provider-<tenant>
  key: OPENROUTER_API_KEY
```

The value is projected into the platform-owned
`tenant-<tenant>/txo-ai-gateway-runtime` Secret consumed by LiteLLM. The value
never enters Hermes, Hindsight, TenantBundle/AgentIdentity spec or status, Git, or
logs. In the existing v0.1 implementation the source Secret has been
enrolled manually by a platform admin as a **temporary bootstrap mechanism**,
not a customer-facing credential management policy. Fabric WebUI #3979 must
provide permissioned self-service for tenant/user credentials; #3980 defines
the effective Fabric Admin -> Tenant Admin -> User policy and ownership model.
OpenBao/ESO may be introduced later, but is not required by the *current*
service-embedding runtime contract.

When the credential is absent, the tenant LiteLLM remains fully usable for
`txo-agent` through CPA but does not advertise `txo-embedding`. When present,
the tenant gateway adds:

```yaml
model_name: txo-embedding
model: openrouter/baai/bge-m3
```

and receives outbound TCP/443. Tenant workloads themselves receive no generic
provider egress.

### Manual OpenRouter enrollment (v0.1)

**Historical v0.1 bootstrap and break-glass runbook only.** The sequence below
documents the current source Secret contract for trusted platform operators;
it must **not** become the normal UX for tenant admins or users, nor an
instruction to onboard Indiba without a voluntary account/provider decision.
See #3979 (WebUI) and #3980 (hierarchical policy and payer/owner scoping).

The current implementation does not need a dedicated OpenBao deployment or
ExternalSecret to *consume* this source Secret. Its platform-managed onboarding
procedure remains manual until the Fabric WebUI is implemented. The operator
watches `Secret/txo-fabric-system/txo-ai-provider-<tenant>` and copies its
`OPENROUTER_API_KEY` only into the corresponding tenant LiteLLM runtime
Secret. This key is for **Hindsight service embedding**, never a backdoor for
Hermes chat.

Create **distinct OpenRouter API keys** for `hairem` and `indiba`. Use
provider-side spending limits/revocation per key where supported. The provider
values are never stored in Git, commits, PR comments, or `TenantBundle` spec.

```bash
# Run interactively as a cluster administrator, from grenat.
# Repeat with TENANT=indiba and a DIFFERENT OpenRouter API key.
TENANT=hairem
(
  set -euo pipefail
  read -r -s -p "OpenRouter API key for ${TENANT}: " provider_key
  printf '\n'
  test -n "$provider_key"
  printf '%s' "$provider_key" |
    kubectl -n txo-fabric-system create secret generic "txo-ai-provider-${TENANT}" \
      --from-file=OPENROUTER_API_KEY=/dev/stdin --dry-run=client -o yaml |
    kubectl -n txo-fabric-system apply --server-side \
      --field-manager=txo-fabric-provider-enrollment -f -
  unset provider_key
)

# Check presence without printing the value or its base64 encoding.
kubectl -n txo-fabric-system get secret "txo-ai-provider-${TENANT}" -o json |
  jq -r 'if (.data.OPENROUTER_API_KEY // "") != "" then "PROVIDER_KEY_PRESENT" else "PROVIDER_KEY_MISSING" end'
```

The same invocation performs first enrollment and rotation. It reads the key
silently into a non-exported subshell variable; the key is supplied to
`kubectl` over stdin, not a command-line argument, and is not printed.
`--server-side` avoids the client-side apply `last-applied-configuration`
annotation containing a second copy of the Secret data. The Kubernetes API
still stores the provider key as a Secret (which requires appropriate etcd
protection, access controls and backup); it is an **intentional credential
bootstrap exception** to GitOps, not a license to patch tenant workloads.

**Do not enroll until the reviewed #3925 code (and the #3928 fail-closed
follow-up if merged) is in an explicitly approved, pinned operator image and
the release has converged on the cluster.** The old shared Hindsight endpoint
must remain available to existing tenants during the transition. Because the
operator watches source Secret changes, importing or rotating the key should
trigger its LiteLLM config/runtime reconciliation and rollout automatically.
No manual edit of LiteLLM, Hindsight, Hermes, or NetworkPolicy is necessary.

Then validate **hAIrem first, Indiba second**:

1. Check the provider Secret key *exists* (never display its value); the
   tenant LiteLLM has `txo-embedding` and is Ready.
2. Confirm Hindsight's embedding URL targets
   `http://txo-ai-gateway.tenant-<tenant>.svc:4000/v1`, with an embedding-only
   LiteLLM key, no OpenRouter credential, and
   `HINDSIGHT_API_LLM_PROVIDER=none`.
3. Prove a real BGE-M3 embedding request, plus Hermes `txo-agent` regression.
   Ensure Hermes -> CPA remains blocked.
4. Check the cutover finalizes only after Hindsight readiness: historical
   shared key revoked; shared-gateway Hindsight egress removed.
5. Confirm LiteLLM embedding metering and cross-tenant credential isolation.

On rotation, enroll a new OpenRouter key using the same command and confirm
the gateway rollout and live embeddings **before revoking the old key** at
OpenRouter. Deleting the Kubernetes Secret does not revoke an upstream
credential; and once Hindsight is migrated, loss of the provider credential
intentionally fails closed. Back up/enroll the provider key again during
disaster recovery: it is not reconstructed from Git.

Do not close #3885 or claim physical acceptance on CI alone.

## AgentIdentity model access

Every Active-tenant AgentIdentity uses its tenant-local LiteLLM facade. Fabric
creates a deterministic virtual key scoped to `txo-agent` and stores only that
key in `Secret/hermes-<agentKey>-model-access`.

Hermes receives:

- `OPENAI_BASE_URL=http://txo-ai-gateway.tenant-<tenant>.svc:4000/v1`;
- `OPENAI_API_KEY` from the scoped model-access Secret;
- `HERMES_MODEL=txo-agent`;
- `TXO_LLM_AUTH_MODE=gateway`.

The `txo-agent` route resolves LiteLLM -> tenant CPA -> an authorized chat provider (currently Codex OAuth proven on hAIrem). Direct
Hermes -> CPA access remains denied by network policy.

## Hindsight embedding access

`HindsightProfile.spec.llmAuthMode=PlatformGateway` enables only governed
embedding inference. Hindsight generative/reflection LLM processing remains
disabled with `HINDSIGHT_API_LLM_PROVIDER=none`.

The canonical runtime contract is:

- `HINDSIGHT_API_EMBEDDINGS_PROVIDER=openai`;
- `HINDSIGHT_API_EMBEDDINGS_OPENAI_BASE_URL=http://txo-ai-gateway.tenant-<tenant>.svc:4000/v1`;
- `HINDSIGHT_API_EMBEDDINGS_OPENAI_MODEL=txo-embedding`;
- `HINDSIGHT_API_EMBEDDINGS_OPENAI_API_KEY=<Hindsight-scoped LiteLLM key>`;
- `HINDSIGHT_API_EMBEDDINGS_OPENAI_DIMENSIONS=` so Hindsight auto-detects width.

The virtual key is restricted to `txo-embedding` and carries tenant metadata plus
`component=hindsight` and `capability=embeddings`. It cannot authorize
`txo-agent` or expose the provider credential.

### Failure-safe migration from the historical shared gateway

Existing Hindsight runtimes are migrated without revoking their working source
credential first:

1. wait until the tenant OpenRouter credential exists and the tenant LiteLLM
   `txo-embedding` route is deployed and Available;
2. mint the destination Hindsight virtual key on tenant LiteLLM;
3. update `hindsight-runtime` to the tenant URL/key and roll the Deployment;
4. only after the Deployment has adopted the new Secret revision and is Available,
   revoke the historical shared alias;
5. remove shared-gateway egress from the Hindsight NetworkPolicy.

During the bounded transition, the Hindsight policy permits both old and new
gateways. Before migration it permits shared only; after finalization it permits
tenant LiteLLM only. If the tenant OpenRouter credential later disappears, Fabric
fails closed and never silently migrates Hindsight back to the shared gateway.

The embedding backend remains BAAI/BGE-M3 through the stable `txo-embedding`
logical alias. Existing vector-width migration constraints remain a Hindsight
data concern; provider credentials and concrete model identifiers stay behind the
gateway boundary.

## Inference policy and credential lifecycle

The active TXO Fabric external-inference consumers are deliberately small:

- Hermes uses the logical chat/model alias `txo-agent` through a per-`AgentIdentity`
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
