# Trivy Operator

## Deployment Information
| Environment | Deployed | Configured | Tested | Version |
|-------------|----------|-----------|-------|---------|
| Dev         | [x]      | [x]       | [x]   | v0.31.0 |
| Prod        | [x]      | [x]       | [x]   | v0.31.0 |

## Validation
**URL:** N/A (Operator-based / Grafana Dashboard ID 17813)

### Automatic Validation (CLI)


### Manual Validation
1. Verify reports are generated in the cluster.
2. Verify Grafana dashboard (if deployed) shows security metrics.

## Technical Notes
- **Namespace:** `security`
- **Category:** `03-security`
- **Status:** **Elite (Goldified)**
- **Specifics:** 
    - **Resource Limit:** 1Gi RAM (Guaranteed QoS) to prevent OOM during large scans.
    - **Gentleman Mode:** `scanJobsConcurrentLimit` set to 2 to avoid cluster CPU saturation.
    - **Talos Compatibility:** `nodeCollector` disabled (incompatible with Talos RO filesystem).
    - **ArgoCD Compliance:** ServiceMonitors and ComplianceReports filtered out via Kustomize delete patches to avoid missing CRD errors.
    - **Security Profile:** Namespace `security` set to `privileged` to allow image scanning.
