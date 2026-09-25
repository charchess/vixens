---
name: vixens-maturity
description: >-
  Vixens maturity-system guide. Use for maturity labels, Kyverno PolicyReports,
  resource/probe/TLS/secret requirements, sizing, PDB/PriorityClass, observability,
  backups, network policy, SSO and maturity upgrades.
license: MIT
compatibility: opencode
metadata:
  domain: kubernetes
  audience: homelab-operators
---

# Vixens maturity

Follow `WORKFLOW.md` and `AGENTS.md`. This skill summarizes the maturity model; current policies/manifests remain the executable source of truth.

## Levels

The current 7-tier model is defined in ADR-023:

| Level | Name | Intent |
|---|---|---|
| 1 | Bronze | deployed |
| 2 | Silver | production-ready basics |
| 3 | Gold | observable |
| 4 | Platinum | reliable |
| 5 | Emerald | durable data |
| 6 | Diamond | secure/integrated |
| 7 | Orichalcum | validated/stable |

Do not manually force a maturity label to hide violations. Fix the desired state and let the policies/controller evaluate it.

## Diagnose first

```bash
kubectl get deployments -A -o json | jq -r '.items[] | "\(.metadata.namespace)/\(.metadata.name) \(.metadata.labels["vixens.io/maturity"] // "unlabeled")"'

kubectl get policyreport -n <namespace> -o json | \
  jq -r '[.items[].results[] | select(.result == "fail") | {policy, message}] | unique_by(.policy) | .[]'
```

Runtime inspection is fine; persistent fixes go through Git/PR/ArgoCD.

## Common requirements

### Resources and probes

Use real workload behavior and current shared components/policies. Avoid copy-pasting stale sizing values from docs.

### Secrets

The active secret pattern is **OpenBao + External Secrets Operator**, not Infisical.

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
spec:
  secretStoreRef:
    name: openbao
    kind: ClusterSecretStore
```

Never replace an `ExternalSecret` with a plaintext Kubernetes `Secret` in Git to satisfy a maturity check.

### TLS

Use the current Traefik/cert-manager convention in the app's environment. Do not infer issuer/middleware details from historical docs.

### Observability

Add useful metrics, ServiceMonitor, dashboards and alerts based on operational needs. A dashboard that nobody can act on is not maturity by itself.

### Reliability

Use PDB/PriorityClass/topology constraints where the workload architecture actually benefits. A single-replica RWO application cannot gain HA merely by adding a PDB.

### Durability

Choose the backup mechanism by data type:

- SQLite → Litestream/DataAngel pattern when applicable;
- PostgreSQL → CloudNativePG mechanisms;
- mutable files → tested filesystem backup pattern;
- shared media → storage/NAS strategy.

Credentials for backup tooling still come from OpenBao/ESO.

### Security

Use Cilium policies, Pod Security/securityContext and SSO/ForwardAuth according to threat model and application capabilities. Observe Hubble drops before broadening network access.

## Upgrade workflow

1. Read current app manifests and policies.
2. List actual failed PolicyReport checks.
3. Group fixes into a small GitHub issue/PR.
4. Build Kustomize/CI.
5. Merge and wait for ArgoCD dev.
6. Validate application behavior.
7. Re-check PolicyReports/maturity.
8. Promote only after dev validation.

Do not perform broad "goldification" by blindly copying manifests from another app.

## Anti-patterns

- hardcoded secrets;
- legacy `InfisicalSecret` examples;
- `kubectl apply/edit` as permanent fix;
- manually setting maturity labels;
- adding permissive NetworkPolicies to silence Hubble;
- PDB/replicas that conflict with RWO storage;
- fake probes that always return success;
- copying agent templates instead of examining current `apps/` patterns.

## References

- `docs/adr/023-*` — maturity decision
- `docs/guides/secret-management.md`
- `docs/reference/configuration-management-strategy.md`
- current Kyverno policies and shared components in the repository
