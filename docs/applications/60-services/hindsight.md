# Hindsight

## Purpose

Hindsight provides long-term memory services for Hermes personas: retain, recall,
consolidation, graph/entity resolution, and reflect synthesis.

## Runtime configuration

The laboratory Helm values are in:

```text
apps/60-services/hindsight/values-umi.yaml
```

The API runtime uses OpenRouter with deliberately separated operations:

- **retain:** `openai/gpt-oss-20b`, structured extraction, low reasoning,
  temperature `0.1`, maximum one concurrent operation;
- **consolidation:** `openai/gpt-oss-20b`, low reasoning, temperature `0.0`,
  maximum one concurrent operation;
- **reflect:** `openai/gpt-oss-120b`, high reasoning, temperature `0.2`,
  maximum one concurrent operation;
- **embeddings:** `baai/bge-m3` through OpenRouter, native 1024-dimensional
  vectors, 8,000-token input cap and no E5 prefixes.

The shared LLM concurrency ceiling is three. Per-operation limits reserve
capacity and prevent background retain/consolidation from starving reflect.

## Secret handling

`hindsight-runtime` supplies `HINDSIGHT_API_LLM_API_KEY` at runtime. OpenRouter
credentials must be stored in OpenBao and materialized into the Kubernetes
runtime secret only through the approved secret-delivery path. They must never
be committed to Git values or documentation.

## Deployment and migration gate

Changing an embeddings backend can invalidate an existing vector index even
when its nominal dimensions match. Before enabling this configuration against an
existing bank, confirm the previous endpoint served the same BGE-M3 checkpoint
and returns compatible 1024-dimensional vectors. Otherwise export/re-embed the
bank before switching traffic.

Deployment validation is currently blocked until the Vixens cluster control
plane and Hindsight workload are restored. After restoration, verify the Hindsight
health endpoint, one retain/recall cycle, a reflect request, and BGE-M3 vector
dimension before declaring the change operational.
