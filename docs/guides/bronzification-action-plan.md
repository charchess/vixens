# Bronzification Action Plan
**Status:** 19/25 apps completed (76%)  
**Remaining:** 6 apps blocked  
**Priority:** Complete traefik + cert-manager → 24/25 (96%)

---

## Quick Reference

| App | Namespace | Blocker | Effort | Priority |
|-----|-----------|---------|--------|----------|
| **traefik** | traefik | Policy errors | 30 min | 🔴 HIGH |
| **cert-manager** | cert-manager | Helm chart | 1-2 hrs | 🔴 HIGH |
| **cert-manager-webhook** | cert-manager | Helm chart | 1-2 hrs | 🔴 HIGH |
| **cert-manager-cainjector** | cert-manager | Helm chart | 1-2 hrs | 🔴 HIGH |
| **cert-manager-webhook-gandi** | cert-manager | Helm chart | 1-2 hrs | 🔴 HIGH |
| **trivy-operator** | security | Missing resources | 30 min | 🟡 MEDIUM |
| **cloudnative-pg** | cnpg-system | Operator config | 1 hr | 🟡 MEDIUM |
| **infisical-operator** | infisical-operator-system | Missing config | 1 hr | 🟡 MEDIUM |
| **vpa-recommender** | vpa | Infra component | 30 min | 🟢 LOW |
| **vpa-updater** | vpa | Infra component | 30 min | 🟢 LOW |

---

## Phase 1: Traefik (30 minutes) 🔴

### Problem
Policy evaluation errors on Pods block Bronze maturity despite passing all actual validations.

### Root Cause
```json
{
  "policy": "check-service-binding",
  "result": "error",
  "message": "JMESPath query failed: Unknown key 'app' in path"
}
```

Policies expect `metadata.labels.app`, but traefik uses `app.kubernetes.io/name` (Kubernetes standard).

### Solution: Add Legacy `app` Label

**File:** `apps/00-infra/traefik/values/prod.yaml`

```bash
# 1. Create branch
git checkout -b fix/traefik-bronze-policy-errors

# 2. Edit values
vim apps/00-infra/traefik/values/prod.yaml
```

**Add to `additionalArguments` section:**
```yaml
deployment:
  podLabels:
    app: traefik  # For legacy Kyverno policy compatibility
```

**Validation:**
```bash
yamllint -c yamllint-config.yml apps/00-infra/traefik/values/prod.yaml
kustomize build apps/00-infra/traefik/overlays/prod
```

**Commit & Deploy:**
```bash
git add apps/00-infra/traefik/values/prod.yaml
git commit -m "fix(traefik): add app label for policy compatibility"
git push -u origin fix/traefik-bronze-policy-errors
gh pr create --title "fix(traefik): add app label for policy compatibility" \
  --body "Fixes check-service-binding and check-servicemonitor policy errors by adding legacy 'app' label" \
  --base main
gh pr merge --squash --auto --delete-branch $(gh pr list --head fix/traefik-bronze-policy-errors --json number --jq '.[0].number')
```

**Verify:**
```bash
# After merge + prod-stable update
sleep 120  # Wait for ArgoCD sync + maturity controller
kubectl get deployment -n traefik traefik \
  -o jsonpath='Maturity: {.metadata.labels.vixens\.io/maturity}'
# Expected: gold or platinum
```

### Alternative: Fix Policies (Better Long-Term)

**Files:** 
- `apps/00-infra/kyverno-policies/base/check-service-binding.yaml`
- `apps/00-infra/kyverno-policies/base/check-servicemonitor.yaml`

Update JMESPath to use fallback:
```yaml
context:
  - name: app_label
    variable:
      jmesPath: request.object.metadata.labels.app || request.object.metadata.labels."app.kubernetes.io/name" || 'none'
```

---

## Phase 2: Cert-Manager (1-2 hours) 🔴

### Problem
Helm chart v1.14.4 doesn't expose `readinessProbe` configuration for controller/cainjector.

### Solution: Kustomize JSON Patch Overlay

#### Step 1: Create Overlay Structure
```bash
mkdir -p apps/00-infra/cert-manager/overlays/prod
```

#### Step 2: Create Kustomization with Patches

**File:** `apps/00-infra/cert-manager/overlays/prod/kustomization.yaml`

```yaml
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

# No resources - we're patching Helm output via ArgoCD multi-source

patches:
  # cert-manager controller - add readinessProbe
  - patch: |-
      - op: add
        path: /spec/template/spec/containers/0/readinessProbe
        value:
          httpGet:
            path: /livez
            port: http-healthz
            scheme: HTTP
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 3
    target:
      kind: Deployment
      name: cert-manager

  # cert-manager-cainjector - add both probes
  - patch: |-
      - op: add
        path: /spec/template/spec/containers/0/livenessProbe
        value:
          httpGet:
            path: /healthz
            port: http
            scheme: HTTP
          initialDelaySeconds: 10
          periodSeconds: 10
      - op: add
        path: /spec/template/spec/containers/0/readinessProbe
        value:
          httpGet:
            path: /healthz
            port: http
            scheme: HTTP
          initialDelaySeconds: 5
          periodSeconds: 5
    target:
      kind: Deployment
      name: cert-manager-cainjector
```

#### Step 3: Update ArgoCD Application

**File:** `argocd/overlays/prod/apps/cert-manager.yaml`

Add third source for Kustomize patches:

```yaml
spec:
  sources:
    # 1. Helm chart
    - repoURL: https://charts.jetstack.io
      chart: cert-manager
      targetRevision: v1.14.4
      helm:
        valueFiles:
          - $values/apps/00-infra/cert-manager/values/common.yaml
          - $values/apps/00-infra/cert-manager/values/prod.yaml
    
    # 2. Values from Git
    - repoURL: https://github.com/charchess/vixens.git
      targetRevision: prod-stable
      ref: values
    
    # 3. Kustomize patches (NEW)
    - repoURL: https://github.com/charchess/vixens.git
      targetRevision: prod-stable
      path: apps/00-infra/cert-manager/overlays/prod
```

#### Step 4: Deploy

```bash
git checkout -b feat/cert-manager-kustomize-probes

# Create overlay
mkdir -p apps/00-infra/cert-manager/overlays/prod
cat > apps/00-infra/cert-manager/overlays/prod/kustomization.yaml << 'EOF'
# (paste kustomization.yaml content from above)
EOF

# Update ArgoCD app
vim argocd/overlays/prod/apps/cert-manager.yaml
# (add third source as shown above)

# Validate
yamllint -c yamllint-config.yml apps/00-infra/cert-manager/overlays/prod/kustomization.yaml
yamllint -c yamllint-config.yml argocd/overlays/prod/apps/cert-manager.yaml

# Commit
git add apps/00-infra/cert-manager/overlays/prod/
git add argocd/overlays/prod/apps/cert-manager.yaml
git commit -m "feat(cert-manager): add readinessProbe via Kustomize patches

Helm chart v1.14.4 doesn't expose readinessProbe for controller/cainjector.
Use Kustomize JSON patches to add probes post-render.

Fixes Bronze require-probes policy violations."

git push -u origin feat/cert-manager-kustomize-probes
gh pr create --title "feat(cert-manager): add readinessProbe via Kustomize patches" \
  --body "$(cat <<'EOBODY'
## Problem
Helm chart jetstack/cert-manager:v1.14.4 doesn't expose readinessProbe configuration for:
- cert-manager (controller)
- cert-manager-cainjector

## Solution
Use ArgoCD multi-source with Kustomize overlay to patch Deployments post-render.

## Changes
- Add apps/00-infra/cert-manager/overlays/prod/kustomization.yaml with JSON patches
- Update argocd/overlays/prod/apps/cert-manager.yaml to include third source

## Patches Applied
- cert-manager: Add readinessProbe (httpGet /livez)
- cert-manager-cainjector: Add liveness + readiness probes

## Bronze Violations Fixed
- require-probes: All cert-manager components now have probes

## Validation
- yamllint passed
- ArgoCD multi-source tested (similar to it-tools pattern)
EOBODY
)" \
  --base main

gh pr merge --squash --auto --delete-branch $(gh pr list --head feat/cert-manager-kustomize-probes --json number --jq '.[0].number')
```

#### Step 5: Verify

```bash
# After merge + prod-stable update + ArgoCD sync
sleep 120

# Check probes applied
kubectl get deployment -n cert-manager cert-manager \
  -o jsonpath='{.spec.template.spec.containers[0].readinessProbe}' | jq

kubectl get deployment -n cert-manager cert-manager-cainjector \
  -o jsonpath='{.spec.template.spec.containers[0].livenessProbe}' | jq

# Check maturity
kubectl get deployment -n cert-manager \
  -l app.kubernetes.io/instance=cert-manager \
  -o custom-columns='NAME:.metadata.name,MATURITY:.metadata.labels.vixens\.io/maturity'
```

---

## Phase 3: Trivy Operator (30 minutes) 🟡

### Problem
Missing `resources.limits` on trivy-operator container.

### Solution: Add Resources to Helm Values

**File:** `apps/03-security/trivy/base/values.yaml` (or similar)

```bash
# 1. Find trivy configuration
ls -la apps/03-security/trivy/

# 2. Create branch
git checkout -b feat/trivy-operator-bronze-resources

# 3. Edit values (adapt to actual structure)
vim apps/03-security/trivy/base/values.yaml
```

**Add resources:**
```yaml
trivy:
  resources:
    requests:
      cpu: 50m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 256Mi
```

**Commit & Deploy:**
```bash
git add apps/03-security/trivy/
git commit -m "feat(trivy-operator): add resources for Bronze maturity"
git push -u origin feat/trivy-operator-bronze-resources
gh pr create --title "feat(trivy-operator): add resources for Bronze maturity" \
  --body "Fixes require-resources Bronze policy violation" --base main
gh pr merge --squash --auto --delete-branch $(gh pr list --head feat/trivy-operator-bronze-resources --json number --jq '.[0].number')
```

---

## Phase 4: CloudNativePG Operator (1 hour) 🟡

### Investigation Required

```bash
# Check deployment structure
kubectl get deployment -n cnpg-system cloudnative-pg -o yaml | head -100

# Check if managed by Helm
helm list -n cnpg-system

# Check for values/configuration
ls -la apps/04-databases/cloudnative-pg/

# Check PolicyReports
kubectl get policyreports -n cnpg-system -o json | \
  jq -r '.items[] | select(.scope.kind == "Deployment") | 
    .results[] | select(.result == "fail") | {policy, message}'
```

**Next Steps:**
1. Identify configuration method (Helm/Kustomize/Operator CRD)
2. Find Bronze violations (likely resources + probes)
3. Apply fixes based on deployment method
4. Verify maturity update

---

## Phase 5: Remaining Operators (2 hours total) 🟢

### Infisical Operator
- Namespace: `infisical-operator-system`
- Deployment: `infisical-opera-controller-manager`
- Likely issue: Resources/probes configuration

### VPA (2 deployments)
- Namespace: `vpa`
- Deployments: `vpa-vertical-pod-autoscaler-recommender`, `vpa-vertical-pod-autoscaler-updater`
- Priority: LOW (infrastructure components)
- Action: Investigate together, likely same root cause

**Investigation Pattern:**
```bash
APP_NS="vpa"
APP_NAME="vpa-vertical-pod-autoscaler-recommender"

# 1. Find active ReplicaSet
RS=$(kubectl get replicasets -n $APP_NS -o json | \
  jq -r ".items[] | select(.spec.replicas > 0) | .metadata.name" | \
  grep "^$APP_NAME")

# 2. Check violations
kubectl get policyreports -n $APP_NS -o json | \
  jq -r --arg rs "$RS" '.items[] | select(.scope.name == $rs) | 
    .results[] | select(.result == "fail") | 
    {policy, category, message}'

# 3. Check configuration structure
ls -la apps/00-infra/vpa/
```

---

## Success Criteria

### Completion Metrics

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| Apps ≥ Bronze | 25/25 (100%) | 19/25 (76%) | 🟡 In Progress |
| Apps ≥ Silver | 15/25 (60%) | 10/25 (40%) | 🟢 Exceeded |
| Apps ≥ Gold | 5/25 (20%) | 4/25 (16%) | 🟢 On Track |
| Critical Infra Bronze | 5/5 (100%) | 0/5 (0%) | 🔴 Blocked |

**Critical Infrastructure Apps:**
- traefik ← MUST complete
- cert-manager (×4) ← MUST complete

### Verification Commands

**Check overall progress:**
```bash
kubectl get deployments -A -o json | \
  jq -r '.items[] | 
    select(.metadata.labels."vixens.io/maturity" != null) | 
    "\(.metadata.namespace)/\(.metadata.name): \(.metadata.labels."vixens.io/maturity")"' | \
  sort
```

**Count by tier:**
```bash
kubectl get deployments -A -o json | \
  jq -r '[.items[] | 
    select(.metadata.labels."vixens.io/maturity" != null) | 
    .metadata.labels."vixens.io/maturity"] | 
    group_by(.) | 
    map({tier: .[0], count: length}) | 
    .[]'
```

**Find remaining `none`:**
```bash
kubectl get deployments -A -o json | \
  jq -r '.items[] | 
    select(.metadata.labels."vixens.io/maturity" == "none") | 
    "\(.metadata.namespace)/\(.metadata.name)"'
```

---

## Timeline Estimate

| Phase | Duration | Cumulative | Completion |
|-------|----------|------------|------------|
| **Phase 1: Traefik** | 30 min | 30 min | 20/25 (80%) |
| **Phase 2: Cert-Manager** | 2 hrs | 2.5 hrs | 24/25 (96%) ✅ Target |
| **Phase 3: Trivy** | 30 min | 3 hrs | 25/25 (100%) ⭐ |
| **Phase 4: CloudNativePG** | 1 hr | 4 hrs | Bonus |
| **Phase 5: Remaining** | 2 hrs | 6 hrs | Bonus |

**Recommended Milestone:** Complete Phase 1-2 (24/25 apps, 96%)

---

## Rollback Procedures

### If PR Breaks Deployment

**Traefik:**
```bash
# Revert PR
gh pr reopen <pr-number>
git revert <commit-hash>
git push origin main

# Rollback ArgoCD
kubectl -n argocd patch application traefik \
  --type json -p='[{"op": "replace", "path": "/spec/source/targetRevision", "value": "<previous-commit>"}]'
```

**Cert-Manager:**
```bash
# Remove third source from ArgoCD app
kubectl -n argocd edit application cert-manager
# Delete the third source entry

# ArgoCD will revert to Helm-only deployment
```

### If Maturity Controller Breaks

```bash
# Check logs
kubectl logs -n kyverno -l app=maturity-controller --tail=100

# Manual run
kubectl create job -n kyverno debug-maturity --from=cronjob/maturity-controller

# Rollback script
git checkout <previous-commit> apps/00-infra/maturity-controller/
```

---

## Notes

- **Maturity Controller Timing:** Runs every 15 minutes (`:00, :15, :30, :45`)
- **ArgoCD Sync:** Typically 1-3 minutes after `prod-stable` tag update
- **ReplicaSet Cleanup:** Manual step required (not automated)
- **Policy Errors vs Fails:** Both block maturity (see investigation report)

**References:**
- Investigation Report: `docs/troubleshooting/bronzification-blockers-2026-03-12.md`
- Maturity System: `docs/adr/023-7-tier-goldification-system-v2.md`
- Kyverno Policies: `apps/00-infra/kyverno-policies/base/`
