---
name: vixens-kubernetes-patterns
description: >-
  Current Kubernetes patterns and best practices for Vixens. Use when designing
  deployments, probes, volumes, resource sizing, secrets integration, Kustomize
  composition, or validating manifests before promotion.
argument-hint: "[pattern-name or component-type]"
license: MIT
compatibility: opencode
metadata:
  domain: kubernetes
  audience: homelab-operators
---

# Vixens Kubernetes patterns

Follow `AGENTS.md`, `WORKFLOW.md`, and current manifests first. This skill summarizes reusable patterns; it is not a second source of truth.

**Focus:** $ARGUMENTS

## Core principles

1. Git is desired state; persistent changes go through PR + CI + ArgoCD.
2. Reuse an existing application pattern before inventing a new abstraction.
3. Base manifests contain environment-independent structure; overlays contain environment-specific differences.
4. Prefer narrow network policy and least privilege.
5. Sensitive values live in OpenBao, not Git.
6. Important validation belongs in CI, not in a workstation-only wrapper.

## Resource sizing

Keep explicit resource requests/limits as a bootstrap-safe fallback even when VPA/Kyverno sizing metadata exists.

```yaml
spec:
  template:
    metadata:
      labels:
        vixens.io/sizing.app: V-medium
    spec:
      containers:
        - name: app
          resources:
            requests:
              cpu: 100m
              memory: 256Mi
            limits:
              cpu: 1000m
              memory: 1Gi
```

Treat current shared sizing components/policies in Git as canonical for exact tier semantics.

## Dynamic configuration

When an application does not interpolate environment variables inside its config format, generate the config in an init container and share it through `emptyDir`.

```yaml
initContainers:
  - name: generate-config
    image: busybox:1.37.0
    command: ["sh", "-c"]
    args:
      - |
        cat > /generated/app.yml <<EOF
        database:
          host: ${DB_HOST}
        EOF
    envFrom:
      - secretRef:
          name: app-secrets
    volumeMounts:
      - name: generated-config
        mountPath: /generated

containers:
  - name: app
    volumeMounts:
      - name: generated-config
        mountPath: /etc/app

volumes:
  - name: generated-config
    emptyDir: {}
```

Avoid `subPath` for files that are created dynamically at runtime; mount the containing directory instead.

## Probes

- Liveness: verify a persistent process or stable health endpoint.
- Readiness: verify the application can actually serve traffic.
- Startup: use when initialization is legitimately slow.
- Do not probe a short-lived command inside a `while true; ...; sleep` loop.

Prefer application-native HTTP/TCP health endpoints over process-name checks when available.

## Secrets

The active secret pattern is:

```text
OpenBao → ClusterSecretStore/openbao → ExternalSecret → Secret → workload
```

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-secrets
spec:
  refreshInterval: 60s
  secretStoreRef:
    name: openbao
    kind: ClusterSecretStore
  target:
    name: app-secrets
    creationPolicy: Owner
  dataFrom:
    - extract:
        key: vixens/prod/apps/category/app
```

Use the actual environment/path conventions from current manifests. Load `vixens-secrets` for diagnosis and rotation.

Do not create new `InfisicalSecret` resources. References to Infisical in superseded ADRs and historical incident reports are historical context only.

## Network policy

With default-deny, declare only required traffic:

- DNS as needed;
- ingress from Traefik or explicitly allowed workloads;
- egress to named in-cluster services/ports;
- `world` only when external access is actually required.

Use Hubble failed-flow telemetry to determine the exact missing flow before widening a policy.

## Storage

- Reuse the established CSI/StorageClass pattern for the workload type.
- Check access mode and rollout behavior for `ReadWriteOnce` volumes.
- Consider `strategy: Recreate` where two simultaneous replicas cannot mount the same RWO volume safely.
- Do not embed NAS endpoints or storage credentials in application manifests when the CSI layer already abstracts them.

## Kustomize components

Components should configure one concrete concern and be opt-in.

Good examples:

```text
revision-history-limit
sync-wave/wave-10
goldilocks/enabled
```

Avoid outcome-oriented monoliths that silently bundle unrelated settings.

## Sync ordering

Operators/CRDs and policies must exist before consumers. Use the current ArgoCD sync-wave conventions from the repo; do not copy historical wave numbers blindly.

Typical dependency direction:

```text
CRDs/operators → stores/policies → shared infra → applications → optional tooling
```

## Validation

Before merge, rely on the repository CI. Useful local/read-only checks include:

```bash
yamllint -c yamllint-config.yml <paths>
kustomize build apps/<category>/<app>/overlays/dev
```

When changing Kustomize wiring, compare rendered object kinds/names before and after to detect accidental resource removal.

After ArgoCD sync, verify health and the specific behavior changed by the PR; do not treat `Synced` alone as functional validation.

## Promotion

Dev follows `main`. Production promotion is performed only through `.github/workflows/promote-prod.yaml` after dev validation. Do not manually move `prod-stable`.

## References

- `AGENTS.md`
- `WORKFLOW.md`
- `docs/procedures/deployment-standard.md`
- `docs/guides/adding-new-application.md`
- `docs/guides/secret-management.md`
- `.opencode/skills/vixens-secrets/SKILL.md`
- `.opencode/skills/vixens-troubleshoot/SKILL.md`
