# Recette Fonctionnelle

Validation des fonctionnalités utilisateur après déploiement.

| Application | Fonctionnalité | Résultat | Date | Commentaire |
| :--- | :--- | :--- | :--- | :--- |
| **hydrus-client** | Accès Web UI (noVNC) | ✅ OK | 2026-01-04 | Accessible via /vnc.html |
| **litestream** | Réplication S3 | ✅ OK | 2026-01-04 | Snapshots et WAL créés dans MinIO |
| **litestream** | Restauration | ✅ OK | 2026-01-04 | Testé via init-container |
