# Recette Fonctionnelle

Validation des fonctionnalités utilisateur après déploiement.

| Application | Fonctionnalité | Résultat | Date | Commentaire |
| :--- | :--- | :--- | :--- | :--- |
| **hydrus-client** | Accès Web UI (noVNC) | ✅ OK | 2026-01-04 | Accessible via /vnc.html |
| **litestream** | Réplication S3 | ✅ OK | 2026-01-04 | 4 bases répliquées (client, mappings, master, caches) |
| **litestream** | Résilience (Chaos Test) | ✅ OK | 2026-01-04 | Restauration intégrale après suppression PV/PVC |
| **mariadb-shared** | Accès Root (CLI) | ✅ OK | 2026-01-07 | Accès validé avec MARIADB_ROOT_PASSWORD |
| **postgresql-shared** | Provisioning User | ✅ OK | 2026-01-07 | Utilisateurs créés via Managed Roles (docspell validé) |
| **authentik** | Accès UI SSO | ✅ OK | 2026-01-07 | Dashboard Admin accessible via authentik.truxonline.com |
| **vaultwarden** | Accès Vault | ✅ OK | 2026-01-07 | Interface Bitwarden opérationnelle (PVC réparé) |
| **external-dns** | Synchro Gandi | ✅ OK | 2026-01-07 | Enregistrements DNS créés sur Gandi (Secret restauré) |
| **Cluster Global** | **Disponibilité** | ✅ **100%** | 2026-01-07 | Tous les services critiques sont Running et Gold |
