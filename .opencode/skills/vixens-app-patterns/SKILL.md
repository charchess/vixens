---
name: vixens-app-patterns
description: >-
  Guide for discovering and applying current Vixens application patterns.
  Use when adding or refactoring a Kubernetes application, choosing native manifests
  vs Helm, persistence, ingress, secrets, NetworkPolicy, monitoring or ArgoCD wiring.
license: MIT
compatibility: opencode
metadata:
  domain: kubernetes-gitops
  audience: homelab-operators
---

# Vixens application patterns

This skill is deliberately **not a template library**.

Vixens evolves quickly; embedded copies of manifests become stale and previously reintroduced retired technologies. The canonical patterns are the manifests currently deployed under `apps/` and `argocd/`.

Read `WORKFLOW.md` and `AGENTS.md` first.

## Discovery workflow

Before writing a new application:

1. identify its category and runtime model;
2. search `apps/` for two or three similar active applications;
3. prefer examples changed recently over old reports/templates;
4. inspect both `base/` and `overlays/{dev,prod}`;
5. inspect the matching ArgoCD Application;
6. identify storage, network, secret and observability dependencies;
7. only then implement the smallest pattern that fits.

Useful searches:

```bash
# External Secrets pattern
rg 'kind: ExternalSecret' apps
rg 'name: openbao' apps

# Cilium policies
rg 'kind: CiliumNetworkPolicy' apps

# ServiceMonitor
rg 'kind: ServiceMonitor' apps

# Stateful / Recreate
rg 'kind: StatefulSet|type: Recreate' apps

# ArgoCD multi-source / Helm
rg 'sources:|chart:' argocd/overlays
```

## Decision points

### Native manifests or Helm?

Use upstream Helm when it is maintained and reduces local ownership. Use native manifests/Kustomize for Vixens-owned workloads or when the chart creates more complexity than it removes.

Do not fork/copy an upstream chart into the repository just to avoid multi-source ArgoCD.

### Stateless or persistent?

Ask what must survive:

- pod restart;
- node loss;
- PVC loss;
- cluster rebuild.

Then choose the storage/backup pattern from `docs/reference/configuration-management-strategy.md`.

### Secrets

Canonical model:

```text
OpenBao → ClusterSecretStore/openbao → ExternalSecret → Secret → workload
```

Use `external-secrets.io/v1`. Never create new `InfisicalSecret` resources.

### Ingress / TLS

Copy the current Traefik/cert-manager convention from a recently maintained application of the same exposure type (public, authenticated, internal).

Do not embed a second HTTP→HTTPS redirect pattern if Traefik already provides it globally.

### NetworkPolicy

Cilium/default-deny means flows should be explicit. Define what the app needs instead of starting with allow-all. Use Hubble/Loki observations to validate uncertain flows.

### Observability

Add only useful signals:

- ServiceMonitor for actionable metrics;
- logs through the central pipeline;
- dashboards when they answer an operational question;
- alerts with a clear response/runbook.

## Base/overlay guideline

Keep environment-independent resources in `base/`; patch environment differences in overlays.

Typical differences:

- hostname / TLS secret;
- remote OpenBao path (`dev` vs `prod`);
- replica count / resources where justified;
- environment-specific integrations.

Avoid duplicating entire manifests between dev and prod.

## Validation

Before PR:

```bash
kustomize build apps/<category>/<app>/overlays/dev >/tmp/app-dev.yaml
kustomize build apps/<category>/<app>/overlays/prod >/tmp/app-prod.yaml

grep '^kind:' /tmp/app-dev.yaml | sort
grep '^kind:' /tmp/app-prod.yaml | sort
```

Check:

- expected objects are rendered;
- selectors/labels match;
- Services target real ports;
- no secret values are present;
- ExternalSecret remote paths are correct per environment;
- network policies match intended flows;
- PVC strategy is compatible with rollout semantics.

Then use the normal Issue → PR → CI → ArgoCD dev → validation → promotion workflow.

## Why there are no bundled templates here

Old versions of this skill embedded full examples/templates. They became stale copies of Vixens and continued teaching removed technologies such as Infisical.

The repository itself is now the pattern library. This keeps agents aligned with current code and reduces duplicated sources of truth.

## References

- `docs/guides/adding-new-application.md`
- `docs/procedures/deployment-standard.md`
- `docs/guides/secret-management.md`
- `docs/reference/configuration-management-strategy.md`
