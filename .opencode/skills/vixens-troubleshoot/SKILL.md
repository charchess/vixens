---
name: vixens-troubleshoot
description: >-
  Vixens cluster troubleshooting adapter. Use for broken apps, CrashLoopBackOff,
  ImagePullBackOff, OOMKilled, failed deployments, ingress/TLS, missing secrets,
  PVC issues, ArgoCD drift, networking failures, or high restart counts.
argument-hint: "[app-name, namespace, or symptom]"
license: MIT
compatibility: opencode
metadata:
  domain: kubernetes
  audience: homelab-operators
---

# Vixens troubleshooting

Follow `AGENTS.md` and `WORKFLOW.md` first. This skill is an optional diagnostic adapter; it does not redefine repository workflow.

**Focus:** $ARGUMENTS

## Non-negotiable GitOps rules

- Git is the desired-state source of truth.
- Start from the current GitHub `main` and inspect open PRs before proposing a fix.
- Diagnosis may use read-only cluster commands.
- Persistent fixes go through branch → PR → CI → ArgoCD.
- Do not use `kubectl apply`, `edit`, `patch`, `set image`, or similar commands as a persistent fix.
- Do not manually create or move production tags. Production promotion goes through `.github/workflows/promote-prod.yaml` only.
- Load `vixens-argocd-safety` before any invasive ArgoCD recovery action.

## Diagnostic flow

```text
1. Observe
2. Identify the failing layer
3. Read the current desired state in Git
4. Compare runtime vs desired state
5. Form the smallest testable hypothesis
6. Fix in Git
7. Let CI and ArgoCD reconcile
8. Verify the symptom and related telemetry
```

Prefer evidence from events, logs, metrics, Hubble/Loki and resource status over speculative changes.

## Quick read-only checks

```bash
kubectl get nodes
kubectl get pods -A
kubectl get events -A --field-selector type=Warning --sort-by='.lastTimestamp' | tail -30
kubectl -n argocd get applications
```

For one workload:

```bash
kubectl -n "$NS" describe pod "$POD"
kubectl -n "$NS" logs "$POD" -c "$CONTAINER" --previous
kubectl -n "$NS" get deploy,statefulset,daemonset,svc,ingress,pvc
```

## CrashLoopBackOff / OOMKilled

Check termination reason, previous logs, probes and effective resources:

```bash
kubectl -n "$NS" get pod "$POD" -o jsonpath='{.status.containerStatuses[*].lastState.terminated.reason}'
kubectl -n "$NS" get pod "$POD" -o jsonpath='{.spec.containers[*].resources}' | jq .
kubectl -n "$NS" logs "$POD" -c "$CONTAINER" --previous
```

Then inspect the corresponding deployment/statefulset in Git. Do not patch production resources to make the symptom disappear.

## Service / ingress / TLS

```bash
kubectl -n "$NS" get svc,endpoints,ingress
kubectl -n "$NS" describe ingress "$INGRESS"
kubectl -n "$NS" get certificate
kubectl -n "$NS" describe certificate "$CERT"
kubectl -n traefik logs -l app.kubernetes.io/name=traefik --tail=100
kubectl -n cert-manager logs -l app=cert-manager --tail=100
```

Typical layers to distinguish:

```text
DNS → Traefik entrypoint → route → Service → Endpoint/Pod
                     ↘ cert-manager / TLS Secret
```

## Secrets

The active architecture is:

```text
OpenBao → ClusterSecretStore/openbao → ExternalSecret → Secret → workload
```

Load `vixens-secrets` for detailed secret troubleshooting.

Useful checks:

```bash
kubectl get clustersecretstore openbao
kubectl -n "$NS" get externalsecret
kubectl -n "$NS" describe externalsecret "$EXTERNAL_SECRET"
kubectl -n external-secrets logs deploy/external-secrets --tail=200
```

Do not print Secret values. Check existence, key names and readiness instead.

Historical `InfisicalSecret` examples in old ADRs/reports are not an active troubleshooting path.

## Network policy / Cilium

When an application can resolve a destination but connections fail, inspect policy before widening egress:

```bash
kubectl -n "$NS" get ciliumnetworkpolicy,networkpolicy
kubectl -n "$NS" describe ciliumnetworkpolicy
```

Use Hubble/Grafana/Loki failed-flow data where available to identify exact source, destination, port and drop reason. Prefer a narrow policy fix over namespace-wide or world-wide access.

## PVC / storage

```bash
kubectl -n "$NS" get pvc
kubectl -n "$NS" describe pvc "$PVC"
kubectl get storageclass
kubectl get pv
```

Then inspect the relevant CSI manifests and application storage pattern in Git. Storage architecture decisions belong in the storage/CSI runbooks, not in ad-hoc runtime patches.

## ArgoCD drift

First establish whether Git or the cluster is wrong:

```bash
kubectl -n argocd get application "$APP" -o yaml
```

Compare the application's `targetRevision`, source path and rendered manifests with current Git. If Git is correct, allow ArgoCD to reconcile. If Git is wrong, fix Git through a PR.

Never repair production by manually moving `prod-stable`. Use the canonical promotion workflow after dev validation.

## Safe runtime actions

Runtime actions are acceptable only when they are operational and do not replace desired state, for example:

- reading logs/status/events;
- deleting a failed pod that a controller will recreate, when the desired state is already correct;
- restarting a workload after an external credential rotation when the app cannot reload it dynamically;
- cordon/drain for node maintenance;
- forcing an ArgoCD refresh without changing desired state.

If an action changes persistent application configuration, it belongs in Git.

## References

- `AGENTS.md`
- `WORKFLOW.md`
- `.opencode/skills/vixens-argocd-safety/SKILL.md`
- `.opencode/skills/vixens-secrets/SKILL.md`
- `.opencode/skills/vixens-kubernetes-patterns/SKILL.md`
- `docs/guides/secret-management.md`
- `docs/guides/promotion-workflow.md`
