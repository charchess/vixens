# Recette Fonctionnelle

Validation des fonctionnalités utilisateur après déploiement.

| Application | Fonctionnalité | Résultat | Date | Commentaire |
| :--- | :--- | :--- | :--- | :--- |
| **hydrus-client** | Accès Web UI (noVNC) | ✅ OK | 2026-01-08 | Accessible via hydrus.truxonline.com |
| **hydrus-client** | Redirection HTTPS | ✅ OK | 2026-01-08 | http -> https redirection verified |
| **DataAngel** | Backup/Restore (16 apps) | ✅ OK | 2026-03-26 | Unified SQLite+FS backup to S3/MinIO. Full restore validated (replaces litestream) |
| **Velero** | Cluster Backup (4 schedules) | ✅ OK | 2026-03-26 | daily-critical, daily-home, daily-media, weekly-full. 100% namespace coverage |
| **authentik** | Accès Web & OIDC | ✅ OK | 2026-01-13 | 302 Redirect verified, blueprint mounted. OIDC ready (Netbird). |
| **Cluster Global** | **Disponibilité** | ✅ **100%** | 2026-01-13 | Authentik operational after DB/Redis credentials fix |
| **grafana** | Dashboards métriques | ✅ OK | 2026-03-23 | Datasource VictoriaMetrics fonctionnel, dashboards existants compatibles (PromQL) |
| **grafana** | Logs (Loki) | ✅ OK | 2026-03-23 | Fluent Bit → Loki pipeline opérationnel. Explore logs fonctionnel |
| **Monitoring Stack** | **Migration complète** | ✅ **OK** | 2026-03-23 | Prometheus→VictoriaMetrics, Promtail→Fluent Bit. PRs #2392-#2398 |
