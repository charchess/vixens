# Decision Matrix: Choosing the Right Template

## Quick Decision Tree

```
┌─ Does app persist data?
│  ├─ NO → Is it Helm-based?
│  │  ├─ YES → templates/stateless-helm/
│  │  └─ NO  → templates/stateless-native/
│  └─ YES → What storage?
│     ├─ SQLite → templates/stateful/
│     ├─ PostgreSQL → templates/stateful/ (no litestream)
│     └─ Config files only → templates/stateful/ (no litestream)
│
└─ Multiple deployments?
   └─ YES → templates/complex/
```

---

## Detailed Matrix

| Scenario | Template | Components | Patterns |
|----------|----------|-----------|----------|
| **Simple web app (whoami)** | stateless-native | base, revision-history-limit | - |
| **Tool from Helm chart (it-tools)** | stateless-helm | revision-history-limit | values.yaml |
| **Password manager (vaultwarden)** | stateful | gold-maturity, base | litestream + config-syncer |
| **Note-taking (trilium)** | stateful | gold-maturity, base | litestream + config-syncer |
| **Finance tracker (firefly-iii)** | stateful | gold-maturity, base | config-syncer only |
| **SSO provider (authentik)** | complex | gold-maturity, base | Multi-deployment + NetworkPolicy + ServiceMonitor |
| **Home automation (homeassistant)** | complex | gold-maturity, base | 5 containers + hostNetwork |

---

## By Characteristic

### Helm vs Native K8s

| Question | Helm Template | Native Template |
|----------|--------------|-----------------|
| Manifests managed by | Helm chart | Kustomize |
| Base structure | `values.yaml` | `deployment.yaml` |
| Kustomize role | Patches only | Full resource definition |
| Limitations | Can't patch pod template | None |
| Examples | it-tools, stirling-pdf | whoami, vaultwarden |

---

### Persistence Type

| Storage | Litestream | Config-Syncer | Init Containers | Example |
|---------|-----------|---------------|-----------------|---------|
| **None** | ❌ | ❌ | ❌ | whoami, it-tools |
| **SQLite** | ✅ | ✅ | restore-db + restore-config | vaultwarden, trilium |
| **PostgreSQL** | ❌ | ✅ | restore-config | firefly-iii |
| **Config files** | ❌ | ✅ | restore-config | authentik |

---

### Complexity Level

| Level | Deployments | NetworkPolicy | ServiceMonitor | VPA | Template |
|-------|------------|---------------|----------------|-----|----------|
| **Simple** | 1 | ❌ | ❌ | ❌ | stateless-* |
| **Standard** | 1 | ⚠️ | ⚠️ | ❌ | stateful |
| **Complex** | 2+ | ✅ | ✅ | ✅ | complex |

---

## Examples by Category

### 00-infra (Infrastructure)

- traefik → Native K8s, Helm values
- argocd → Helm chart + kustomize patches
- cilium-lb → Custom CRD (LoadBalancerIPPool)

### 10-home (Home Automation)

- homeassistant → complex (5 containers, hostNetwork, UDP service)
- mealie → stateful (PostgreSQL + config files)
- mosquitto → stateless-native

### 20-media (Media)

- jellyfin → stateful (PostgreSQL + transcoding cache)
- radarr/sonarr → stateful (SQLite + config files)
- qbittorrent → stateless-native (downloads on NFS)

### 60-services (Business Services)

- vaultwarden → stateful (SQLite + attachments)
- firefly-iii → stateful (PostgreSQL + uploads)
- openclaw → stateful (SQLite + config)

### 70-tools (Developer Tools)

- it-tools → stateless-helm
- trilium → stateful (SQLite + notes)
- stirling-pdf → stateless-helm

---

## Component Selection Guide

### Dev Overlay (Always)

```yaml
components:
  - ../../../../_shared/components/revision-history-limit  # Always
patches:
  - patch: "spec:\n  replicas: 0"  # Disable in dev
    target: {kind: Deployment}
```

### Prod Overlay (Standard)

```yaml
components:
  - ../../../../_shared/components/gold-maturity          # Always
  - ../../../../_shared/components/base                   # Common config
  - ../../../../_shared/components/revision-history-limit # Always
```

### Prod Overlay (High Availability)

```yaml
components:
  - ../../../../_shared/components/gold-maturity
  - ../../../../_shared/components/base
  - ../../../../_shared/components/resources              # Explicit resources
  - ../../../../_shared/components/poddisruptionbudget/0  # PDB
  - ../../../../_shared/components/priority/high          # High priority
  - ../../../../_shared/components/revision-history-limit
```

---

## Pattern Selection

### When to Add NetworkPolicy

✅ Add if:
- App handles sensitive data (auth, passwords, PII)
- Multi-tenant workload
- Ingress needs restriction (not just Traefik)

❌ Skip if:
- Simple tool (it-tools, whoami)
- Already isolated by namespace RBAC

### When to Add ServiceMonitor

✅ Add if:
- App exposes metrics endpoint
- Performance monitoring needed
- SLO/SLA tracking required

❌ Skip if:
- No metrics endpoint
- Low-criticality app

### When to Add VPA

✅ Add if:
- Resource usage unpredictable
- Need tuning recommendations
- High-traffic app

❌ Skip if:
- Resource usage stable
- Low-traffic app
- Already using HPA

---

## Anti-Patterns (DON'T)

❌ **Explicit resources in YAML**
```yaml
# WRONG
resources:
  requests: {cpu: 100m, memory: 256Mi}
```
✅ Use sizing labels instead

---

❌ **No probes**
```yaml
# WRONG
containers:
  - name: app
    image: my-app
    # Missing livenessProbe, readinessProbe
```
✅ Always add probes

---

❌ **Hardcoded secrets**
```yaml
# WRONG
env:
  - name: PASSWORD
    value: "mysecretpassword"
```
✅ Use Infisical or k8s Secret

---

❌ **No backup for stateful apps**
```yaml
# WRONG: PVC but no litestream/config-syncer
```
✅ Add backup strategy (litestream/external)

---

## Validation Checklist

Before deploying, verify:

- [ ] Template matches app type (stateless/stateful/complex)
- [ ] Sizing labels present (per-container)
- [ ] Probes defined (liveness/readiness)
- [ ] Overlays configured (dev: replicas 0, prod: gold-maturity)
- [ ] Persistence strategy (if stateful: litestream/external backup)
- [ ] Secrets managed (Infisical or k8s Secret with rotation)
- [ ] NetworkPolicy (if sensitive data)
- [ ] ServiceMonitor (if metrics exposed)

---

**See Also**:
- `docs/checklist.md` — Full DoD checklist
- `../SKILL.md` — Main skill documentation
