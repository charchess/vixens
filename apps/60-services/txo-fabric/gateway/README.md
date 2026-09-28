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

## Network boundary

`txo-fabric-system` is default-deny. `NetworkPolicy/txo-ai-gateway-access` allows:

- inbound TCP/4000 from Fabric-managed tenant namespaces, limited to Hermes agent pods;
- inbound TCP/4000 from the TXO Fabric operator for future virtual-key reconciliation;
- DNS egress;
- PostgreSQL egress only to the `postgresql-shared` pods on TCP/5432;
- public HTTPS egress on TCP/443 while excluding private/link-local IPv4 ranges.

The AgentIdentity integration slice must restrict each Hermes runtime to this
Service instead of granting generic Internet egress.

## Physical acceptance for this foundation

Before integrating AgentIdentity reconciliation, validate on grenat that:

1. both ExternalSecrets are Ready without printing secret values;
2. the CNPG DatabaseRole and Database are applied;
3. `Deployment/txo-ai-gateway` is Ready;
4. `/health/readiness` succeeds;
5. a LiteLLM virtual key can be generated using the master credential without displaying either key;
6. that virtual key can call model `txo-default` through the gateway;
7. LiteLLM records the request against its PostgreSQL-backed key/spend state.

The next slice of #3607 will create/reconcile one scoped virtual key per
`AgentIdentity`, inject only that virtual key into Hermes, point Hermes at
`http://txo-ai-gateway.txo-fabric-system.svc:4000/v1`, and replace the current
`AuthBlocked` status with a real ModelAccessReady condition after validation.
