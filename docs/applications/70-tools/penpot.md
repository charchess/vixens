# Penpot

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [ ]   | latest  |
| Prod          | [x]     | [x]       | [ ]   | latest  |

## Validation
**URL :** https://penpot.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://penpot.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://penpot.dev.truxonline.com | grep "Penpot"
```

## Notes Techniques
- **Namespace :** `tools`
- **Base de données :** `postgresql-shared` (DB: `penpot`, User: `penpot`)
- **Cache :** `redis-shared` (DB: 1)
- **Stockage Assets :** S3 (configuré via env vars)
- **Composants :**
    - `penpot-backend` (API)
    - `penpot-frontend` (UI)
    - `penpot-exporter` (Export PDF/Image)

## Configuration Secrets (Infisical)
Chemin : `/apps/70-tools/penpot`
Clés requises :
- `POSTGRES_PASSWORD`
- `PENPOT_SECRET_KEY`
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_DEFAULT_REGION`
- `PENPOT_ASSETS_STORAGE_BUCKET`
