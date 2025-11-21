# Tasks - Deploy Music Assistant
## Phase 1: Base Structure
- [ ] Create apps/music-assistant/base/ (namespace, deployment, pvc, service, ingress, kustomization)
- [ ] Configure volumeMounts: config PVC + NFS media-shared (ReadOnly)
- [ ] Create overlays/dev/

## Phase 2: Deploy Dev
- [ ] Create ArgoCD Application
- [ ] Push to dev, access web UI
- [ ] Configure local music provider (NFS path /data/media/music)

## Phase 3: Home Assistant Integration
- [ ] Add Music Assistant integration in HA
- [ ] Configure media players
- [ ] Test playback to HA speakers

## Phase 4: Multi-Env Rollout
- [ ] Deploy test/staging/prod
