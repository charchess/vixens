# Renovate Bot

## Informations de Déploiement

| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev | [x] | [x] | [x] | 44.103.3 |
| Prod | [x] | [x] | [x] | 44.103.3 |

## Architecture

Renovate runs as a CronJob in namespace `tools` every 6 hours.

Production additionally exports Renovate's machine-readable dependency report into VictoriaMetrics so Grafana can show **current image version vs Renovate-resolved target**.

```text
CronJob/renovate
  |
  +-- container: renovate
  |     - scans the GitOps repository
  |     - queries registries/datasources
  |     - writes /tmp/renovate/renovate-report.json
  |
  +-- container: report-exporter
        - waits for a fresh report
        - keeps datasource=docker dependencies only
        - converts them to Prometheus exposition format
        - POSTs the snapshot to VictoriaMetrics
```

Relevant production files:

- `apps/70-tools/renovate/base/cronjob.yaml`
- `apps/70-tools/renovate/overlays/prod/report-export-patch.yaml`
- `apps/70-tools/renovate/overlays/prod/report-exporter-configmap.yaml`
- `apps/70-tools/renovate/overlays/prod/victoriametrics-egress.yaml`
- `apps/70-tools/renovate/overlays/prod/kustomization.yaml`

## Validation

### CronJob and containers

```bash
kubectl -n tools get cronjob renovate

kubectl -n tools get cronjob renovate \
  -o jsonpath='{.spec.jobTemplate.spec.template.spec.containers[*].name}{"\n"}'
```

Expected in production:

```text
renovate report-exporter
```

### Recent jobs

```bash
kubectl -n tools get jobs \
  --sort-by=.metadata.creationTimestamp
```

### Logs

For the Renovate container:

```bash
kubectl -n tools logs job/<job-name> -c renovate
```

For the report exporter:

```bash
kubectl -n tools logs job/<job-name> -c report-exporter
```

Successful export example:

```text
Exported 123 Renovate Docker dependency rows to VictoriaMetrics
```

## Forcer un rapport Renovate immédiatement

The normal schedule is every 6 hours. To force a run and refresh the version dashboard immediately:

```bash
kubectl -n tools create job \
  --from=cronjob/renovate \
  renovate-report-bootstrap-$(date +%s)
```

One-liner that waits for completion and prints the exporter log:

```bash
JOB=renovate-report-bootstrap-$(date +%s); \
kubectl -n tools create job --from=cronjob/renovate "$JOB" && \
kubectl -n tools wait --for=condition=complete job/"$JOB" --timeout=30m && \
kubectl -n tools logs job/"$JOB" -c report-exporter
```

If the Job fails, inspect both containers separately. A successful Renovate run and a failed report export are intentionally treated as different concerns.

## Machine-readable report

Production enables Renovate's self-hosted file report:

```text
RENOVATE_REPORT_TYPE=file
RENOVATE_REPORT_PATH=/tmp/renovate/renovate-report.json
```

The upstream file-report feature is **experimental**, so the exporter is deliberately isolated from Renovate's core job success path.

The exporter waits for a report whose modification time belongs to the current Job, avoiding accidental reuse of a stale report left on the shared work volume.

Only dependencies with:

```text
datasource=docker
```

are exported.

## Exported metrics

### `vixens_renovate_image_version_info`

One series per Docker dependency with labels such as:

```text
repository
manager
package_file
app
dep_name
current_version
current_value
target_version
target_value
current_digest
target_digest
update_type
status
```

The sample value is the report timestamp.

### `vixens_renovate_image_dependencies_total`

Counts dependencies by status.

### `vixens_renovate_report_timestamp_seconds`

Timestamp of the most recently imported report; used by Grafana to show report freshness/age.

## Status semantics

The exporter maps Renovate results to three statuses:

| Export status | Dashboard `Version à jour` | Meaning |
|---------------|-----------------------------|---------|
| `current` | Yes | Renovate has no update candidate for the dependency. |
| `outdated` | No | Renovate has at least one update candidate. |
| `skipped` | Unknown | Renovate skipped/could not evaluate the dependency normally. |

`Unknown` is intentionally different from `Yes`: a skipped dependency is not proof that the image is current.

## Renovate target semantics

The dashboard column **Renovate target** is not a naive registry `latest` tag.

It is derived from Renovate's own update candidates and therefore inherits Renovate's datasource, versioning and repository rules. This is important for images with custom tag formats or policies that intentionally block certain updates.

The exporter chooses the furthest update candidate currently reported by Renovate, with update types ranked roughly as:

```text
digest < pin < bump < replacement < patch < minor < major
```

This makes the dashboard useful for visibility while Renovate remains the source of truth for what updates are actually known under repository policy.

## Container & Image Versions dashboard

Grafana UID: `container-image-versions`.

The dashboard combines two separate sources:

1. **Running containers** from kube-state-metrics (`kube_pod_container_info` + running state).
2. **Available image updates** from Renovate-exported metrics.

Useful filters include:

- Namespace for running containers.
- `Version à jour = Yes / No / Unknown`.
- Application.
- Update type (`patch`, `minor`, `major`, digest, etc.).

Top stats include:

- Number of running containers.
- Images with an update available.
- Major updates available.
- Age/freshness of the latest Renovate report.

The table shows current and target versions/digests and links the GitOps file back to GitHub for remediation.

### Why there is an 8-hour lookback

Renovate runs every 6 hours. Grafana queries the latest series over an 8-hour window so the table remains populated between scheduled runs and tolerates modest scheduling delays.

If the freshness stat grows beyond the expected window, investigate the CronJob/exporter rather than assuming all images are current.

## VictoriaMetrics import path

The exporter sends Prometheus text to:

```text
http://vmsingle-vm-stack.monitoring.svc.cluster.local:8428/api/v1/import/prometheus
```

A dedicated additive Cilium policy permits the Renovate pod to reach this service/port. Do not broaden the policy to unrestricted monitoring namespace access unless required by another feature.

## Configuration

### Secrets Infisical

**Path:** `/apps/70-tools/renovate`

Required variable:

- `RENOVATE_TOKEN`: GitHub token with repository/workflow permissions required by Renovate.

### Scheduling

- **Schedule:** every 6 hours (`0 */6 * * *`).
- **Concurrency policy:** `Forbid`.
- **Job history:** successful and failed Jobs retained according to CronJob configuration.

### Enabled managers

The repository uses Renovate managers including:

- Terraform.
- Helm values.
- Kubernetes.
- Regex/custom patterns.

### Custom versioning

Custom `packageRules` handle non-standard tags such as LinuxServer.io or date/nightly schemes. This is one reason the dashboard deliberately uses Renovate's resolved target instead of attempting to calculate `latest` independently.

## Operational checks

### Verify the report-export integration is deployed

```bash
kubectl -n tools get cronjob renovate \
  -o jsonpath='{range .spec.jobTemplate.spec.template.spec.containers[*]}{.name}{"\t"}{.image}{"\n"}{end}'

kubectl -n tools get cm renovate-report-exporter
```

### Query imported metrics from VictoriaMetrics/Prometheus-compatible datasource

Example PromQL:

```promql
vixens_renovate_report_timestamp_seconds
```

```promql
max_over_time(vixens_renovate_image_version_info{status="outdated"}[8h])
```

### Exporter reports zero rows

Check:

1. Renovate actually wrote the report file.
2. The report schema still contains `repositories -> packageFiles -> deps` as expected by the exporter.
3. Docker dependencies use `datasource="docker"`.
4. VictoriaMetrics import endpoint is reachable through Cilium.

The report format is experimental; if upstream changes the schema, update `report-exporter-configmap.yaml`. The exporter logs an error but should not make the Renovate maintenance job itself fail solely because observability export failed.

## ArgoCD / production promotion

Production applications track `prod-stable`. After promotion, Argo can temporarily retain a cached resolution of this mutable tag.

If Renovate GitOps changes are promoted but the live CronJob still shows only the `renovate` container, hard-refresh the application:

```bash
kubectl -n argocd annotate application renovate \
  argocd.argoproj.io/refresh=hard \
  --overwrite
```

Then verify `renovate report-exporter` is present in the CronJob template.

## Known Issues

### OOMKilled During Large Dependency Scans (Resolved)

Renovate is bursty and previously exceeded its memory limit during large dependency scans. Production therefore uses explicit resource sizing rather than relying on VPA for the CronJob.

The operational principle remains: a failed dependency-maintenance run is more expensive than allowing sufficient burst headroom.

## Change history (2026-09)

- **#3370**: add Renovate file report, exporter sidecar, VictoriaMetrics metrics and Cilium egress; introduce Container & Image Versions dashboard.
- **#3371**: add `Version à jour` filter, update-type filter, freshness/age and update-count stats, GitOps links and digest visibility.

## Références

- [Renovate Documentation](https://docs.renovatebot.com/)
- [Self-hosted configuration](https://docs.renovatebot.com/self-hosted-configuration/)
- [Kubernetes Manager](https://docs.renovatebot.com/modules/manager/kubernetes/)
- `docs/guides/security-observability.md`
