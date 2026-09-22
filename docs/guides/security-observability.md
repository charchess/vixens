# Security Observability Runbook

This runbook documents the production security-observability stack built around Trivy Operator, Grafana, Renovate and VictoriaMetrics.

It is intended to answer three operational questions:

1. **How many security problems do we have?**
2. **What exactly are the problems and where are they?**
3. **Which container images have updates available according to Renovate?**

## End-to-end architecture

```text
                     +-------------------------+
                     | Kubernetes workloads    |
                     +------------+------------+
                                  |
                                  v
                     +-------------------------+
                     | Trivy Operator          |
                     | namespace: security     |
                     +------------+------------+
                                  |
                      ScanJobs    |    Client/server vuln matching
                                  v
                     +-------------------------+
                     | trivy-server:4954       |
                     +------------+------------+
                                  |
              +-------------------+-------------------+
              |                                       |
              v                                       v
   VulnerabilityReport CRDs                  /metrics endpoint
                                                      |
                                                      v
                                           VictoriaMetrics
                                                      |
                                                      v
                                                  Grafana
                                                      |
                         +----------------------------+-----------------------+
                         |                            |                       |
                         v                            v                       v
             Security Overview         Vulnerability Explorer    Container/Image Versions
                                                                         ^
                                                                         |
                                                            vixens_renovate_* metrics
                                                                         |
                                                          Renovate report exporter
                                                                         |
                                                              Renovate CronJob
```

## Dashboards

### 1. Trivy Operator Security Overview

Grafana UID:

```text
trivy-operator-security
```

Use this dashboard for triage and trend visibility.

It contains:

- Critical / High / Medium / Low totals.
- Unique Critical/High CVEs.
- Affected workloads.
- Fixable Critical/High findings.
- Exposed-secret summary.
- Configuration-audit summary.
- Top affected workloads.

Severity stats are drill-down links. For example, clicking `CRITICAL` opens the Vulnerability Explorer with `severity=Critical` already selected.

### 2. Trivy Vulnerability Explorer

Grafana UID:

```text
trivy-vulnerability-explorer
```

Use this dashboard when the overview says "there are N problems" and you need the actual remediation list.

Filters:

- Namespace.
- Severity.
- Workload.
- Image.
- Fix available.
- CVE / regex.
- Package / regex.

Important columns:

```text
Severity
CVE
Score
Namespace
Workload
Package
Installed
Fixed
Container
Image
Title
```

`Installed` is the vulnerable component version found in the image. `Fixed` is the component version where the advisory says the vulnerability is corrected.

`Fixed` is **not** necessarily the tag of a container image. A new parent image must contain the corrected package/library/binary.

### 3. Container & Image Versions

Grafana UID:

```text
container-image-versions
```

This dashboard has two views:

- Running containers and their current image/digest from kube-state-metrics.
- Docker dependencies and Renovate-resolved update targets from the latest Renovate report.

Useful filter:

```text
Version à jour
- All
- Yes
- No
- Unknown
```

Mapping:

```text
Yes     -> current
No      -> outdated
Unknown -> skipped
```

Use `Version à jour = No` for a quick "what can be updated?" list.

`Unknown` means Renovate did not evaluate the dependency normally; it must not be interpreted as current.

## Trivy reports from kubectl

Grafana is the primary UI, but the Kubernetes CRDs remain the source for detailed Trivy reports.

List report types:

```bash
kubectl get vulnerabilityreports -A
kubectl get exposedsecretreports -A
kubectl get configauditreports -A
kubectl get rbacassessmentreports -A
kubectl get sbomreports -A
```

Detailed Critical/High findings with a known fix:

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

## Force a fresh Renovate version report

Renovate normally runs every 6 hours.

Force an immediate run:

```bash
kubectl -n tools create job \
  --from=cronjob/renovate \
  renovate-report-bootstrap-$(date +%s)
```

Wait and show exporter output:

```bash
JOB=renovate-report-bootstrap-$(date +%s); \
kubectl -n tools create job --from=cronjob/renovate "$JOB" && \
kubectl -n tools wait --for=condition=complete job/"$JOB" --timeout=30m && \
kubectl -n tools logs job/"$JOB" -c report-exporter
```

Expected exporter log:

```text
Exported <N> Renovate Docker dependency rows to VictoriaMetrics
```

The dashboard uses an 8-hour lookback while Renovate runs every 6 hours, so a successful report remains visible between scheduled executions.

## Verify Renovate metrics

PromQL examples:

```promql
vixens_renovate_report_timestamp_seconds
```

```promql
max_over_time(vixens_renovate_image_version_info{status="outdated"}[8h])
```

If the Container & Image Versions dashboard is empty:

1. Confirm the CronJob has both containers:

   ```bash
   kubectl -n tools get cronjob renovate \
     -o jsonpath='{.spec.jobTemplate.spec.template.spec.containers[*].name}{"\n"}'
   ```

   Expected:

   ```text
   renovate report-exporter
   ```

2. Force a Job.
3. Inspect `report-exporter` logs.
4. Confirm `vixens_renovate_report_timestamp_seconds` exists in VictoriaMetrics.

## Verify Trivy client/server

Expected components:

```bash
kubectl -n security get deploy trivy-trivy-operator
kubectl -n security get sts trivy-server
kubectl -n security get svc trivy-service
```

Expected runtime config:

```bash
kubectl -n security get cm trivy-operator-config \
  -o jsonpath='{.data.OPERATOR_BUILT_IN_TRIVY_SERVER}{"\n"}'

kubectl -n security get cm trivy-operator-trivy-config \
  -o jsonpath='{.data.trivy\.mode}{"\n"}'

kubectl -n security get cm trivy-operator \
  -o jsonpath='{.data.trivy\.serverURL}{"\n"}'

kubectl -n security get cm trivy-operator-trivy-config \
  -o jsonpath='{.data.trivy\.serverURL}{"\n"}'
```

Expected:

```text
true
ClientServer
http://trivy-service.security:4954
http://trivy-service.security:4954
```

Check server health:

```bash
kubectl -n security run trivy-healthcheck --rm -it --restart=Never \
  --image=curlimages/curl -- \
  curl -fsS http://trivy-service.security:4954/healthz
```

Expected:

```text
ok
```

## Verify detailed Trivy metrics

Port-forward the operator:

```bash
kubectl -n security port-forward deploy/trivy-trivy-operator 8080:8080
```

From another terminal:

```bash
curl -s http://127.0.0.1:8080/metrics |
grep '^trivy_vulnerability_id{' |
head
```

If `trivy_vulnerability_id` is missing, verify:

```bash
kubectl -n security get cm trivy-operator-config \
  -o jsonpath='{.data.OPERATOR_METRICS_VULN_ID_ENABLED}{"\n"}'
```

Expected:

```text
true
```

## Grafana provisioning troubleshooting

### Check dashboard ConfigMaps

```bash
kubectl -n monitoring get cm | grep grafana-dashboard
```

### Check the sidecar files

```bash
kubectl -n monitoring exec deploy/grafana \
  -c grafana-sc-dashboard -- \
  sh -c 'ls -lah /tmp/dashboards; find /tmp/dashboards -maxdepth 1 -type f -print'
```

### Check Grafana logs

```bash
kubectl -n monitoring logs deploy/grafana \
  -c grafana --since=30m |
grep -Ei 'dashboard|provision|error|warn'
```

### Check the Grafana API

If the JSON exists in `/tmp/dashboards` but the UI is confusing, query the API directly:

```bash
GF_USER="$(kubectl -n monitoring get secret grafana-admin \
  -o jsonpath='{.data.admin-user}' | base64 -d)"
GF_PASS="$(kubectl -n monitoring get secret grafana-admin \
  -o jsonpath='{.data.admin-password}' | base64 -d)"

kubectl -n monitoring port-forward svc/grafana 3000:80 >/tmp/grafana-pf.log 2>&1 &
PF_PID=$!
sleep 2
curl -sS -u "${GF_USER}:${GF_PASS}" \
  'http://127.0.0.1:3000/api/search?query=Trivy' | jq .
kill "$PF_PID"
```

Provisioned dashboards are under **Dashboards -> Browse**, not Grafana's dashboard-import catalog.

## Production promotion workflow

The repository uses Git as source of truth.

Normal flow:

```text
feature branch
   -> PR
   -> CI
   -> squash merge to main
   -> dev-vYYYY.MM.PR tag
   -> manual promote-prod workflow
   -> prod-vYYYY.MM.PR
   -> prod-stable
   -> ArgoCD sync
```

Promote a version:

```bash
gh workflow run promote-prod.yaml --ref main -f version=YYYY.MM.PR
```

For the work documented here, the main PR sequence was:

```text
#3363 scanner/timeouts
#3364 memory-cache experiment (not promoted)
#3365 Trivy client/server
#3366 controller-side server URL fix
#3367 initial security dashboard
#3368 detailed vulnerability metrics
#3370 Explorer + image versions + Renovate exporter
#3371 dashboard UX/filters/hygiene
```

## Mutable `prod-stable` / ArgoCD cache gotcha

`prod-stable` is intentionally mutable. ArgoCD may temporarily retain its old resolved commit after a promotion.

If Git says production has moved but live resources are still old, hard-refresh the affected Application:

```bash
kubectl -n argocd annotate application trivy \
  argocd.argoproj.io/refresh=hard --overwrite

kubectl -n argocd annotate application grafana \
  argocd.argoproj.io/refresh=hard --overwrite

kubectl -n argocd annotate application renovate \
  argocd.argoproj.io/refresh=hard --overwrite
```

For Grafana's multi-source Application, `.status.sync.revision` may be `<none>`. Inspect all revisions instead:

```bash
kubectl -n argocd get application grafana -o json |
jq -r '.status.sync.revisions[]?'
```

## Dashboard ConfigMap immutability

Grafana dashboard ConfigMaps are `immutable: true`.

When dashboard JSON changes, use a new ConfigMap name and let ArgoCD create the replacement/prune the previous revision. Keep the Grafana dashboard UID stable.

Do not attempt to patch `.data` on an existing immutable ConfigMap.

## Known non-issues / interpretation notes

### `--cache-dir /tmp/trivy/.cache` still appears in ScanJobs

This does not mean client/server mode failed. ScanJobs still need local client-side filesystem/cache functionality. The important change is that vulnerability DB/matching is centralized in the Trivy server.

### `Fixed` is not an image tag

It is the version of the vulnerable package/library/component containing the fix. Use Renovate/image release information to determine which parent container image includes that component version.

### Large Trivy numbers do not imply the same number of unique CVEs

A single CVE can occur across several images, packages or workloads. The dashboards therefore expose both finding counts and unique CVE counts.

### A Renovate status of `skipped` is not `current`

The dashboard maps this to `Unknown` to avoid false confidence.

## Source of truth

Do not fix these systems with persistent live `kubectl edit` changes.

Use GitOps files under:

```text
apps/03-security/trivy/
apps/02-monitoring/grafana/
apps/70-tools/renovate/
argocd/overlays/prod/apps/
```

Operational commands such as hard refreshes, exact Job creation, log inspection and temporary diagnostic pods are acceptable; persistent configuration belongs in Git.
