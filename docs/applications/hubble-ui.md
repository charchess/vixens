# Hubble UI

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v0.13.3 |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://hubble.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://hubble.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que la carte des services (Service Map) s'affiche.

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :**
    - `Cilium` (Hubble Relay activé)
- **Particularités :** Interface graphique pour l'observabilité réseau Cilium. Se connecte au service `hubble-relay` dans `kube-system`.
