# Tasks - Synology CSI Path Isolation

- [ ] Create `/synology-csi` folder/path in Infisical UI (dev environment)
- [ ] Move `synology-csi-client-info` secret to `/synology-csi` path in Infisical
- [ ] Update InfisicalSecret CRD `secretsPath` to `/synology-csi` (currently at root `/`)
- [ ] Apply InfisicalSecret changes via Git + ArgoCD
- [ ] Wait for secret sync (60s) and verify secret content in Kubernetes
- [ ] Restart Synology CSI pods to pick up updated secret
- [ ] Test Synology CSI functionality (PVC creation, mounting)
- [ ] Update ADR 007 with Synology CSI migration details
- [ ] Document path isolation pattern in specs
- [ ] Validate with `openspec validate --strict`
- [ ] Archive this change after successful implementation
