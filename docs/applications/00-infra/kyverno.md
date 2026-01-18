# Kyverno

## Overview

Kyverno is a Kubernetes-native policy engine that manages policies as Kubernetes resources. It enables policy-as-code for cluster governance, security, and best practices enforcement.

**Category:** Infrastructure (00-infra)
**Namespace:** `kyverno`
**Helm Chart:** [kyverno/kyverno](https://kyverno.github.io/kyverno/) v3.4.0

## Architecture

Kyverno consists of 4 controllers:

| Controller | Purpose |
|------------|---------|
| Admission Controller | Validates/mutates resources at admission time |
| Background Controller | Applies policies to existing resources |
| Cleanup Controller | Handles resource cleanup policies |
| Reports Controller | Generates policy reports |

## Policies

### Validation Policies (Audit Mode)

| Policy | Description |
|--------|-------------|
| `require-resources` | Ensures all containers have CPU/memory requests and limits |
| `require-revision-history-limit` | Ensures Deployments/StatefulSets have revisionHistoryLimit <= 3 |

### Mutation Policies

| Policy | Description |
|--------|-------------|
| `add-default-labels` | Adds `app.kubernetes.io/managed-by: argocd` label to workloads |

## Configuration

### Dev Environment
- Single replica per controller
- Audit mode only (no enforcement)
- Control-plane tolerations

### Prod Environment
- HA configuration available (3 replicas per controller)
- Ready for Enforce mode when policies are validated

## Accessing Policy Reports

Policy reports are available as Kubernetes resources:

```bash
# List policy reports
kubectl get policyreport -A
kubectl get clusterpolicyreport

# Check violations
kubectl get policyreport -A -o json | jq '.items[].results[] | select(.result == "fail")'
```

## Related Tasks

- `vixens-wkrp`: Global revisionHistoryLimit patch (related but manual approach)
- `vixens-mu5b`: Original Kyverno deployment task (merged into vixens-w30)

## Resources

- [Kyverno Documentation](https://kyverno.io/docs/)
- [Policy Library](https://kyverno.io/policies/)
- [Helm Chart Values](https://github.com/kyverno/kyverno/blob/main/charts/kyverno/values.yaml)

## Troubleshooting

### Webhook Errors
If admission webhooks fail, check controller health:
```bash
kubectl get pods -n kyverno
kubectl logs -n kyverno -l app.kubernetes.io/component=admission-controller
```

### Policy Not Applied
Check if the policy is active:
```bash
kubectl get clusterpolicy
kubectl describe clusterpolicy <name>
```
