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
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://hubble.dev.truxonline.com
# Attendu: HTTP 301/302/308 (Location: https://...)

# 2. Vérifier l'accès HTTPS et le contenu
curl -L -k https://hubble.dev.truxonline.com | grep -o "<title>Hubble UI</title>"
# Attendu: <title>Hubble UI</title>
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que la carte des services (Service Map) s'affiche.
3. **CRITIQUE :** Vérifier qu'il n'y a pas de bandeau rouge "Cannot connect to backend" ou "Reconnecting..." en haut de page. Cela indiquerait un problème de communication avec `hubble-relay`.

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :**
    - `Cilium` (Hubble Relay activé)
- **Particularités :** Interface graphique pour l'observabilité réseau Cilium. Se connecte au service `hubble-relay` dans `kube-system` sur le port 80 (gRPC-web).