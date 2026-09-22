# Grafana

## Informations de Déploiement

| Environnement | Déployé | Configuré | Testé | Chart |
|---------------|---------|-----------|-------|-------|
| Dev | [x] | [x] | [x] | grafana 10.3.0 |
| Prod | [x] | [x] | [x] | grafana 10.3.0 |

## Status Elite ✅

- **PriorityClass:** `vixens-critical`.
- **Probes:** Liveness & Readiness configured.
- **Storage:** RWO (Retain in Prod) with Recreate strategy.
- **Dashboard provisioning:** sidecar watches ConfigMaps labelled `grafana_dashboard: "1"` across namespaces configured by the Helm values.

## Validation

**URL:** `https://grafana.[env].truxonline.com`

### Méthode Automatique (Curl)

```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://grafana.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS et le contenu
curl -L -k https://grafana.dev.truxonline.com/login | grep "Grafana"
```

### Méthode Manuelle

1. Accéder à l'URL.
2. Se connecter.
3. Ouvrir **Dashboards -> Browse**.
4. Vérifier que les dashboards provisionnés sont présents et ne retournent pas d'erreur de datasource.

Provisioned dashboards are **not** found in Grafana's dashboard-import/catalog suggestions. They appear under **Dashboards -> Browse**.

## Security / version dashboards

### Trivy Operator Security Overview

UID: `trivy-operator-security`

High-level security posture:

- Critical / High / Medium / Low vulnerability counts.
- Unique Critical/High CVEs.
- Affected workloads.
- Fixable Critical/High findings.
- Exposed secrets and configuration-audit summaries.
- Top affected workloads.

Severity stats and workload entries link to the detailed explorer with relevant variables preselected.

### Trivy Vulnerability Explorer

UID: `trivy-vulnerability-explorer`

Detailed remediation view with filters for namespace, severity, workload, image, fix availability, CVE regex and package regex.

The table is ordered so the most useful remediation fields are visible first:

```text
Severity -> CVE -> Score -> Namespace -> Workload -> Package -> Installed -> Fixed -> ...
```

CVE values link to NVD.

### Container & Image Versions

UID: `container-image-versions`

Combines:

1. Running container/image/digest information from kube-state-metrics.
2. Current vs Renovate-resolved image targets from `vixens_renovate_*` metrics.

Important filters:

- Namespace.
- `Version à jour = Yes / No / Unknown`.
- Application.
- Update type.

Important summary stats:

- Running containers.
- Updates available.
- Major updates available.
- Age/freshness of the last Renovate report.

The GitOps file column links directly to GitHub.

## Dashboard provisioning design

Dashboard JSON is stored in ConfigMaps under:

```text
apps/02-monitoring/grafana/base/dashboards/
```

Each active dashboard ConfigMap has:

```yaml
metadata:
  labels:
    grafana_dashboard: "1"
```

The Grafana dashboard sidecar writes these JSON documents into `/tmp/dashboards` in the Grafana pod, where the Grafana file provisioner loads them.

### Verify sidecar materialization

```bash
kubectl -n monitoring exec deploy/grafana \
  -c grafana-sc-dashboard -- \
  sh -c 'ls -lah /tmp/dashboards && find /tmp/dashboards -maxdepth 1 -type f -print'
```

For the security dashboards, expect files such as:

```text
/tmp/dashboards/trivy.json
/tmp/dashboards/trivy-vulnerability-explorer.json
/tmp/dashboards/container-image-versions.json
```

### Verify Grafana actually registered a dashboard

Use the Grafana API rather than relying only on the UI:

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

A registered dashboard returns its `uid`, title and `/d/<uid>/...` URL.

## Immutable dashboard ConfigMaps

Dashboard ConfigMaps are intentionally `immutable: true`.

Because Kubernetes cannot patch `.data` on an immutable ConfigMap, dashboard revisions use a **new ConfigMap name** (`-v2`, `-v3`, etc.) when JSON changes. ArgoCD creates the replacement and prunes the old ConfigMap.

Do not simply modify an immutable dashboard ConfigMap in place unless the application is explicitly configured for replacement semantics.

The dashboard UID inside the JSON remains stable even when the ConfigMap name changes, so Grafana links/bookmarks remain valid.

## Placeholder dashboard hygiene

The repository may contain placeholder dashboard files for future integrations. A placeholder such as:

```json
{}
```

is **not valid as a Grafana dashboard** because it has no title.

Traefik, Home Assistant and PostgreSQL placeholder ConfigMaps are therefore kept in-repo but are not included in Grafana's Kustomize resources until they contain valid dashboard JSON.

This avoids repeated Grafana log errors such as:

```text
failed to load dashboard ... Dashboard title cannot be empty
```

## Troubleshooting

### ConfigMap exists, dashboard absent

Check the chain in order:

```text
GitOps desired state
  -> ConfigMap exists
  -> sidecar writes JSON to /tmp/dashboards
  -> Grafana provisioner loads JSON
  -> Grafana API returns dashboard
```

Commands:

```bash
kubectl -n monitoring get cm | grep grafana-dashboard

kubectl -n monitoring logs deploy/grafana \
  -c grafana-sc-dashboard --since=30m

kubectl -n monitoring logs deploy/grafana \
  -c grafana --since=30m |
grep -Ei 'dashboard|provision|error|warn'
```

If the JSON file exists in `/tmp/dashboards` and `/api/search` returns the dashboard, provisioning is working even if it was being searched for in the wrong UI section.

### Multi-source ArgoCD revision displays `<none>`

The Grafana ArgoCD Application is multi-source. Therefore:

```text
.status.sync.revision
```

may be `<none>` even while the app is Synced/Healthy.

Use `.status.sync.revisions[]`:

```bash
kubectl -n argocd get application grafana -o json |
jq -r '.status.sync.revisions[]?'
```

### Mutable `prod-stable` cache

After production promotion, ArgoCD may temporarily retain the previous resolution of the mutable `prod-stable` tag.

If Git is promoted but the new dashboard ConfigMaps are absent:

```bash
kubectl -n argocd annotate application grafana \
  argocd.argoproj.io/refresh=hard \
  --overwrite
```

Then wait for sync/self-heal and verify the new ConfigMaps.

## Notes Techniques

- **Namespace:** `monitoring`.
- **Datasource:** VictoriaMetrics/Prometheus-compatible metrics are used by the security/version dashboards.
- **Other dependencies:** Loki, Infisical admin secret.
- **Deployment:** Helm chart managed by ArgoCD.
- **Dashboard source of truth:** Git ConfigMaps, not UI edits.

## Change history (2026-09)

- **#3367**: first Vixens-native Trivy overview.
- **#3368**: detailed vulnerability metric/table support.
- **#3370**: split Overview and Vulnerability Explorer; add Container & Image Versions.
- **#3371**: UX/filter/drill-down/freshness pass and removal of invalid placeholders from provisioning.

See also `docs/guides/security-observability.md`.

---

> ⚠️ **HIBERNATION DEV**
> This application may be disabled in the dev environment to save resources. Check the current ArgoCD dev overlay before using dev for dashboard testing.
