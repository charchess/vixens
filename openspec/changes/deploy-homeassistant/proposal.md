# Deploy Home Assistant on Kubernetes

## Why

Home Assistant is a critical application for home automation, currently running outside the Kubernetes infrastructure. Deploying it on Kubernetes provides:

**Benefits:**
- **High Availability**: Pod restarts on failure, node rescheduling
- **Resource Management**: CPU/memory limits, guaranteed QoS
- **Backup Integration**: Velero snapshots, persistent storage via Synology CSI
- **GitOps Workflow**: Declarative configuration, version-controlled deployment
- **Network Integration**: Native Kubernetes networking, Cilium policies
- **Ingress**: TLS termination via Traefik, Let's Encrypt certificates

**Current State:**
- Home Assistant runs on bare metal or VM (outside Kubernetes)
- Manual configuration and updates
- No automated backup strategy
- No ingress with proper TLS

**Target State:**
- Home Assistant deployed via ArgoCD
- Persistent storage via Synology CSI (configuration, database)
- Ingress via Traefik with TLS (homeassistant.{env}.truxonline.com)
- Resource limits and health checks
- Secrets managed via Infisical (if needed)

## What Changes

### 1. Application Structure

Create Kustomize-based Home Assistant structure:

```
apps/applications/homeassistant/
├── base/
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── deployment.yaml          # StatefulSet or Deployment
│   ├── service.yaml             # ClusterIP service
│   ├── pvc.yaml                 # Persistent volume claim (Synology CSI)
│   ├── configmap.yaml           # Optional: configuration
│   └── README.md
└── overlays/
    ├── dev/
    │   ├── kustomization.yaml
    │   ├── ingress.yaml         # homeassistant.dev.truxonline.com
    │   └── patches.yaml         # Environment-specific patches
    ├── test/
    ├── staging/
    └── prod/
        ├── kustomization.yaml
        ├── ingress.yaml         # homeassistant.truxonline.com
        └── patches.yaml         # Production resources, HA config
```

### 2. Deployment Configuration

**Container Image:**
- Official Home Assistant image: `ghcr.io/home-assistant/home-assistant:stable`
- Consider version pinning for production

**Storage:**
- PVC via Synology CSI: 10Gi (dev/test), 50Gi (prod)
- Mounted at `/config` for Home Assistant configuration
- StorageClass: `synology-iscsi-storage`

**Resources:**
- Dev/Test: 500m CPU, 1Gi memory
- Prod: 1 CPU, 2Gi memory (adjust based on integrations)

**Health Checks:**
- Liveness probe: HTTP GET http://localhost:8123/ (after startup delay)
- Readiness probe: HTTP GET http://localhost:8123/

**Security Context:**
- May need privileged mode for USB devices (Zigbee/Z-Wave)
- Consider `hostNetwork: true` for mDNS discovery (optional)

### 3. Network Configuration

**Service:**
- Type: ClusterIP
- Port: 8123
- Protocol: TCP

**Ingress (Traefik):**
- Hostname: `homeassistant.{env}.truxonline.com`
- TLS: Let's Encrypt certificate via cert-manager
- Annotations: Traefik middleware for WebSocket support

**Special Considerations:**
- WebSocket support for Home Assistant frontend
- May need sticky sessions (session affinity)
- Consider external access for mobile app (cloud integration)

### 4. ArgoCD Integration

Create ArgoCD Applications for each environment:

**Example (dev):**
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: homeassistant
  namespace: argocd
  annotations:
    argocd.argoproj.io/sync-wave: "4"  # Applications wave
spec:
  project: default
  destination:
    server: https://kubernetes.default.svc
    namespace: homeassistant
  source:
    repoURL: https://github.com/charchess/vixens.git
    targetRevision: dev
    path: apps/applications/homeassistant/overlays/dev
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### 5. Migration Strategy

**If migrating from existing instance:**
1. Backup current Home Assistant configuration directory
2. Create PVC and copy configuration via temporary pod
3. Test in dev environment first
4. Blue/green deployment: keep old instance until validated
5. Update DNS to point to new ingress
6. Decommission old instance after successful migration

## Impact

**User Experience:**
- ✅ Single URL with TLS for Home Assistant access
- ✅ Mobile app compatibility (cloud integration may be needed)
- ✅ Faster startup via Kubernetes node resources

**Operations:**
- ✅ Automated deployment via GitOps
- ✅ Easy rollback with ArgoCD
- ✅ Centralized logging and monitoring
- ✅ Backup integration with Velero

**Reliability:**
- ✅ Pod restarts on crash
- ✅ Persistent storage survives pod restarts
- ✅ Resource guarantees (no resource starvation)

**Risk:**
- ⚠️ Migration complexity if moving from existing instance
- ⚠️ USB device passthrough may require hostPath or privileged mode
- ⚠️ Network discovery (mDNS) may not work without hostNetwork
- ⚠️ Initial configuration may need manual intervention
- Mitigation: Test thoroughly in dev, document device requirements

## Non-Goals

- Not migrating existing Home Assistant configuration (manual migration if needed)
- Not setting up Zigbee/Z-Wave USB devices (document separately if needed)
- Not configuring Home Assistant integrations (user responsibility)
- Not implementing Home Assistant backup automation (use Velero)
- Not exposing Home Assistant on public internet (use VPN or cloud integration)
