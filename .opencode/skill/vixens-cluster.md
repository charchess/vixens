---
description: >-
  Vixens Kubernetes cluster operations expert. ALWAYS USE for: kubectl commands,
  ArgoCD sync/refresh, Talos operations, pod debugging, deployment issues, GitOps workflow,
  prod-stable tag, namespace operations, resource usage, events, logs, storage, PVC,
  Synology CSI, cluster health, node management. Trigger on: "check cluster", "kubectl",
  "argocd", "talos", "sync app", "deploy", "pod logs", "why isn't X working".
---

# Vixens Cluster Operations Expert

You are an expert in operating the Vixens Kubernetes clusters.

## Cluster Access

### Production (Default)
```bash
export KUBECONFIG=/home/charchess/vixens/.secrets/prod/kubeconfig-prod
export TALOSCONFIG=/home/charchess/vixens/.secrets/prod/talosconfig-prod
```

### Development
```bash
export KUBECONFIG=/home/charchess/vixens/.secrets/dev/kubeconfig-dev
export TALOSCONFIG=/home/charchess/vixens/.secrets/dev/talosconfig-dev
```

## Cluster Info

| Cluster | Nodes | VIP | Talos | K8s |
|---------|-------|-----|-------|-----|
| **Prod** | peach, pearl, phoebe, poison, powder | 192.168.111.190 | v1.12.4 | v1.34.0 |
| **Dev** | daphne, diva, dulce | 192.168.111.160 | - | - |

## ArgoCD Operations

### Check All Apps
```bash
kubectl -n argocd get applications -o custom-columns='NAME:.metadata.name,SYNC:.status.sync.status,HEALTH:.status.health.status'
```

### Check Apps Not Synced/Healthy
```bash
kubectl -n argocd get applications | grep -v "Synced.*Healthy"
```

### Check Specific App
```bash
kubectl -n argocd get application $APP -o yaml
```

### Check App Revision
```bash
kubectl -n argocd get application $APP -o jsonpath='{.status.sync.revision}'
```

### Force Refresh (fetch latest from Git)
```bash
kubectl -n argocd patch application $APP --type merge -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}'
```

### Force Sync (apply changes)
```bash
kubectl -n argocd patch application $APP --type merge -p '{"operation":{"initiatedBy":{"automated":true},"sync":{"revision":"HEAD"}}}'
```

### Force Sync with Prune
```bash
kubectl -n argocd patch application $APP --type merge -p '{"operation":{"initiatedBy":{"automated":true},"sync":{"revision":"HEAD","prune":true}}}'
```

## GitOps Workflow

### Branch Strategy
- `main` = Dev HEAD (deployed to dev cluster)
- `prod-stable` tag = Production (deployed to prod cluster)
- Apps target `prod-stable` for prod, `main` for dev

### Promote to Prod
```bash
# Update prod-stable tag to main
git tag -f prod-stable main
git push origin refs/tags/prod-stable --force

# Or use workflow (if you have permissions)
gh workflow run promote-prod.yaml -f version=vX.Y.Z
```

### Check prod-stable vs main
```bash
git log --oneline -1 prod-stable
git log --oneline -1 main
git rev-list --left-right --count prod-stable...main
```

## Debugging

### Pod Issues
```bash
# Get pods not ready
kubectl get pods -A -o wide | grep -v Running | grep -v Completed

# Get pod events
kubectl -n $NS describe pod $POD | grep -A20 Events

# Get pod logs
kubectl -n $NS logs $POD -c $CONTAINER --tail=100

# Previous container logs (after crash)
kubectl -n $NS logs $POD -c $CONTAINER --previous
```

### High Restart Pods
```bash
kubectl get pods -A -o json | jq -r '.items[] | select(.status.containerStatuses != null) | select(([.status.containerStatuses[].restartCount] | add) > 5) | "\(.metadata.namespace)/\(.metadata.name): \([.status.containerStatuses[].restartCount] | add) restarts"'
```

### Resource Usage
```bash
kubectl top pods -A --sort-by=memory | head -20
kubectl top nodes
```

### Events (cluster-wide)
```bash
kubectl get events -A --sort-by='.lastTimestamp' | tail -30
```

### Events for specific namespace
```bash
kubectl get events -n $NS --sort-by='.lastTimestamp' | tail -20
```

## Kyverno & Policies

### Check Policy Reports Summary
```bash
kubectl get policyreport -A -o json | jq -r '[.items[].results[]? | select(.result == "fail")] | group_by(.policy) | map({policy: .[0].policy, count: length}) | sort_by(-.count) | .[:10][] | "\(.count) failures: \(.policy)"'
```

### Check Specific App Violations
```bash
kubectl get policyreport -n $NS -o json | jq -r '.items[] | select(.scope.name == "'$APP'") | .results[]? | select(.result == "fail") | .policy'
```

## Storage

### Check PVCs
```bash
kubectl get pvc -A -o custom-columns='NAMESPACE:.metadata.namespace,NAME:.metadata.name,STATUS:.status.phase,SIZE:.spec.resources.requests.storage,STORAGECLASS:.spec.storageClassName'
```

### Synology CSI
- StorageClass: `synelia-iscsi-retain` (prod), `synelia-iscsi-delete` (dev)
- CSI namespace: `synology-csi`

### Check Synology CSI Health
```bash
kubectl -n synology-csi get pods
kubectl -n synology-csi logs -l app=synology-csi-controller --tail=50
```

## Talos Operations

```bash
# Cluster health
talosctl health

# Dashboard (interactive)
talosctl dashboard

# Node services
talosctl services

# Node logs
talosctl logs kubelet -n $NODE

# Reboot node
talosctl reboot -n $NODE

# Upgrade Talos
talosctl upgrade -n $NODE --image ghcr.io/siderolabs/installer:vX.Y.Z

# Get cluster config
talosctl get machineconfig -n $NODE
```

## Common Namespaces

| Namespace | Purpose |
|-----------|---------|
| `argocd` | GitOps controller |
| `kyverno` | Policy engine + maturity controller |
| `monitoring` | Prometheus, Grafana, Loki, Promtail |
| `databases` | PostgreSQL, Redis shared instances |
| `media` | Jellyfin, *arr apps |
| `tools` | Homepage, Netbox, misc tools |
| `networking` | External-DNS, Netbird |
| `security` | Trivy, Authentik |
| `finance` | Firefly-iii |
| `birdnet-go` | BirdNET-Go |
| `traefik` | Ingress controller |
| `cert-manager` | TLS certificates |

## Quick Health Check
```bash
echo "=== Nodes ===" && kubectl get nodes
echo "=== Unhealthy Apps ===" && kubectl -n argocd get applications | grep -v "Synced.*Healthy" | head -10
echo "=== Problem Pods ===" && kubectl get pods -A | grep -v Running | grep -v Completed | head -10
echo "=== Recent Events ===" && kubectl get events -A --field-selector type=Warning --sort-by='.lastTimestamp' | tail -10
```
