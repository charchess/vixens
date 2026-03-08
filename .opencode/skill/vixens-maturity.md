---
description: >-
  Vixens 7-Tier Maturity System expert (ADR-023). ALWAYS USE for: maturity labels,
  Bronze/Silver/Gold/Platinum/Emerald/Diamond/Orichalcum upgrades, policy violations,
  Kyverno policyreports, goldification, "why is my app still bronze", require-resources,
  require-probes, InfisicalSecret conversion, sizing labels, PDB, PriorityClass,
  or ANY maturity-related question. Trigger on partial mentions like "upgrade app",
  "fix violations", "maturity", "tier", "level", "goldify".
---

# Vixens Maturity Expert

You are an expert in the Vixens 7-tier maturity system defined in ADR-023.

## 7-Tier System (ADR-023 v2)

| Level | Name | Philosophy |
|-------|------|------------|
| 🥉 1 | **Bronze** | "Déployée" — App runs, accessible |
| 🥈 2 | **Silver** | "Production Ready" — Limits, probes, TLS, secrets |
| 🥇 3 | **Gold** | "Observable" — Metrics, ServiceMonitor, Goldilocks |
| 💎 4 | **Platinum** | "Reliable" — PriorityClass, PDB, sizing justified |
| 🟢 5 | **Emerald** | "Data Durability" — Litestream, Config-Syncer, Velero |
| 💠 6 | **Diamond** | "Secure & Integrated" — PSA, NetworkPolicies, SSO |
| 🌟 7 | **Orichalcum** | "Parfaite" — 7d stability, 0 CVE, validated sizing |

> **Important**: The old "Elite" tier no longer exists. Use the 7-tier system above.

## Quick Checks

### Check Current Maturity
```bash
export KUBECONFIG=/home/charchess/vixens/.secrets/prod/kubeconfig-prod
kubectl get deployments -A -o json | jq -r '.items[] | "\(.metadata.namespace)/\(.metadata.name): \(.metadata.labels["vixens.io/maturity"] // "unlabeled")"'
```

### Check Maturity Distribution
```bash
kubectl get deployments -A -o json | jq -r '[.items[] | .metadata.labels["vixens.io/maturity"] // "unlabeled"] | group_by(.) | map({level: .[0], count: length}) | sort_by(.level) | .[]'
```

### Check Bronze Apps (Blocking Silver)
```bash
kubectl get deployments -A -l "vixens.io/maturity=bronze" -o custom-columns='NAMESPACE:.metadata.namespace,NAME:.metadata.name,MISSING:.metadata.labels.vixens\.io/maturity-missing'
```

### Check Policy Violations for an App
```bash
kubectl get policyreport -n $NAMESPACE -o json | jq -r '[.items[].results[] | select(.result == "fail") | {policy: .policy, message: .message}] | unique_by(.policy) | .[] | "\(.policy): \(.message[0:100])"'
```

### Check Silver-Blocking Violations Only
```bash
kubectl get policyreport -n $NAMESPACE -o json | jq '[.items[].results[] | select(.result == "fail") | {policy: .policy, message: .message}] | unique_by(.policy) | .[] | select(.message | test("Silver|Level 2"; "i"))'
```

## Silver Requirements (Level 2)

| Requirement | Policy | How to Fix |
|-------------|--------|------------|
| Resources | `require-resources` | Add `resources.limits` + `requests` on ALL containers |
| Probes | `require-probes` | Add liveness + readiness on ALL containers |
| Secrets | `check-infisical-secrets` | Use InfisicalSecret, not hardcoded Secret |
| TLS | `check-ingress-tls` | Ensure ingress has `spec.tls` block |
| Startup probe | `check-startup-probe` | Add or bypass with `vixens.io/fast-start: "true"` |

### Fix Resources
```yaml
resources:
  requests:
    cpu: 100m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 512Mi
```

### Fix Probes (on sidecars too!)
```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 30
  failureThreshold: 3
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
```

### For sidecars without HTTP endpoints
```yaml
livenessProbe:
  exec:
    command: ["pgrep", "-f", "process-name"]
  initialDelaySeconds: 10
  periodSeconds: 30
readinessProbe:
  exec:
    command: ["test", "-f", "/tmp/ready"]
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Bypass Annotations

| Annotation | Bypasses | Use When |
|------------|----------|----------|
| `vixens.io/fast-start: "true"` | Startup probe | Container starts < 5s |
| `vixens.io/no-long-connections: "true"` | preStop hook | No long-running connections |
| `vixens.io/nometrics: "true"` | Metrics/ServiceMonitor | No metrics endpoint |
| `vixens.io/explicitly-allow-root: "true"` | SecurityContext | Must run as root |

## Shared Components

```yaml
# In kustomization.yaml
components:
  - ../../../../_shared/components/gold-maturity      # sync-wave, goldilocks, revisionHistoryLimit
  - ../../../../_shared/components/priority/high      # PriorityClass
  - ../../../../_shared/components/poddisruptionbudget/1  # PDB minAvailable=1
  - ../../../../_shared/components/probes/basic       # Standard probes
```

## Maturity Controller

- Runs every 15 minutes as CronJob in kyverno namespace
- Updates `vixens.io/maturity` label based on policy violations
- Check last run: `kubectl -n kyverno get jobs -l app=maturity-controller --sort-by='.metadata.creationTimestamp' | tail -5`
- Manual trigger: `kubectl -n kyverno create job maturity-manual --from=cronjob/maturity-controller`

## Common Upgrade Path

### Bronze → Silver
1. Add resources on ALL containers (including sidecars like litestream, config-syncer)
2. Add probes on ALL containers (including sidecars)
3. Ensure secrets via InfisicalSecret (not hardcoded Secret)
4. Add `app.kubernetes.io/managed-by: argocd` label
5. Ensure Ingress has TLS block

### Silver → Gold
1. Add `goldilocks.fairwinds.com/enabled: "true"` annotation
2. Add `revisionHistoryLimit: 3`
3. Add sync-wave annotation
4. Add ServiceMonitor if metrics exposed
5. Add prometheus annotations: `prometheus.io/scrape: "true"`, `prometheus.io/port: "8080"`

### Gold → Platinum
1. Add PriorityClass (use shared component)
2. Add PDB for multi-replica deployments
3. Add `vixens.io/sizing-rationale` annotation
4. Review Goldilocks recommendations
5. Add sizing labels: `vixens.io/sizing.<container>: B-small`

### Platinum → Emerald
1. Configure backups (Litestream for SQLite, Velero for PVCs)
2. Add `vixens.io/backup-profile: standard` label
3. Add Config-Syncer for configuration data

### Emerald → Diamond
1. Add NetworkPolicy
2. Harden SecurityContext (runAsNonRoot, drop capabilities)
3. Integrate with Authentik SSO if applicable

### Diamond → Orichalcum
1. 7 days stability without restarts
2. 0 CVE in container images
3. Validated sizing from Goldilocks
