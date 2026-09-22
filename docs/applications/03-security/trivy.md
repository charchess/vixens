# Trivy Operator

## Deployment Information

| Environment | Deployed | Configured | Tested | Operator | Scanner / Server |
|-------------|----------|-----------|--------|----------|------------------|
| Dev | [x] | [x] | [x] | 0.34.0 | 0.74.0 |
| Prod | [x] | [x] | [x] | 0.34.0 | 0.74.0 |

## Architecture

Production uses Trivy Operator in **client/server mode**.

```text
Kubernetes workload
      |
      v
Trivy Operator
      |
      +--> ScanJob(s) per workload/container
      |      - filesystem/image inspection
      |      - secret scanning remains client-side
      |
      +--> trivy-service.security:4954
                 |
                 v
           trivy-server-0
           - vulnerability DB
           - CVE matching
```

The server is implemented as a single-replica StatefulSet with an `emptyDir` cache. There is intentionally no PVC: the vulnerability database may be downloaded again after a restart.

Relevant production files:

- `apps/03-security/trivy/overlays/prod/client-server.yaml`
- `apps/03-security/trivy/overlays/prod/scan-runtime-patch.yaml`
- `apps/03-security/trivy/overlays/prod/servicemonitor.yaml`
- `apps/03-security/trivy/overlays/prod/kustomization.yaml`

### Runtime settings

Important settings currently used in production:

```text
OPERATOR_CONCURRENT_SCAN_JOBS_LIMIT=2
OPERATOR_SCAN_JOB_TIMEOUT=20m
OPERATOR_BUILT_IN_TRIVY_SERVER=true
trivy.mode=ClientServer
trivy.serverURL=http://trivy-service.security:4954
trivy.tag=0.74.0
trivy.timeout=15m0s
OPERATOR_METRICS_VULN_ID_ENABLED=true
```

The server URL exists in **two places on purpose**:

1. `ConfigMap/trivy-operator`: consumed by the workload controller before ScanJob submission.
2. `ConfigMap/trivy-operator-trivy-config`: consumed by Trivy clients inside ScanJobs.

Removing either one can make the system appear healthy while scans stop being submitted or clients stop reaching the server.

## Why client/server mode was introduced

Standalone Trivy ScanJobs created one scanner container per workload container. Those scanner containers shared `/tmp/trivy/.cache` inside the same Job.

With Trivy 0.74.0, multi-container workloads could fail with errors such as:

```text
Failed to acquire cache or database lock
cache may be in use by another process: timeout
```

`OPERATOR_CONCURRENT_SCAN_JOBS_LIMIT` only limits the number of Jobs. It does **not** serialize sibling scanner containers inside one Job.

The production fix was to centralize the vulnerability database and matching in the Trivy server while keeping the ScanJobs for workload inspection. This was validated with a five-container TrueNAS CSI ReplicaSet: all five scanners completed successfully while sharing the ScanJob cache directory, with no cache/database lock errors.

Do not reintroduce the old `cache.backend: memory` workaround unless new evidence requires it; client/server mode is the chosen architecture.

## Reports

Trivy Operator stores detailed findings as Kubernetes CRDs. Useful report types include:

```bash
kubectl get vulnerabilityreports -A
kubectl get exposedsecretreports -A
kubectl get configauditreports -A
kubectl get rbacassessmentreports -A
kubectl get sbomreports -A
```

### Vulnerability summary

```bash
kubectl get vulnerabilityreports -A -o json |
jq -r '
  .items[]
  | [
      .metadata.namespace,
      (.metadata.labels["trivy-operator.resource.kind"] // ""),
      (.metadata.labels["trivy-operator.resource.name"] // ""),
      (.report.artifact.repository // ""),
      (.report.artifact.tag // ""),
      (.report.summary.criticalCount // 0),
      (.report.summary.highCount // 0),
      (.report.summary.mediumCount // 0),
      (.report.summary.lowCount // 0)
    ]
  | @tsv
' | column -t
```

### Detailed Critical/High findings with a fix

```bash
kubectl get vulnerabilityreports -A -o json |
jq -r '
.items[] as $r
| $r.report.vulnerabilities[]
| select(
    (.severity == "CRITICAL" or .severity == "HIGH")
    and (.fixedVersion // "") != ""
  )
| [
    .severity,
    $r.metadata.namespace,
    ($r.metadata.labels["trivy-operator.resource.name"] // ""),
    .vulnerabilityID,
    .resource,
    .installedVersion,
    .fixedVersion,
    (.score // ""),
    .title
  ]
| @tsv
' | column -t -s $'\t'
```

### `Installed` vs `Fixed`

In vulnerability reports and Grafana:

- **Installed** = version of the vulnerable component actually found in the scanned image/binary.
- **Fixed** = first version known by the advisory data to contain the fix.

Examples:

```text
libssl3t64
Installed: 3.5.4-1~deb13u2
Fixed:     3.5.5-1~deb13u2
```

This normally requires using a newer container image that contains the corrected Debian package. Do not patch a running pod manually.

For an embedded Go dependency:

```text
github.com/rclone/rclone
Installed: v1.71.2+dirty
Fixed:     1.73.5
```

The fix usually requires a rebuilt/upgraded parent image containing rclone >= 1.73.5. `+dirty` indicates the binary was built from a source tree with local/uncommitted changes, so it may not exactly match the upstream tag.

`Fixed` is a component version, **not** necessarily a container-image tag. The Container & Image Versions dashboard and Renovate provide the complementary image-update view.

## Metrics

The operator is scraped through `ServiceMonitor/trivy-operator` in namespace `security`.

Important metrics:

- `trivy_image_vulnerabilities`: aggregate vulnerability counts.
- `trivy_image_exposedsecrets`: aggregate exposed-secret counts.
- `trivy_resource_configaudits`: aggregate configuration-audit findings.
- `trivy_vulnerability_id`: one detailed series per vulnerability finding.

`OPERATOR_METRICS_VULN_ID_ENABLED=true` enables the detailed vulnerability metric used by the Vulnerability Explorer. It intentionally increases time-series cardinality; do not enable every `*_INFO` metric by default without checking the volume first.

Verify detailed metrics directly:

```bash
kubectl -n security port-forward deploy/trivy-trivy-operator 8080:8080
```

Then from another shell:

```bash
curl -s http://127.0.0.1:8080/metrics |
grep '^trivy_vulnerability_id{' |
head
```

Expected labels include `namespace`, `resource_kind`, `resource_name`, `container_name`, `image_repository`, `image_tag`, `vuln_id`, `severity`, `vuln_score`, `resource`, `installed_version`, `fixed_version` and `vuln_title`.

## Grafana dashboards

### Trivy Operator Security Overview

UID: `trivy-operator-security`

Purpose: high-level security posture / "number of fires".

Includes:

- Critical / High / Medium / Low counts.
- Unique Critical/High CVEs.
- Affected workloads.
- Fixable Critical/High findings.
- Exposed-secret and config-audit summaries.
- Top affected workloads.
- Drill-down links to the Vulnerability Explorer.

Clicking a severity counter opens the explorer with that severity preselected.

### Trivy Vulnerability Explorer

UID: `trivy-vulnerability-explorer`

Purpose: remediation-oriented detailed findings.

Filters include:

- Namespace.
- Severity.
- Workload.
- Image.
- Fix available.
- CVE / regex.
- Package / regex.

The table is intentionally ordered with remediation-relevant information first:

```text
Severity -> CVE -> Score -> Namespace -> Workload -> Package -> Installed -> Fixed -> ...
```

CVE values link to NVD.

### Container & Image Versions

UID: `container-image-versions`

This dashboard complements Trivy by showing running image/digest information and Renovate-resolved image updates. See `docs/applications/70-tools/renovate.md` and `docs/guides/security-observability.md`.

## Validation / Troubleshooting

### Operator and server

```bash
kubectl -n security get deploy trivy-trivy-operator
kubectl -n security get sts trivy-server
kubectl -n security get svc trivy-service
kubectl -n security get pods
```

Verify mode/config:

```bash
kubectl -n security get cm trivy-operator-config -o jsonpath='{.data.OPERATOR_BUILT_IN_TRIVY_SERVER}{"\n"}'
kubectl -n security get cm trivy-operator-trivy-config -o jsonpath='{.data.trivy\.mode}{"\n"}'
kubectl -n security get cm trivy-operator -o jsonpath='{.data.trivy\.serverURL}{"\n"}'
kubectl -n security get cm trivy-operator-trivy-config -o jsonpath='{.data.trivy\.serverURL}{"\n"}'
```

Expected:

```text
true
ClientServer
http://trivy-service.security:4954
http://trivy-service.security:4954
```

### Server health from the cluster

```bash
kubectl -n security run trivy-healthcheck --rm -it --restart=Never \
  --image=curlimages/curl -- \
  curl -fsS http://trivy-service.security:4954/healthz
```

Expected: `ok`.

### Look for the old locking failure

```bash
kubectl -n security logs sts/trivy-server --since=30m 2>&1 |
grep -Ei 'cache may be in use|database may be in use|failed to acquire|context deadline exceeded|fatal|panic' \
|| echo 'RAS côté serveur'

kubectl -n security logs deploy/trivy-trivy-operator --since=30m 2>&1 |
grep -Ei 'cache may be in use|database may be in use|failed to acquire|context deadline exceeded|fatal|panic' \
|| echo 'RAS côté operator'
```

### No ScanJobs are created

Check the controller-side server URL first:

```bash
kubectl -n security get cm trivy-operator \
  -o jsonpath='{.data.trivy\.serverURL}{"\n"}'
```

If built-in server mode is enabled but this value is missing, the controller can health-check/requeue without submitting ScanJobs.

### ArgoCD and mutable `prod-stable`

Production uses the mutable tag `prod-stable`. ArgoCD can temporarily keep a cached resolution after a promotion.

Hard-refresh Trivy when Git is promoted but the live revision/config is stale:

```bash
kubectl -n argocd annotate application trivy \
  argocd.argoproj.io/refresh=hard \
  --overwrite
```

For multi-source applications such as Grafana, `.status.sync.revision` can be `<none>`; use `.status.sync.revisions[]` instead.

## Change history (2026-09)

- **#3363**: Trivy scanner 0.74.0, scan timeout 15m, operator job timeout 20m.
- **#3364**: memory-cache experiment; merged but intentionally not promoted.
- **#3365**: built-in Trivy client/server architecture.
- **#3366**: expose `trivy.serverURL` to the workload controller; restored ScanJob submission.
- **#3367**: initial native Grafana security overview.
- **#3368**: detailed `trivy_vulnerability_id` metric and detailed vulnerability tables.
- **#3370**: split Overview / Vulnerability Explorer and add Container & Image Versions plus Renovate report export.
- **#3371**: dashboard UX pass, filters, drill-downs, freshness indicators, placeholder-dashboard cleanup.

## Technical Notes

- **Namespace:** `security`.
- **Category:** `03-security`.
- **Scan concurrency:** 2 Jobs.
- **Trivy server resources:** request 50m CPU / 512Mi RAM, limit 500m CPU / 1Gi RAM (`B-medium`).
- **PriorityClass:** `vixens-low` for the Trivy server.
- **Talos:** node collector remains disabled where incompatible with the read-only host filesystem.
- **GitOps:** generated `base/manifests.yaml` is not edited manually; production changes belong in overlays/patches.
