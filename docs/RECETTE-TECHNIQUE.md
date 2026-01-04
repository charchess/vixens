# Recette Technique

Validation des aspects techniques (infra, logs, perfs).

| Application | Composant | État | Date | Métrique/Observation |
| :--- | :--- | :--- | :--- | :--- |
| **hydrus-client** | Pod Status | ✅ Running | 2026-01-04 | 2/2 containers ready (App + Litestream sidecar) |
| **hydrus-client** | Ressources | ✅ OK | 2026-01-04 | CPU: 2, RAM: 4Gi |
| **litestream** | Logs Réplication | ✅ OK | 2026-01-04 | 4 bases suivies (client, mappings, master, caches) |
| **litestream** | Restauration | ✅ OK | 2026-01-04 | Multi-InitContainers validés pour réassemblage |
| **litestream** | Secrets | ✅ Synced | 2026-01-04 | Path /shared/litestream utilisé |
| **litestream** | Config | ✅ OK | 2026-01-04 | ConfigMap unifié pour multi-DB |
| **traefik** | Routing | ✅ OK | 2026-01-04 | Middleware qualifié par namespace |
