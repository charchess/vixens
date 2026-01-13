# Recette Fonctionnelle

Validation des fonctionnalités utilisateur après déploiement.

| Application | Fonctionnalité | Résultat | Date | Commentaire |
| :--- | :--- | :--- | :--- | :--- |
| **hydrus-client** | Accès Web UI (noVNC) | ✅ OK | 2026-01-08 | Accessible via hydrus.truxonline.com |
| **hydrus-client** | Redirection HTTPS | ✅ OK | 2026-01-08 | http -> https redirection verified |
| **litestream** | Résilience (Chaos Test) | ✅ OK | 2026-01-08 | Successful full restore from S3 on 40Gi volume |
| **authentik** | Accès Web & OIDC | ✅ OK | 2026-01-13 | 302 Redirect verified, blueprint mounted. OIDC ready (Netbird). |
| **Cluster Global** | **Disponibilité** | ✅ **100%** | 2026-01-13 | Authentik operational after DB/Redis credentials fix |
