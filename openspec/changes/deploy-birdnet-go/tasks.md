# Tasks - Deploy BirdNET-Go
## Phase 1: Base Structure
- [ ] Create apps/birdnet/base/ (namespace, deployment, pvc, service, ingress, configmap, kustomization)
- [ ] Configure RTSP stream URL in configmap
- [ ] Create overlays/dev/

## Phase 2: Deploy Dev
- [ ] Create ArgoCD Application
- [ ] Push to dev, monitor pod logs for detections
- [ ] Test web UI access

## Phase 3: Home Assistant Integration
- [ ] Add REST sensor to Home Assistant config
- [ ] Test automation trigger on detection

## Phase 4: Multi-Env Rollout
- [ ] Deploy test/staging/prod
