# Tasks - Deploy Mosquitto

## Phase 1: Base Structure
- [ ] Create `apps/mosquitto/base/` (namespace, statefulset, pvc, service, configmap, kustomization)
- [ ] Create `apps/mosquitto/overlays/dev/` (kustomization)

## Phase 2: Deploy Dev
- [ ] Create ArgoCD Application `argocd/overlays/dev/apps/mosquitto.yaml`
- [ ] Push to dev, wait for sync
- [ ] Test: `kubectl exec -n homeassistant <pod> -- nc -zv mosquitto.mosquitto.svc 1883`

## Phase 3: Multi-Env Rollout
- [ ] Deploy test/staging/prod via git flow
