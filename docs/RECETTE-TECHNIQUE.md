# Recette Technique

Validation des aspects techniques (infra, logs, perfs).

| Application | Composant | État | Date | Métrique/Observation |
| :--- | :--- | :--- | :--- | :--- |
| **hydrus-client** | Pod Status | ✅ Running | 2026-01-08 | 2/2 containers ready. Priority: vixens-medium |
| **hydrus-client** | Gold Standard | ✅ Compliant | 2026-01-08 | Integrity check (sqlite3) + Conditional restore active |
| **hydrus-client** | Storage (ISCSI) | ✅ Resized | 2026-01-08 | PVC increased to 40Gi to accommodate large WAL sets |
| **litestream** | Restoration | ✅ Robust | 2026-01-08 | Automated cleanup of .tmp files + Cache bypass valid |
| **litestream** | Secrets | ✅ Synced | 2026-01-04 | Path /shared/litestream utilisé |
| **litestream** | Config | ✅ OK | 2026-01-04 | ConfigMap unifié pour multi-DB |
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
