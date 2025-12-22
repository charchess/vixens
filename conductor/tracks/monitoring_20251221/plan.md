# Plan: Monitoring & Observability Enhancements

## Phase 1: Tool Deployment (Alertmanager, Headlamp, Hubble UI)
- [x] Task: Deploy Alertmanager in `monitoring` namespace.
- [x] Task: Configure Alertmanager webhook receiver (using Infisical secret).
- [x] Task: Deploy Headlamp in `tools` namespace.
- [x] Task: Deploy Hubble UI in `kube-system` (or `monitoring`) namespace.
- [x] Task: Create Ingresses for Headlamp and Hubble UI (Basic Access).
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Tool Deployment' (Protocol in workflow.md)

## Phase 2: Global Configuration (Goldilocks)
- [x] Task: Audit all namespaces for Goldilocks labeling.
- [x] Task: Apply `goldilocks.fairwinds.com/enabled=true` label to all applicable namespaces (Apps + System).
- [ ] Task: Verify data propagation in the Goldilocks dashboard.
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Global Configuration' (Protocol in workflow.md)

## Phase 3: Integration & Dashboards
- [x] Task: Add/Verify Grafana dashboards for PostgreSQL Cluster.
- [x] Task: Add/Verify Grafana dashboards for Traefik Ingress.
- [x] Task: Add/Verify Grafana dashboards for Home Assistant.
- [x] Task: Verify Loki log aggregation coverage for all active namespaces.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Integration & Dashboards' (Protocol in workflow.md)

## Phase 4: Validation & Archon Sync
- [ ] Task: Functional validation: Verify HTTPS redirection and TLS for all new tool UIs.
- [ ] Task: Test Alertmanager alert firing path (webhook reception).
- [x] Task: Synchronize track progress with Archon (Create granular tasks).
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Validation & Archon Sync' (Protocol in workflow.md)
