---
name: vixens-maturity
description: >-
  Vixens 7-Tier Maturity System expert (ADR-023). ALWAYS USE for: maturity labels,
  Bronze/Silver/Gold/Platinum/Emerald/Diamond/Orichalcum upgrades, policy violations,
  Kyverno policyreports, goldification, "why is my app still bronze", require-resources,
  require-probes, InfisicalSecret conversion, sizing labels, PDB, PriorityClass,
  or ANY maturity-related question. Trigger on partial mentions like "upgrade app",
  "fix violations", "maturity", "tier", "level", "goldify".
license: MIT
compatibility: opencode
metadata:
  domain: kubernetes
  audience: homelab-operators
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

---

## ⚙️ Configuration Modalities (GitOps + DRY)

### How Each Configuration is Managed

| Configuration | Managed By | Location | DRY Pattern |
|--------------|------------|----------|-------------|
| **PriorityClass** | Shared Component | `_shared/components/priority/{critical,high,medium,low}` | Include component in kustomization.yaml |
| **PDB** | Shared Component | `_shared/components/poddisruptionbudget/{1,2}` | Include component in kustomization.yaml |
| **Probes** | Shared Component or Manual | `_shared/components/probes/basic` | Component for standard apps, manual for custom endpoints |
| **Resources** | Manual + Goldilocks | App's deployment.yaml | Start with estimates, tune via Goldilocks VPA |
| **Sizing Labels** | Mutable Kyverno | Auto-injected by policy | Kyverno mutates based on resource values |
| **Maturity Label** | Maturity Controller | Auto-updated every 15 min | CronJob evaluates PolicyReports |
| **Sync Waves** | Shared Component | `_shared/components/gold-maturity` | Includes sync-wave + revisionHistoryLimit |
| **Goldilocks** | Shared Component | `_shared/components/gold-maturity` | Enables VPA recommendations |
| **Secrets** | InfisicalSecret | App's infisicalsecret.yaml | Never hardcode, always external |
| **TLS** | Ingress + cert-manager | App's ingress.yaml | `spec.tls` block + Certificate resource |
| **NetworkPolicy** | Manual | App's networkpolicy.yaml | Per-app custom rules |
| **Backup Profile** | Label + Velero | App's deployment labels | `vixens.io/backup-profile: standard` |

### Shared Components (DRY Principle)

**Never copy-paste. Use components:**

```yaml
# apps/<app>/overlays/prod/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

components:
  # Gold requirements (sync-wave, goldilocks, revisionHistoryLimit)
  - ../../../../_shared/components/gold-maturity
  
  # Platinum requirements
  - ../../../../_shared/components/priority/high           # PriorityClass
  - ../../../../_shared/components/poddisruptionbudget/1   # PDB minAvailable=1
  
  # Optional: Standard probes if app doesn't define custom ones
  - ../../../../_shared/components/probes/basic
```

### Mutable Kyverno Policies

**Some configurations are auto-injected:**

| Policy | What it Does | You Provide | Kyverno Adds |
|--------|-------------|-------------|--------------|
| `mutate-sizing-labels` | Adds sizing classification | `resources.requests` | `vixens.io/sizing.<container>: B-small` |
| `mutate-managed-by` | Adds ArgoCD label | Nothing | `app.kubernetes.io/managed-by: argocd` |

**These are automatic — you don't need to add these labels manually.**

### Maturity Controller (Auto-Evaluation)

```
Every 15 minutes:
  1. CronJob runs in kyverno namespace
  2. Reads PolicyReports for each deployment
  3. Counts violations by tier
  4. Updates vixens.io/maturity label
  
Result: Labels are always accurate to current state
```

**You don't manually set maturity labels.** Fix violations → Controller promotes automatically.

---

## Quick Checks

### Check Current Maturity
```bash
export KUBECONFIG=.secrets/prod/kubeconfig-prod
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

---

## Silver Requirements (Level 2)

| Requirement | Policy | How to Fix (GitOps) |
|-------------|--------|---------------------|
| Resources | `require-resources` | Add `resources.limits` + `requests` in deployment.yaml |
| Probes | `require-probes` | Add probes in deployment.yaml OR use `probes/basic` component |
| Secrets | `check-infisical-secrets` | Replace `Secret` with `InfisicalSecret` |
| TLS | `check-ingress-tls` | Add `spec.tls` block in ingress.yaml |
| Startup probe | `check-startup-probe` | Add startupProbe OR bypass annotation |

### Fix Resources (in Git)
```yaml
# apps/<app>/base/deployment.yaml
resources:
  requests:
    cpu: 100m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 512Mi
```

### Fix Probes (Option 1: Component)
```yaml
# apps/<app>/overlays/prod/kustomization.yaml
components:
  - ../../../../_shared/components/probes/basic
```

### Fix Probes (Option 2: Custom in Git)
```yaml
# apps/<app>/base/deployment.yaml
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

### For Sidecars Without HTTP Endpoints
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

---

## Bypass Annotations

| Annotation | Bypasses | Use When |
|------------|----------|----------|
| `vixens.io/fast-start: "true"` | Startup probe | Container starts < 5s |
| `vixens.io/no-long-connections: "true"` | preStop hook | No long-running connections |
| `vixens.io/nometrics: "true"` | Metrics/ServiceMonitor | No metrics endpoint |
| `vixens.io/explicitly-allow-root: "true"` | SecurityContext | Must run as root |

---

## Maturity Controller

- Runs every 15 minutes as CronJob in kyverno namespace
- Updates `vixens.io/maturity` label based on policy violations
- Check last run: `kubectl -n kyverno get jobs -l app=maturity-controller --sort-by='.metadata.creationTimestamp' | tail -5`
- Manual trigger: `kubectl -n kyverno create job maturity-manual --from=cronjob/maturity-controller`

---

## Common Upgrade Path (GitOps)

### Bronze → Silver

**All changes in Git, not kubectl:**

1. **Resources** — Edit `deployment.yaml`, add `resources` block
2. **Probes** — Add probes OR include `probes/basic` component
3. **Secrets** — Replace `Secret` with `InfisicalSecret`
4. **TLS** — Add `spec.tls` block to ingress
5. **Commit, push** — ArgoCD syncs, controller re-evaluates

### Silver → Gold

**Use shared component for DRY:**

```yaml
# apps/<app>/overlays/prod/kustomization.yaml
components:
  - ../../../../_shared/components/gold-maturity  # Adds all Gold requirements
```

This adds:
- `goldilocks.fairwinds.com/enabled: "true"` annotation
- `revisionHistoryLimit: 3`
- Sync-wave annotation

**If app exposes metrics:**
- Add ServiceMonitor in Git
- Add prometheus annotations

### Gold → Platinum

**Use shared components:**

```yaml
components:
  - ../../../../_shared/components/gold-maturity
  - ../../../../_shared/components/priority/high           # ← PriorityClass
  - ../../../../_shared/components/poddisruptionbudget/1   # ← PDB
```

**Sizing is auto-managed:**
- Kyverno `mutate-sizing-labels` adds `vixens.io/sizing.<container>` based on your resources
- You just need to add `vixens.io/sizing-rationale` annotation explaining why

```yaml
# apps/<app>/base/deployment.yaml
metadata:
  annotations:
    vixens.io/sizing-rationale: "Memory based on Goldilocks P95 recommendation"
```

### Platinum → Emerald

1. **Litestream** — Add sidecar for SQLite backup (if applicable)
2. **Config-Syncer** — Add sidecar for config backup
3. **Velero label** — Add `vixens.io/backup-profile: standard`

### Emerald → Diamond

1. **NetworkPolicy** — Add `networkpolicy.yaml` in base/
2. **SecurityContext** — Harden (runAsNonRoot, drop capabilities)
3. **SSO** — Integrate with Authentik if applicable

### Diamond → Orichalcum

1. **7 days stability** — No restarts (automatic monitoring)
2. **0 CVE** — Trivy scan clean
3. **Validated sizing** — Goldilocks recommendations applied

---

## Anti-Patterns to Avoid

| ❌ Wrong | ✅ Right |
|----------|----------|
| Copy-paste PriorityClass YAML | Use `priority/high` component |
| Copy-paste PDB YAML | Use `poddisruptionbudget/1` component |
| Manually add sizing labels | Let Kyverno mutate them |
| Manually set maturity label | Let controller evaluate |
| `kubectl apply` to add resources | Edit Git, commit, push |
| Hardcode secrets in deployment | Use InfisicalSecret |

---

## Troubleshooting Maturity

### App Stuck at Bronze

```bash
# Check what's blocking
kubectl get policyreport -n $NS -o json | jq '[.items[].results[] | select(.result == "fail") | .policy] | unique'
```

Common blockers:
- Missing resources on sidecar (config-syncer, litestream)
- Missing probes on sidecar
- Hardcoded Secret instead of InfisicalSecret
- Ingress missing TLS block

### Controller Not Running

```bash
# Check CronJob
kubectl -n kyverno get cronjob maturity-controller

# Check recent jobs
kubectl -n kyverno get jobs -l app=maturity-controller --sort-by='.metadata.creationTimestamp' | tail -3

# Manual trigger
kubectl -n kyverno create job maturity-manual --from=cronjob/maturity-controller
```

### Sizing Labels Wrong

Sizing labels are computed by Kyverno based on `resources.requests`:
- If resources change, labels update on next pod creation
- Trigger update: `kubectl rollout restart deployment/$DEPLOY`
