# Headlamp

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v0.39.0 |
| Prod          | [x]     | [x]       | [ ]   | v0.39.0 |

## Validation
**URL :** https://headlamp.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://headlamp.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://headlamp.dev.truxonline.com | grep "Headlamp"
# Attendu: Présence de "Headlamp" dans le titre ou le contenu
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que les ressources du cluster sont visibles.

## Notes Techniques
- **Namespace :** `tools`
- **Particularités :** Interface graphique pour Kubernetes.
