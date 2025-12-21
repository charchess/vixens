# Plan: Monitoring & Observability Enhancements

## Phase 1: Tool Deployment (Alertmanager, Headlamp, Hubble UI)
- [ ] Task: Deploy Alertmanager in `monitoring` namespace.
- [ ] Task: Configure Alertmanager webhook receiver (using Infisical secret).
- [ ] Task: Deploy Headlamp in `tools` namespace.
- [ ] Task: Deploy Hubble UI in `kube-system` (or `monitoring`) namespace.
- [ ] Task: Create Ingresses for Headlamp and Hubble UI (Basic Access).
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Tool Deployment' (Protocol in workflow.md)

## Phase 2: Global Configuration (Goldilocks)
- [ ] Task: Audit all namespaces for Goldilocks labeling.
- [ ] Task: Apply `goldilocks.fairwinds.com/enabled=true` label to all applicable namespaces (Apps + System).
- [ ] Task: Verify data propagation in the Goldilocks dashboard.
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Global Configuration' (Protocol in workflow.md)

## Phase 3: Integration & Dashboards
- [ ] Task: Add/Verify Grafana dashboards for PostgreSQL Cluster.
- [ ] Task: Add/Verify Grafana dashboards for Traefik Ingress.
- [ ] Task: Add/Verify Grafana dashboards for Home Assistant.
- [ ] Task: Verify Loki log aggregation coverage for all active namespaces.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Integration & Dashboards' (Protocol in workflow.md)

## Phase 4: Validation & Archon Sync
- [ ] Task: Functional validation: Verify HTTPS redirection and TLS for all new tool UIs.
- [ ] Task: Test Alertmanager alert firing path (webhook reception).
- [ ] Task: Synchronize track progress with Archon (Create granular tasks).
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Validation & Archon Sync' (Protocol in workflow.md)
