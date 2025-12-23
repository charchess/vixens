# Loki

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v3.0.0  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://loki.[env].truxonline.com (API)

### Méthode Automatique (Curl)
```bash
curl -I -k https://loki.dev.truxonline.com/ready
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Vérifier la source de données Loki dans Grafana.
2. Explorer les logs dans Grafana.

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :**
    - `Synology-CSI` (Stockage logs via PVC `synelia-iscsi-retain`)
- **Particularités :** Déployé via Manifestes (StatefulSet). Mode monolithique (Single Binary). Stockage sur disque persistent.
