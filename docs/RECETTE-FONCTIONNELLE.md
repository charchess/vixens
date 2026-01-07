# Recette Fonctionnelle

Validation des fonctionnalités utilisateur après déploiement.

| Application | Fonctionnalité | Résultat | Date | Commentaire |
| :--- | :--- | :--- | :--- | :--- |
| **hydrus-client** | Accès Web UI (noVNC) | ✅ OK | 2026-01-04 | Accessible via /vnc.html |
| **litestream** | Réplication S3 | ✅ OK | 2026-01-04 | 4 bases répliquées (client, mappings, master, caches) |
| **litestream** | Résilience (Chaos Test) | ✅ OK | 2026-01-04 | Restauration intégrale après suppression PV/PVC |
| **mariadb-shared** | Accès Root (CLI) | ✅ OK | 2026-01-07 | Accès validé avec MARIADB_ROOT_PASSWORD |
| **postgresql-shared** | Provisioning User | ✅ OK | 2026-01-07 | Utilisateurs créés via Managed Roles (docspell validé) |
