# Recette Technique

Validation des aspects techniques (infra, logs, perfs).

| Application | Composant | État | Date | Métrique/Observation |
| :--- | :--- | :--- | :--- | :--- |
| **hydrus-client** | Pod Status | ✅ Running | 2026-01-08 | 2/2 containers ready. Priority: vixens-medium |
| **hydrus-client** | Gold Standard | ✅ Compliant | 2026-01-08 | Integrity check (sqlite3) + Conditional restore active |
| **hydrus-client** | Storage (ISCSI) | ✅ Resized | 2026-01-08 | PVC increased to 40Gi to accommodate large WAL sets |
| **DataAngel** | Backup (16 apps) | ✅ Running | 2026-03-26 | Unified SQLite+FS backup to S3/MinIO. Replaces litestream + config-syncer + rclone |
| **DataAngel** | Sidecar Injection | ✅ OK | 2026-03-26 | Shared component `_shared/components/dataangel/` + per-app patches |
| **DataAngel** | Secrets | ✅ Synced | 2026-03-26 | MinIO credentials via Infisical |
| **traefik** | Routing | ✅ OK | 2026-01-04 | Middleware qualifié par namespace |
| **postgresql-shared** | HA Scale-down | ✅ OK | 2026-01-07 | Retour à 1 instance stable après test HA |
| **redis-shared** | Sécurité Auth | ✅ OK | 2026-01-07 | requirepass activé via Infisical |
| **mariadb-shared** | Déploiement Kustomize | ✅ OK | 2026-01-07 | Migration StatefulSet natif réussie |
| **authentik** | QoS & Storage | ✅ OK | 2026-01-07 | vixens-critical + Recreate strategy validés |
| **docspell** | Stabilité CPU | ✅ OK | 2026-01-07 | Request 200m CPU : Arrêt des CrashLoops |
| **external-dns** | Architecture Secret | ✅ OK | 2026-01-07 | Séparation Secret/Binaire pour contourner cache ArgoCD |
| **vaultwarden** | Probes / Stability | ✅ OK | 2026-01-07 | Liveness assouplie (60s) : Arrêt des CrashLoops |
| **traefik** | Shared Middleware | ✅ OK | 2026-01-09 | Centralisation de redirect-https (namespace traefik) |
| **renovate** | Priority Class | ✅ OK | 2026-01-11 | vixens-medium active on CronJob pods |
| **renovate** | Base Branch | ✅ OK | 2026-01-11 | baseBranches set to main (ADR-017) |
| **ArgoCD** | Resource Conflicts | ✅ Resolved | 2026-01-11 | Shared namespaces centralized, ownership conflicts fixed |
| **mariadb-shared** | App Duplication | ✅ Fixed | 2026-01-11 | Removed duplicate mariadb-shared-config application |
| **mariadb-shared** | Sync Waves | ✅ OK | 2026-01-11 | Internal sync waves (Secret -> StatefulSet) added |
| **postgresql-shared** | User Mapping | ✅ Fixed | 2026-01-13 | Managed roles mapping logic restored for Authentik |
| **authentik** | Redis Credentials | ✅ Fixed | 2026-01-13 | Secret path aligned with shared Redis instance |
| **authentik** | OIDC Blueprints | ✅ Ready | 2026-01-13 | Netbird blueprint mounted in /blueprints/vixens/ |
| **fluent-bit** | Remplacement Promtail | ✅ Running | 2026-03-23 | 5/5 DaemonSet pods, 0 restarts (Promtail avait 600+). Logs → Loki via loki output plugin |
| **fluent-bit** | Position DB | ✅ OK | 2026-03-23 | hostPath /var/lib/fluent-bit pour persistance du curseur de lecture |
| **victoria-metrics** | Remplacement Prometheus | ✅ Running | 2026-03-23 | k8s-stack 0.72.5 : vmsingle (30d retention, 10Gi), vmagent, vmalert, alertmanager |
| **victoria-metrics** | RAM Monitoring | ✅ Réduit | 2026-03-23 | Stack total ~700 Mi (vs ~2.25 GiB avant). Réduction 69% |
| **loki** | Sizing V-medium | ✅ Stabilisé | 2026-03-23 | Upgrade G-medium → V-medium après surcharge Fluent Bit. 0 restarts post-upgrade |
| **loki** | NetworkPolicy | ✅ Corrigé | 2026-03-23 | Ajout namespace monitoring dans ingress (bloquait Fluent Bit) |
| **grafana** | Datasource VictoriaMetrics | ✅ OK | 2026-03-23 | URL mise à jour vers vmsingle:8428. Dashboards fonctionnels |
| **grafana** | Dashboards (8) | ✅ OK | 2026-03-26 | HA, PostgreSQL, Traefik, Trivy, DataAngel, ArgoCD, Velero, Cert-Manager |
| **local-path-provisioner** | StorageClasses | ✅ Running | 2026-03-26 | v0.0.35 : local-path-delete + local-path-retain StorageClasses alongside Synology iSCSI |
| **VPA** | Auto Mode | ✅ OK | 2026-03-26 | 52 containers on V-* profiles (VPA Auto mode). Kyverno auto-detects V-* prefix |
| **Velero** | Schedules (4) | ✅ OK | 2026-03-26 | daily-critical, daily-home, daily-media, weekly-full. 100% namespace coverage |
| **Security** | runAsNonRoot | ✅ OK | 2026-03-26 | Enforced on 11 apps |
| **Security** | preStop hooks | ✅ OK | 2026-03-26 | Configured on 8 critical apps for graceful shutdown |
