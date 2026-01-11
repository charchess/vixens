# Monitoring Litestream

## Architecture
Litestream expose des métriques au format Prometheus via un sidecar tournant dans le même pod que l'application SQLite.

- **Port par défaut :** 9090
- **Endpoint :** `/metrics`

## Configuration du Scrape
Le monitoring est configuré de manière automatique via le "Service Discovery" de Prometheus basé sur les annotations des pods.

### 1. Configuration Prometheus
Prometheus est configuré (dans `apps/02-monitoring/prometheus/base/values.yaml`) pour rechercher tous les pods avec l'annotation suivante :

```yaml
extraScrapeConfigs: |
  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
    - role: pod
    relabel_configs:
    - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
      action: keep
      regex: true
```

### 2. Activation sur une application
Pour activer le monitoring sur une application utilisant Litestream, ajoutez les annotations suivantes dans le `template.metadata.annotations` de votre déploiement :

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "9090"
  prometheus.io/path: "/metrics"
```

## Métriques Clés à Surveiller
- `litestream_total_wal_bytes` : Nombre total de bytes écrits dans le WAL (Shadow WAL). Utile pour calculer le débit d'écriture.
- `litestream_wal_size` : Taille actuelle du WAL. Si cette taille approche une fraction significative de la DB, un checkpoint est conseillé.
- `litestream_sync_error_count` : Doit toujours être à 0. Indique des échecs de réplication S3.
- `litestream_db_size` : Taille de la base de données principale.

## Requêtes Prometheus Utiles
- **Débit d'écriture (Write Rate) :** `rate(litestream_total_wal_bytes[5m])`
- **Nombre d'opérations S3 :** `sum(rate(litestream_replica_operation_total[1h])) by (operation)`
