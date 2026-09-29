# TXO AI egress gateway

This directory contains the proprietor-managed AI egress boundary for TXO Fabric.
It is deliberately separate from tenant cells: tenant runtimes must not receive
upstream provider credentials.

## First implementation slice

The first slice uses LiteLLM as an OpenAI-compatible gateway in
`txo-fabric-system`:

```text
Hermes (later in #3607)
    |
    | scoped LiteLLM virtual key
    v
Service/txo-ai-gateway:4000
    |
    +--> PostgreSQL txo_ai_gateway (keys, spend, budgets/rate-limit state)
    |
    +--> OpenRouter (provider credential exists only here)
```

Hindsight keeps the local embedding/reranker path already validated physically in
#3602. Remote Hindsight embeddings are not part of this first gateway slice.

The gateway exposes one stable local model alias, `txo-default`. The initial
upstream is `openrouter/openai/gpt-5.6-luna`; tenant declarations will bind to the
local alias rather than to this provider/model identifier so routing can change
centrally later.

## Secrets

No secret value is stored in Git. `ExternalSecret` resources read
`vixens/prod/apps/60-services/txo-ai-gateway` from `ClusterSecretStore/openbao`.
The OpenBao object must contain these properties:

- `postgres_username` — expected to be `txo_ai_gateway`;
- `postgres_password` — gateway database role password;
- `litellm_master_key` — LiteLLM administrative key (must use LiteLLM's `sk-` form);
- `litellm_salt_key` — persistent LiteLLM encryption/hash salt;
- `openrouter_api_key` — upstream OpenRouter credential.

Two Kubernetes Secrets are projected with least necessary scope:

- `databases/txo-ai-gateway-postgresql` contains only `username` and `password` for CloudNativePG;
- `txo-fabric-system/txo-ai-gateway-runtime` contains the runtime variables consumed by LiteLLM.

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
- inbound TCP/4000 from the TXO Fabric operator for future virtual-key reconciliation;
- DNS egress;
- PostgreSQL egress only to the `postgresql-shared` pods on TCP/5432;
- public HTTPS egress on TCP/443 while excluding private/link-local IPv4 ranges.

`NetworkPolicy/txo-ai-gateway-migration-egress` separately limits the migration
Job to DNS plus TCP/5432 toward `postgresql-shared`. The migration pod uses a
distinct `app.kubernetes.io/name` label and is never selected by
`Service/txo-ai-gateway`.

The AgentIdentity integration slice must restrict each Hermes runtime to this
Service instead of granting generic Internet egress.

## Physical acceptance for this foundation

Before integrating AgentIdentity reconciliation, validate on grenat that:

1. both ExternalSecrets are Ready without printing secret values;
2. the CNPG DatabaseRole and Database are applied;
3. `Job/txo-ai-gateway-migrations` completes successfully and the LiteLLM schema is present;
4. `Deployment/txo-ai-gateway` is Ready;
5. `/health/readiness` succeeds;
6. a LiteLLM virtual key can be generated using the master credential without displaying either key;
7. that virtual key can call model `txo-default` through the gateway;
8. LiteLLM records the request against its PostgreSQL-backed key/spend state.

The next slice of #3607 will create/reconcile one scoped virtual key per
`AgentIdentity`, inject only that virtual key into Hermes, point Hermes at
`http://txo-ai-gateway.txo-fabric-system.svc:4000/v1`, and replace the current
`AuthBlocked` status with a real ModelAccessReady condition after validation.
