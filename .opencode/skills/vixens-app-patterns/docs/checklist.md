# Deployment Checklist (Definition of Done)

## Pre-Deployment Validation

### 1. ✅ Manifests Validation

```bash
# YAML syntax
yamllint apps/<category>/<app>/**/*.yaml

# Kustomize build (dev)
kustomize build apps/<category>/<app>/overlays/dev

# Kustomize build (prod)
kustomize build apps/<category>/<app>/overlays/prod

# Kinds regression check (after kustomization.yaml changes)
kustomize build apps/<category>/<app>/overlays/dev | grep '^kind:' | sort
# Compare with previous build — missing kind = resource dropped
```

---

### 2. ✅ Sizing Labels (MANDATORY)

**Deployment metadata**:
```yaml
metadata:
  labels:
    vixens.io/sizing-v2: "true"        # ✅ Required
    vixens.io/vpa-mode: Auto           # ✅ Required
```

**Pod template labels**:
```yaml
spec:
  template:
    metadata:
      labels:
        vixens.io/sizing.<app>: V-medium         # ✅ Main container
        vixens.io/sizing.litestream: V-nano      # ⚠️ If litestream sidecar
        vixens.io/sizing.config-syncer: V-nano   # ⚠️ If config-syncer
        vixens.io/sizing.restore-config: B-nano  # ⚠️ If restore-config init
        vixens.io/sizing.restore-db: B-nano      # ⚠️ If restore-db init
```

**❌ NEVER**:
```yaml
# NEVER add explicit resources
containers:
  - name: app
    resources:  # ❌ FORBIDDEN
      requests: {cpu: 100m, memory: 256Mi}
```

---

### 3. ✅ Priority & Tolerations (MANDATORY)

```yaml
spec:
  template:
    spec:
      priorityClassName: vixens-medium  # ✅ Required: critical|high|medium|low
      tolerations:                      # ✅ Required
        - key: node-role.kubernetes.io/control-plane
          operator: Exists
          effect: NoSchedule
```

---

### 4. ✅ Revision History (MANDATORY)

```yaml
spec:
  revisionHistoryLimit: 3  # ✅ Required (reduces etcd growth)
```

**Helm apps**: Use component `revision-history-limit` in overlays (can't patch directly).

---

### 5. ✅ Probes (MANDATORY)

```yaml
containers:
  - name: app
    livenessProbe:     # ✅ Required
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 30
      periodSeconds: 30
    readinessProbe:    # ✅ Required
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 10
      periodSeconds: 10
    startupProbe:      # ⚠️ Recommended (slow-start apps)
      httpGet:
        path: /health
        port: 8080
      failureThreshold: 30
      periodSeconds: 10
```

---

### 6. ✅ Overlays (MANDATORY)

**Dev overlay**:
```yaml
# ✅ Must disable replicas
patches:
  - patch: "spec:\n  replicas: 0"
    target: {kind: Deployment}
resources:
  - ../../base
  - ingress.yaml  # dev.truxonline.com
components:
  - ../../../../_shared/components/revision-history-limit
```

**Prod overlay**:
```yaml
# ✅ Must have gold-maturity
resources:
  - ../../base
  - ingress.yaml  # truxonline.com
components:
  - ../../../../_shared/components/gold-maturity  # ✅ Always
  - ../../../../_shared/components/base
  - ../../../../_shared/components/revision-history-limit
```

---

### 7. ✅ Ingress (MANDATORY for web apps)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod  # ✅ Required
    traefik.ingress.kubernetes.io/router.middlewares: traefik-https-redirect@kubernetescrd  # ✅ Required (HTTPS)
spec:
  ingressClassName: traefik
  tls:  # ✅ Required
    - hosts: [my-app.truxonline.com]
      secretName: my-app-tls
  rules:
    - host: my-app.truxonline.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service: {name: my-app, port: {number: 80}}
```

---

### 8. ✅ Security Context (MANDATORY)

```yaml
spec:
  template:
    spec:
      securityContext:  # ✅ Pod-level
        fsGroup: 1000
        fsGroupChangePolicy: OnRootMismatch
        runAsNonRoot: true  # ⚠️ Recommended
        seccompProfile:
          type: RuntimeDefault
      containers:
        - name: app
          securityContext:  # ✅ Container-level
            runAsUser: 1000
            runAsGroup: 1000
            allowPrivilegeEscalation: false
            capabilities:
              drop: ["ALL"]
```

---

## Stateful Apps Only

### 9. ✅ Persistence (MANDATORY for stateful)

**PVC**:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-app-data
spec:
  accessModes: [ReadWriteOnce]  # ✅ RWO for single-pod
  storageClassName: synelia-iscsi-retain  # ✅ Retain policy
  resources:
    requests:
      storage: 10Gi
```

---

### 10. ✅ Backup Strategy (MANDATORY for stateful)

**SQLite apps**:
- ✅ Litestream sidecar + ConfigMap
- ✅ Config-syncer sidecar (rclone)
- ✅ Init containers: restore-db + restore-config

**PostgreSQL apps**:
- ✅ Config-syncer sidecar (user uploads/files)
- ✅ Document PostgreSQL backup (external)

**Config-only apps**:
- ✅ Config-syncer sidecar
- ✅ Init container: restore-config

---

### 11. ✅ Restore Pattern (MANDATORY for stateful)

```yaml
initContainers:
  # Restore config from S3
  - name: restore-config
    image: rclone/rclone:1.73
    command: ["sh", "-c"]
    args:
      - |
        export RCLONE_CONFIG_S3_TYPE=s3
        export RCLONE_CONFIG_S3_PROVIDER=Other
        export RCLONE_CONFIG_S3_ACCESS_KEY_ID=$LITESTREAM_ACCESS_KEY_ID
        export RCLONE_CONFIG_S3_SECRET_ACCESS_KEY=$LITESTREAM_SECRET_ACCESS_KEY
        export RCLONE_CONFIG_S3_ENDPOINT=$LITESTREAM_ENDPOINT
        rclone copy s3:$LITESTREAM_BUCKET/config /data --transfers 4 || true
    envFrom:
      - secretRef: {name: my-app-secrets}
    volumeMounts:
      - name: data
        mountPath: /data

  # Restore SQLite DB (if applicable)
  - name: restore-db
    image: litestream/litestream:0.5.9
    args:
      - restore
      - -config
      - /etc/litestream.yml
      - -if-db-not-exists
      - -if-replica-exists
      - /data/db.sqlite3
    envFrom:
      - secretRef: {name: my-app-secrets}
    volumeMounts:
      - name: data
        mountPath: /data
      - name: litestream-config
        mountPath: /etc/litestream.yml
        subPath: litestream.yml
```

---

### 12. ✅ Secrets (MANDATORY for stateful)

**Infisical pattern** (preferred):
```yaml
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: my-app-secrets-sync
spec:
  hostAPI: http://192.168.111.69:8085
  resyncInterval: 60
  authentication:
    universalAuth:
      credentialsRef:
        secretName: infisical-universal-auth
        secretNamespace: argocd
      secretsScope:
        projectSlug: vixens
        envSlug: dev  # ✅ Override in prod
        secretsPath: /apps/<category>/<app>
  managedSecretReference:
    secretName: my-app-secrets
    creationPolicy: Owner
```

**Prod patch**:
```yaml
- op: replace
  path: /spec/authentication/universalAuth/secretsScope/envSlug
  value: prod
```

---

## Complex Apps Only

### 13. ✅ NetworkPolicy (RECOMMENDED for complex)

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: my-app
spec:
  podSelector:
    matchLabels:
      app: my-app
  policyTypes: [Ingress, Egress]
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: traefik
      ports:
        - protocol: TCP
          port: 80
  egress:
    - {}  # Allow all egress
```

---

### 14. ✅ ServiceMonitor (RECOMMENDED for complex)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: my-app
spec:
  selector:
    matchLabels:
      app: my-app
  endpoints:
    - port: metrics
      path: /metrics
      interval: 30s
```

---

### 15. ✅ VPA (OPTIONAL for complex)

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: my-app
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-app
  updatePolicy:
    updateMode: "Off"  # Manual review only
```

---

## Post-Deployment Validation

### 16. ✅ ArgoCD Sync

```bash
# Check ArgoCD application status
kubectl get application -n argocd my-app

# Manual sync (if auto-sync disabled)
argocd app sync my-app

# Check sync status
argocd app get my-app
```

---

### 17. ✅ Pod Health

```bash
# Check pod status
kubectl get pods -n <namespace> -l app=my-app

# Check pod events
kubectl describe pod -n <namespace> <pod-name>

# Check logs
kubectl logs -n <namespace> <pod-name> -c <container-name>
```

---

### 18. ✅ Ingress Access

```bash
# HTTP → HTTPS redirect
curl -I http://my-app.truxonline.com
# Expected: HTTP 301/302/308

# HTTPS access
curl -k https://my-app.truxonline.com
# Expected: HTTP 200

# TLS certificate
echo | openssl s_client -connect my-app.truxonline.com:443 -servername my-app.truxonline.com 2>/dev/null | openssl x509 -noout -dates
```

---

### 19. ✅ Backup Validation (Stateful Only)

```bash
# Check S3 bucket for backups
mc ls prod/<bucket>/<app>/

# Check litestream logs (if SQLite)
kubectl logs -n <namespace> <pod-name> -c litestream

# Check config-syncer logs
kubectl logs -n <namespace> <pod-name> -c config-syncer
```

---

### 20. ✅ Documentation

- [ ] Update `docs/applications/<category>/<app>.md`
- [ ] Update `docs/STATUS.md` (mark ✅/⚠️/❌)
- [ ] Mark deployment table checkboxes:
  - `[x] Déployé` (pod running)
  - `[x] Configuré` (ingress, secrets, etc.)
  - `[x] Testé` (validation passed)

---

## Final Checklist Summary

### Mandatory (ALL apps)

- [ ] 1. Sizing labels (vixens.io/sizing-v2, per-container)
- [ ] 2. Priority class + CP toleration
- [ ] 3. Revision history limit: 3
- [ ] 4. Probes (liveness, readiness)
- [ ] 5. Overlays (dev: replicas 0, prod: gold-maturity)
- [ ] 6. Ingress (HTTPS, cert-manager, https-redirect)
- [ ] 7. Security context (fsGroup, runAsNonRoot, drop ALL)
- [ ] 8. yamllint + kustomize build pass

### Stateful apps MUST ALSO have

- [ ] 9. PVC (RWO, synelia-iscsi-retain)
- [ ] 10. Backup strategy (litestream OR external documented)
- [ ] 11. Restore init containers
- [ ] 12. Secrets (Infisical OR k8s Secret)

### Complex apps MAY have

- [ ] 13. NetworkPolicy (if sensitive data)
- [ ] 14. ServiceMonitor (if metrics exposed)
- [ ] 15. VPA (if resource tuning needed)

### Post-deployment

- [ ] 16. ArgoCD sync successful
- [ ] 17. Pod running & healthy
- [ ] 18. Ingress accessible (HTTPS)
- [ ] 19. Backup validated (stateful only)
- [ ] 20. Documentation updated

---

**Ready to deploy?** ✅ All checkboxes checked → Deploy to dev → Validate → Deploy to prod
