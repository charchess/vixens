# TXO AI egress gateway

This directory contains the proprietor-managed AI egress boundary for TXO Fabric.
It is deliberately separate from tenant cells: tenant runtimes must not receive
upstream provider credentials.

## Architecture

TXO Fabric uses LiteLLM as an OpenAI-compatible gateway in `txo-fabric-system`:

```text
AgentIdentity controller
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
    | scoped LiteLLM virtual key
    | model = txo-default
    v
Service/txo-ai-gateway:4000/v1
```

Hindsight keeps the local embedding/reranker path already validated physically in
#3602. Remote Hindsight embeddings are not part of this gateway slice.

The gateway exposes one stable local model alias, `txo-default`. The initial
upstream is `openrouter/openai/gpt-5.6-luna`; tenant declarations bind to the local
alias rather than to this provider/model identifier so routing can change centrally
later.

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
scoped LiteLLM virtual key.

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
starts. The completed Job is kept for diagnostics until the next sync, when
`BeforeHookCreation` replaces it.

The migration pod receives only the PostgreSQL username/password from the runtime
Secret. Provider credentials and the LiteLLM master key are not projected into the
migration pod.

## Network boundary

`txo-fabric-system` is default-deny. `NetworkPolicy/txo-ai-gateway-access` allows:

- inbound TCP/4000 from Fabric-managed tenant namespaces, limited to Hermes agent pods;
- inbound TCP/4000 from the TXO Fabric operator for virtual-key reconciliation;
- DNS egress;
- PostgreSQL egress only to the `postgresql-shared` pods on TCP/5432;
- public HTTPS egress on TCP/443 while excluding private/link-local IPv4 ranges.

The operator egress policy permits only DNS, the cluster-wide kube-apiserver path,
and TCP/4000 to the local AI gateway. Each Hermes egress policy permits DNS,
TCP/4000 to the local AI gateway and, when configured, TCP/8888 to its tenant-local
Hindsight service. Hermes is not granted generic Internet egress for model access.

`NetworkPolicy/txo-ai-gateway-migration-egress` separately limits the migration
Job to DNS plus TCP/5432 toward `postgresql-shared`. The migration pod uses a
distinct `app.kubernetes.io/name` label and is never selected by
`Service/txo-ai-gateway`.

## Physical validation

The gateway foundation was physically validated on grenat for #3607:

1. the dedicated Prisma migration Job applied all 171 bundled migrations successfully;
2. `Deployment/txo-ai-gateway` became Ready and `/health/readiness` returned 200;
3. a scoped virtual key restricted to `txo-default` was generated;
4. that key reached `openrouter/openai/gpt-5.6-luna` through the local gateway and returned a successful completion;
5. tenant/agent metadata was preserved on the key;
6. the request cost was persisted in LiteLLM spend logs asynchronously.

The AgentIdentity integration must now be physically validated by allowing a real
Hermes runtime such as `fabric-smoke-probe` to leave `AuthBlocked`, call
`txo-default` through the gateway, and produce tenant/agent-attributed spend without
receiving any upstream provider credential.
