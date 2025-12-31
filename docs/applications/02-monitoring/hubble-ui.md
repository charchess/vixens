# Hubble UI

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Prod          | [x]     | [x]       | [x]   | latest  |

## Validation
**URL :** https://hubble.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://hubble.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://hubble.dev.truxonline.com | grep "Hubble"
# Attendu: Présence de "Hubble" dans le titre ou le contenu
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Sélectionner un namespace pour visualiser les flux Cilium.

## Notes Techniques
- **Namespace :** `monitoring`
- **Particularités :** Interface graphique pour Cilium Hubble (observabilité réseau).
