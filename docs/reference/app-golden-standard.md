# App Golden Standard — Vixens Reference

**Canonical reference for deploying applications on the Vixens cluster.**

This document defines the mandatory and optional building blocks for an application deployment,
explains the Kyverno-based sizing system, and provides annotated examples.

The `apps/template-app/` directory is the living implementation of this standard.

---

## Table of Contents

1. [Maturity Tiers at a Glance](#maturity-tiers-at-a-glance)
2. [Mandatory Blocks](#mandatory-blocks)
3. [Optional Blocks](#optional-blocks)
4. [Sizing System (Kyverno Labels)](#sizing-system-kyverno-labels)
5. [Priority Classes](#priority-classes)
6. [Annotated Template](#annotated-template)
7. [Common Patterns by Container Type](#common-patterns-by-container-type)
8. [QoS: Burstable vs Guaranteed](#qos-burstable-vs-guaranteed)
9. [Checklist](#checklist)

---

## Maturity Tiers at a Glance

| Block | Bronze | Silver | Gold | Platinum | Elite |
|---|:---:|:---:|:---:|:---:|:---:|
| `priorityClassName` | ✓ | ✓ | ✓ | ✓ | ✓ |
| CP toleration | ✓ | ✓ | ✓ | ✓ | ✓ |
| Kyverno sizing labels | ✓ | ✓ | ✓ | ✓ | ✓ |
| `revisionHistoryLimit: 3` | ✓ | ✓ | ✓ | ✓ | ✓ |
| Infisical secret | — | ✓ | ✓ | ✓ | ✓ |
| Liveness/Readiness probes | — | ✓ | ✓ | ✓ | ✓ |
| Litestream (SQLite backup) | — | — | ✓ | ✓ | ✓ |
| Config-Syncer (flat-file backup) | — | — | ✓ | ✓ | ✓ |
| Metrics annotations | — | — | ✓ | ✓ | ✓ |
| PodDisruptionBudget | — | — | — | ✓ | ✓ |
| NetworkPolicy | — | — | — | ✓ | ✓ |
| Guaranteed QoS (G-sizing) | — | — | — | — | ✓ |

See [quality-standards.md](quality-standards.md) for full scoring criteria.

---

## Mandatory Blocks

Every application **must** have all of the following regardless of tier.

### 1. `priorityClassName`

```yaml
spec:
  template:
    spec:
      priorityClassName: vixens-medium  # see Priority Classes section
```

### 2. Control Plane Toleration

Allows scheduling on CP nodes when workers are under memory pressure.
Applied automatically by the `add-cp-toleration` Kyverno MutatePolicy,
but **must be present in the manifest** to pass the `check-cp-toleration` audit.

```yaml
spec:
  template:
    spec:
      tolerations:
        - key: node-role.kubernetes.io/control-plane
          operator: Exists
          effect: NoSchedule
```

### 3. Kyverno Sizing Labels

Resource requests/limits are **never set in YAML**. They are injected at admission time
by the `sizing-mutate` Kyverno policy based on pod labels.

Labels must appear on **both** `metadata.labels` (Deployment) and
`spec.template.metadata.labels` (Pod). The policy reads Pod labels.

```yaml
metadata:
  labels:
    app: my-app
    vixens.io/sizing: small                # generic fallback (required)
    vixens.io/sizing.my-app: small         # per main container (required)
    vixens.io/sizing.litestream: micro     # if using litestream sidecar
    vixens.io/sizing.config-syncer: micro  # if using config-syncer sidecar
    vixens.io/sizing.restore-config: micro # if using restore-config init
    vixens.io/sizing.restore-db: micro     # if using restore-db init
```

> **Fallback**: if no per-container label is found, the policy falls back to `micro`
> (10m/128Mi). This is safe but **not what you want for the main container** — always
> set `vixens.io/sizing.<container-name>` explicitly.

### 4. `revisionHistoryLimit: 3`

Reduces etcd storage by capping stored ReplicaSet revisions.

```yaml
spec:
  revisionHistoryLimit: 3
```

---

## Optional Blocks

### Litestream (SQLite backup + restore)

Use when the app stores data in a SQLite file.

**Init container** (`restore-db`) — restores from S3 at startup:
```yaml
initContainers:
  - name: restore-db
    image: litestream/litestream:0.5.9
    args: [restore, -config, /etc/litestream.yml, -if-db-not-exists, -if-replica-exists, /config/app.db]
    envFrom:
      - secretRef:
          name: my-app-secrets
    volumeMounts:
      - name: config
        mountPath: /config
      - name: my-app-litestream-config
        mountPath: /etc/litestream.yml
        subPath: litestream.yml
```

**Sidecar** (`litestream`) — continuous replication:
```yaml
containers:
  - name: litestream
    image: litestream/litestream:0.5.9
    args: [replicate, -config, /etc/litestream.yml]
    ports:
      - containerPort: 9090
        name: metrics
    envFrom:
      - secretRef:
          name: my-app-secrets
    volumeMounts:
      - name: config
        mountPath: /config
      - name: my-app-litestream-config
        mountPath: /etc/litestream.yml
        subPath: litestream.yml
```

### Config-Syncer (flat-file backup)

Use when the app stores config as flat files (YAML, JSON, etc.) that must survive pod restart.

**Init container** (`restore-config`) — pulls from S3 at startup.
**Sidecar** (`config-syncer`) — pushes to S3 every 60 seconds.

See `apps/template-app/base/deployment.yaml` for the full implementation.

### Metrics annotations

```yaml
spec:
  template:
    metadata:
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
```

### Reloader annotation

Triggers pod restart when a referenced Secret or ConfigMap changes:
```yaml
annotations:
  reloader.stakater.com/auto: "true"
```

---

## Sizing System (Kyverno Labels)

### Standard Tiers (Burstable QoS)

| Label value | CPU req/lim | Memory req/lim | Typical use |
|---|---|---|---|
| `micro` | 10m / 100m | 64Mi / 128Mi | Sidecars, exporters, restore inits |
| `small` | 50m / 500m | 256Mi / 512Mi | Go/Rust apps, lightweight tools |
| `medium` | 200m / 1000m | 512Mi / 1Gi | Python/Node web apps |
| `large` | 1000m / 2000m | 2Gi / 4Gi | Databases, heavy apps (Jellyfin) |
| `xlarge` | 2000m / 4000m | 4Gi / 8Gi | AI processing, heavy indexers |
| `renovate` | 2000m / 4000m | 2Gi / 4Gi | Renovate bot (burst CPU, moderate RAM) |

### Guaranteed QoS Tiers (Elite / Orichalcum only)

Guaranteed QoS means `requests == limits`. Use only for truly critical services
where OOMKill or CPU throttling is unacceptable.

| Label value | CPU | Memory | QoS class |
|---|---|---|---|
| `G-small` | 50m / 50m | 128Mi / 128Mi | Guaranteed |
| `G-medium` | 200m / 200m | 512Mi / 512Mi | Guaranteed |
| `G-large` | 1000m / 1000m | 2Gi / 2Gi | Guaranteed |
| `G-xl` | 2000m / 2000m | 4Gi / 4Gi | Guaranteed |

See [guaranteed-qos-sizing.md](guaranteed-qos-sizing.md) for full rationale.

### Per-Container Label Pattern

```yaml
# Pod template labels
labels:
  app: my-app
  vixens.io/sizing: medium               # generic fallback — matches unnamed containers
  vixens.io/sizing.my-app: medium        # main container
  vixens.io/sizing.litestream: micro     # litestream sidecar
  vixens.io/sizing.config-syncer: micro  # config-syncer sidecar
  vixens.io/sizing.restore-config: micro # restore-config init
  vixens.io/sizing.restore-db: micro     # restore-db init
  vixens.io/sizing.fix-perms: micro      # fix-perms init (root, sets ownership)
```

> **Rule**: Sidecars and init containers always use `micro` unless you have
> evidence they need more (check VPA/Goldilocks recommendations).

---

## Priority Classes

| Class | Value | Use when |
|---|---|---|
| `vixens-critical` | 100000 | Infra core — must never be evicted (Traefik, Cilium, ArgoCD) |
| `vixens-high` | 50000 | Vital user services (Home Assistant, Authentik) |
| `vixens-medium` | 10000 | Standard interactive apps (*arr, Jellyfin, tools) |
| `vixens-low` | 0 | Background jobs, downloaders, batch tasks |

---

## Annotated Template

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  labels:
    app: my-app
    # Sizing labels at Deployment level — mirrors pod template labels
    # (required for audit policy matching)
    vixens.io/sizing: small
    vixens.io/sizing.my-app: small
    vixens.io/sizing.litestream: micro
    vixens.io/sizing.config-syncer: micro
    vixens.io/sizing.restore-config: micro
    vixens.io/sizing.restore-db: micro
spec:
  replicas: 1
  revisionHistoryLimit: 3        # MANDATORY — limit etcd growth
  strategy:
    type: Recreate               # required for RWO PVCs
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
        # Sizing labels at Pod level — read by Kyverno sizing-mutate policy
        vixens.io/sizing: small
        vixens.io/sizing.my-app: small
        vixens.io/sizing.litestream: micro
        vixens.io/sizing.config-syncer: micro
        vixens.io/sizing.restore-config: micro
        vixens.io/sizing.restore-db: micro
      annotations:
        reloader.stakater.com/auto: "true"
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      priorityClassName: vixens-medium   # MANDATORY
      tolerations:                       # MANDATORY — CP scheduling
        - key: node-role.kubernetes.io/control-plane
          operator: Exists
          effect: NoSchedule
      securityContext:
        fsGroup: 1000
        fsGroupChangePolicy: OnRootMismatch
      initContainers:
        - name: restore-config           # optional — flat-file restore
          image: rclone/rclone:1.73
          # NO resources: block — injected by Kyverno from label micro
          ...
        - name: restore-db               # optional — SQLite restore
          image: litestream/litestream:0.5.9
          # NO resources: block
          ...
      containers:
        - name: my-app                   # main container
          image: my-org/my-app:latest
          # NO resources: block — injected by Kyverno from label small
          ports:
            - containerPort: 8080
              name: http
          livenessProbe:
            httpGet:
              path: /health
              port: http
            initialDelaySeconds: 15
          readinessProbe:
            httpGet:
              path: /health
              port: http
            initialDelaySeconds: 5
          envFrom:
            - secretRef:
                name: my-app-secrets
          volumeMounts:
            - name: config
              mountPath: /config
        - name: litestream               # optional — SQLite replication sidecar
          image: litestream/litestream:0.5.9
          # NO resources: block
          ...
        - name: config-syncer            # optional — flat-file sync sidecar
          image: rclone/rclone:1.73
          # NO resources: block
          ...
      volumes:
        - name: config
          persistentVolumeClaim:
            claimName: my-app-config-pvc
```

---

## Common Patterns by Container Type

| Container name | Type | Sizing label | Notes |
|---|---|---|---|
| `<app-name>` | main container | `vixens.io/sizing.<app-name>: <tier>` | Choose tier based on app profile |
| `litestream` | sidecar | `vixens.io/sizing.litestream: micro` | Always micro unless VPA says otherwise |
| `config-syncer` | sidecar | `vixens.io/sizing.config-syncer: micro` | Always micro |
| `restore-config` | init | `vixens.io/sizing.restore-config: micro` | Short-lived, always micro |
| `restore-db` | init | `vixens.io/sizing.restore-db: micro` | Short-lived, always micro |
| `fix-perms` | init | `vixens.io/sizing.fix-perms: micro` | Root init, always micro |
| `install-python-deps` | init | `vixens.io/sizing.install-python-deps: small` | May need more for pip install |
| `config-init` | init | `vixens.io/sizing.config-init: micro` | Busybox copy, always micro |

---

## QoS: Burstable vs Guaranteed

**Burstable (default)** — `requests < limits`

- CPU and memory requests are reserved on the node.
- Pod can burst up to limits if node has spare capacity.
- If node runs out of memory, Burstable pods are candidates for OOMKill before Guaranteed.
- Use for all non-critical apps.

**Guaranteed** — `requests == limits` (G-sizing tiers)

- Kubernetes never OOMKills a Guaranteed pod except under extreme node pressure.
- CPU is throttled at limit but not killed.
- Wastes headroom if the app doesn't use its full allocation.
- Reserve for `vixens-critical` priority class apps only.

**Rule of thumb**: Start with Burstable + appropriate tier. Upgrade to Guaranteed (G-sizing)
only when the app has proven it needs stable, dedicated resources at Elite tier.

---

## Checklist

Before merging any app deployment:

```
[ ] priorityClassName set
[ ] CP toleration present
[ ] Kyverno sizing labels on pod template (NOT explicit resources: blocks)
[ ] revisionHistoryLimit: 3 set
[ ] No explicit resources: blocks in any container (main, sidecars, inits)
[ ] Litestream sidecar + restore-db init paired (if SQLite)
[ ] Config-Syncer sidecar + restore-config init paired (if flat files)
[ ] yamllint passes (just lint)
[ ] kustomize build succeeds on overlay
```

---

**References**

- Living template: [`apps/template-app/`](../../apps/template-app/)
- Golden example (production): [`apps/10-home/homeassistant/`](../../apps/10-home/homeassistant/)
- Sizing tiers detail: [`sizing.deprecated/README.md`](../../apps/_shared/components/sizing.deprecated/README.md)
- Resource standards: [`RESOURCE_STANDARDS.md`](RESOURCE_STANDARDS.md)
- Quality tiers: [`quality-standards.md`](quality-standards.md)
- Guaranteed QoS: [`guaranteed-qos-sizing.md`](guaranteed-qos-sizing.md)
- Kyverno policy: [`apps/00-infra/kyverno/base/policies/sizing-mutate.yaml`](../../apps/00-infra/kyverno/base/policies/sizing-mutate.yaml)
