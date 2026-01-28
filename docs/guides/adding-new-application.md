# Adding a New Application

This guide walks you through deploying a new application to the Vixens Kubernetes cluster using GitOps best practices.

---

## Prerequisites

Before starting, decide on:
- **Application name:** Lowercase, kebab-case (e.g., `jellyfin`, `home-assistant`)
- **Namespace:** Which namespace? (See [Namespace Strategy](#namespace-strategy))
- **Category:** Which category? (See [Categories](#application-categories))
- **Environment:** Deploy to dev first, promote to prod later

---

## Application Categories

Applications are organized by category (matching directory structure):

| Category | Prefix | Examples | Purpose |
|----------|--------|----------|---------|
| **Infrastructure** | `00-infra` | ArgoCD, Traefik, Cilium | Core infrastructure |
| **Monitoring** | `02-monitoring` | Prometheus, Grafana, Loki | Observability stack |
| **Databases** | `10-databases` | PostgreSQL, Redis | Shared databases |
| **Media** | `20-media` | Jellyfin, Radarr, Sonarr | Media management |
| **Network** | `40-network` | AdGuard, External-DNS | Network services |
| **Services** | `50-services` | Home Assistant, Vaultwarden | General services |
| **Tools** | `70-tools` | Homepage, Linkwarden | Utilities |

---

## Namespace Strategy

| Strategy | When to Use | Example |
|----------|-------------|---------|
| **Shared namespace** | Apps in same category with similar security requirements | `media` namespace for all *arr apps |
| **Dedicated namespace** | Apps with specific security/isolation needs | `homeassistant` for Home Assistant |
| **Existing namespace** | App fits into established namespace | Add new media app to `media` |

**Shared Namespaces (Current):**
- `media` - Media applications (*arr stack, Jellyfin, etc.)
- `tools` - Utility applications
- `services` - General services
- `monitoring` - Monitoring stack

---

## Step-by-Step Process

### 1. Create Base Directory Structure

```bash
cd /root/vixens/apps/<category>/<app-name>
mkdir -p base overlays/{dev,prod}
```

Example for `jellyseerr`:
```bash
cd /root/vixens/apps/20-media/jellyseerr
mkdir -p base overlays/{dev,prod}
```

### 2. Create Base Resources

Create `base/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

# If using SHARED namespace, reference it
# If DEDICATED namespace, include namespace.yaml below

resources:
  # Uncomment if dedicated namespace:
  # - namespace.yaml
  - deployment.yaml
  - service.yaml
  # Add other resources as needed:
  # - ingress.yaml
  # - configmap.yaml
  # - pvc.yaml

# Optional: Add common labels
commonLabels:
  app.kubernetes.io/name: <app-name>
  app.kubernetes.io/instance: <app-name>
```

**If using DEDICATED namespace**, create `base/namespace.yaml`:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: <app-name>
  labels:
    name: <app-name>
    # Optional: Pod Security Standard
    pod-security.kubernetes.io/enforce: baseline
```

**If using SHARED namespace** (e.g., `media`):
- DO NOT create namespace.yaml
- Set `namespace: media` in all resources

### 3. Create Base Deployment

Create `base/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: <app-name>
  namespace: <namespace>  # media, tools, or app-specific
  labels:
    app.kubernetes.io/name: <app-name>
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: <app-name>
  template:
    metadata:
      labels:
        app.kubernetes.io/name: <app-name>
    spec:
      containers:
      - name: <app-name>
        image: <registry>/<image>:<tag>
        ports:
        - containerPort: <port>
          name: http
        # Environment variables (non-secret)
        env:
        - name: TZ
          value: "America/Toronto"
        # Optional: Resource limits
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        # Optional: Volume mounts
        # volumeMounts:
        # - name: config
        #   mountPath: /config
        # - name: data
        #   mountPath: /data
      # Optional: Volumes
      # volumes:
      # - name: config
      #   persistentVolumeClaim:
      #     claimName: <app-name>-config
```

### 4. Create Base Service

Create `base/service.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: <app-name>
  namespace: <namespace>
spec:
  selector:
    app.kubernetes.io/name: <app-name>
  ports:
  - name: http
    port: 80
    targetPort: http
    protocol: TCP
  type: ClusterIP
```

### 5. Configure Dev Overlay

Create `overlays/dev/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: <namespace>

bases:
  - ../../base

# Add environment-specific label
commonLabels:
  environment: dev

# Optional: Reference shared resources
resources:
  # If using HTTP redirect middleware:
  # - ../../_shared/middlewares/base

# Optional: Patches
# patchesStrategicMerge:
#   - deployment-patch.yaml
#   - ingress-patch.yaml
```

**If the app needs Ingress**, create `overlays/dev/ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: <app-name>
  namespace: <namespace>
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    traefik.ingress.kubernetes.io/router.entrypoints: websecure
    traefik.ingress.kubernetes.io/router.middlewares: <namespace>-<app-name>-http-redirect@kubernetescrd
    # Optional: Homepage integration
    gethomepage.dev/enabled: "true"
    gethomepage.dev/name: "<App Display Name>"
    gethomepage.dev/description: "<Short description>"
    gethomepage.dev/group: "<Category>"
    gethomepage.dev/icon: "<icon-name>.png"
spec:
  ingressClassName: traefik
  tls:
  - hosts:
    - <app-name>.dev.truxonline.com
    secretName: <app-name>-tls-dev
  rules:
  - host: <app-name>.dev.truxonline.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: <app-name>
            port:
              number: 80
```

**IMPORTANT:** If using HTTP redirect middleware, add to `kustomization.yaml`:

```yaml
resources:
  - ingress.yaml
  - http-redirect.yaml
```

And create `overlays/dev/http-redirect.yaml`:

```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: <app-name>-http-redirect
  namespace: <namespace>
spec:
  redirectScheme:
    scheme: https
    permanent: true
```

### 6. Configure Prod Overlay

Create `overlays/prod/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: <namespace>

bases:
  - ../../base

commonLabels:
  environment: prod

# Same structure as dev, different domain
resources:
  - ingress.yaml
  - http-redirect.yaml  # If needed

# Optional: Different resource limits
patchesStrategicMerge:
  - deployment-patch.yaml
```

Prod ingress uses `.truxonline.com` (no `.dev`):
```yaml
spec:
  tls:
  - hosts:
    - <app-name>.truxonline.com
    secretName: <app-name>-tls-prod
  rules:
  - host: <app-name>.truxonline.com
    ...
```

### 7. Manage Secrets via Infisical

**DO NOT commit secrets to Git!**

#### 7.1 Add Secrets to Infisical

1. Access Infisical: http://192.168.111.69:8085
2. Project: `vixens`
3. Environment: `dev` or `prod`
4. Path: `/apps/<category>/<app-name>`
5. Add secrets:
   - Database credentials
   - API keys
   - Passwords
   - Tokens

Example path: `/apps/20-media/jellyseerr`

#### 7.2 Create InfisicalSecret Resource

If app needs secrets, create `base/infisical-secret.yaml`:

```yaml
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: <app-name>-secrets
  namespace: <namespace>
spec:
  hostAPI: http://infisical.infisical.svc.cluster.local:80
  resyncInterval: 60
  authentication:
    universalAuth:
      secretsScope:
        projectSlug: vixens
        envSlug: dev  # Will be patched to 'prod' in prod overlay
        secretsPath: /apps/<category>/<app-name>
      credentialsRef:
        secretName: infisical-universal-auth
        secretNamespace: argocd
  managedSecretReference:
    secretName: <app-name>-secrets
    secretType: Opaque
    creationPolicy: "Owner"
```

Add to `base/kustomization.yaml`:
```yaml
resources:
  - infisical-secret.yaml
```

#### 7.3 Patch InfisicalSecret for Prod

Create `overlays/prod/infisical-secret-patch.yaml`:

```yaml
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: <app-name>-secrets
  namespace: <namespace>
spec:
  authentication:
    universalAuth:
      secretsScope:
        envSlug: prod  # Override to prod
```

Add to `overlays/prod/kustomization.yaml`:
```yaml
patchesStrategicMerge:
  - infisical-secret-patch.yaml
```

#### 7.4 Use Secrets in Deployment

Update `base/deployment.yaml`:

```yaml
spec:
  template:
    spec:
      containers:
      - name: <app-name>
        env:
        - name: DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: <app-name>-secrets
              key: DATABASE_PASSWORD
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: <app-name>-secrets
              key: API_KEY
```

### 8. Create ArgoCD Application

Create `argocd/overlays/dev/apps/<app-name>.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: <app-name>
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: https://github.com/<your-org>/vixens.git
    targetRevision: main
    path: apps/<category>/<app-name>/overlays/dev
  destination:
    server: https://kubernetes.default.svc
    namespace: <namespace>
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
```

Add to `argocd/overlays/dev/kustomization.yaml`:

```yaml
resources:
  # ... existing apps ...
  - apps/<app-name>.yaml
```

### 9. Validation Checklist

Before pushing to Git:

- [ ] Base resources created (deployment, service)
- [ ] Overlays configured (dev, prod)
- [ ] Secrets added to Infisical (if needed)
- [ ] InfisicalSecret resource created (if needed)
- [ ] Ingress configured with correct domains
- [ ] HTTP redirect middleware added (if exposing publicly)
- [ ] ArgoCD application manifest created
- [ ] Kustomize builds without errors: `kustomize build apps/<category>/<app-name>/overlays/dev`
- [ ] No secrets committed to Git: `git diff | grep -i password`

### 10. Deploy to Dev

```bash
# Stage changes
git add apps/<category>/<app-name>/
git add argocd/overlays/dev/apps/<app-name>.yaml
git add argocd/overlays/dev/kustomization.yaml

# Commit with conventional commit format
git commit -m "feat: deploy <app-name> in <namespace> namespace

- Create Kustomize base and overlays
- Configure Ingress for dev
- Integrate with Infisical for secrets
- Add ArgoCD application manifest

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"

# Push feature branch and create PR
git push origin feature/add-<app-name>
gh pr create --base main --head feature/add-<app-name>
```

### 11. Verify Deployment

```bash
# Set dev kubeconfig
export KUBECONFIG=.secrets/dev/kubeconfig-dev

# Check ArgoCD app
kubectl -n argocd get application <app-name>

# Check pod status
kubectl -n <namespace> get pods -l app.kubernetes.io/name=<app-name>

# Check logs
kubectl -n <namespace> logs -l app.kubernetes.io/name=<app-name> --tail=50

# Check ingress
kubectl -n <namespace> get ingress <app-name>

# Test URL
curl -I https://<app-name>.dev.truxonline.com
```

### 12. Promote to Prod

Once validated in dev, create prod ArgoCD application:

1. Create `argocd/overlays/prod/apps/<app-name>.yaml` (same as dev but `targetRevision: prod-stable`)
2. Add to `argocd/overlays/prod/kustomization.yaml`
3. Commit and push to `dev` branch
4. Follow [GitOps Workflow](gitops-workflow.md) to promote to `main`

---

## Common Patterns

### Pattern 1: Simple Web App (No Secrets)
- Base: deployment + service
- Overlays: ingress + http-redirect
- No InfisicalSecret needed

### Pattern 2: Web App with Secrets
- Base: deployment + service + infisical-secret
- Overlays: ingress + http-redirect + infisical-secret-patch
- Secrets in Infisical

### Pattern 3: *arr App (Media Stack)
- Shared `media` namespace
- Init container for config patching
- Shared config-patcher script
- PostgreSQL database connection

### Pattern 4: Monitoring Tool
- Dedicated namespace (optional)
- ServiceMonitor for Prometheus
- Grafana dashboard ConfigMap
- No public ingress (cluster-only)

### Pattern 5: SQLite Application with Sidecar Backup
- Uses **Litestream** for continuous SQLite replication to S3
- **Init Container**: Restores DB from S3 on startup
- **Sidecar Container**: Replicates WAL to S3 in real-time
- Requires S3 credentials in Infisical

---

## SQLite Backup Strategy (Litestream)

For applications relying on SQLite (e.g., *arr stack, Hydrus, Vaultwarden), use **Litestream** to ensure data durability without relying on Volume snapshots.

### 1. Architecture
- **Init Container (`restore-db`)**: Checks if DB exists. If not, restores from S3.
- **Sidecar Container (`litestream`)**: Runs alongside the app, watching the DB file and pushing changes to S3.
- **Shared Volume**: The DB directory must be shared between init, app, and sidecar containers.

### 2. Infisical Setup
Add S3 credentials to Infisical at `/infra/backup-s3` (or app-specific path):
- `LITESTREAM_ACCESS_KEY_ID`
- `LITESTREAM_SECRET_ACCESS_KEY`
- `LITESTREAM_BUCKET`
- `LITESTREAM_ENDPOINT` (e.g., `s3.fr-par.scw.cloud`)

### 3. Safety Exclusions for Rclone

When using **Rclone** for configuration backup alongside **Litestream** for SQLite replication, you **MUST** exclude Litestream's working directories to prevent storage saturation and circular backups.

**Rule:** Always add `--exclude ".*litestream/**"` to your `rclone copy` and `rclone sync` commands.

### 4. Implementation Example

**In `base/deployment.yaml`:**

```yaml
      initContainers:
        - name: restore-db
          image: litestream/litestream:latest
          args: ["restore", "-if-db-not-exists", "-if-replica-exists", "/data/my-app.db"]
          volumeMounts:
            - name: data
              mountPath: /data
          envFrom:
            - secretRef:
                name: <app-name>-litestream-secret

      containers:
        - name: <app-name>
          # ... app config ...
          volumeMounts:
            - name: data
              mountPath: /data

        - name: litestream
          image: litestream/litestream:latest
          args: ["replicate", "/data/my-app.db", "s3://$(LITESTREAM_BUCKET)/<app-name>/my-app.db"]
          volumeMounts:
            - name: data
              mountPath: /data
          envFrom:
            - secretRef:
                name: <app-name>-litestream-secret
          livenessProbe:
            exec:
              command: ["/usr/local/bin/litestream", "version"]
```

**In `base/infisical-secret.yaml`:**
Ensure you map the S3 credentials to the secret used by Litestream.

---

## Troubleshooting

### ArgoCD app stuck in "Progressing"
```bash
kubectl -n argocd get application <app-name> -o yaml
# Check .status.conditions
```

### Pod CrashLoopBackOff
```bash
kubectl -n <namespace> logs <pod-name>
kubectl -n <namespace> describe pod <pod-name>
```

### Ingress not working
```bash
# Check ingress
kubectl -n <namespace> get ingress <app-name> -o yaml

# Check Traefik logs
kubectl -n traefik logs -l app.kubernetes.io/name=traefik

# Check cert-manager
kubectl -n cert-manager logs -l app=cert-manager
kubectl get certificate -A
```

### Secrets not syncing
```bash
# Check InfisicalSecret status
kubectl -n <namespace> get infisicalsecret <app-name>-secrets -o yaml

# Check Infisical operator logs
kubectl -n infisical logs -l app.kubernetes.io/name=secrets-operator
```

---

## Additional Resources

- [GitOps Workflow Guide](gitops-workflow.md) - Promoting to production
- [Secret Management Guide](secret-management.md) - Infisical details
- [Reference: Kustomize Patterns](../reference/kustomize-patterns.md) - Advanced patterns
- [Reference: Naming Conventions](../reference/naming-conventions.md) - Naming rules
- [ADR-011: Namespace Ownership](../adr/011-namespace-ownership-strategy.md) - Namespace strategy

---

**Last Updated:** 2025-12-30
