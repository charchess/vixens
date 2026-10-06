# Radarr

## Tier de Maturity

| Tier | Statut | Date |
|------|--------|------|
| 🥉 Bronze | ✅ | 2026-02-24 |
| 🥈 Silver | ✅ | 2026-02-24 |
| 🥇 Gold | ⏳ | - |

## Informations de Déploiement
# Radarr

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Prod          | [x]     | [x]       | [x]   | -       |

## Validation
**URL :** https://radarr.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://radarr.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://radarr.dev.truxonline.com | grep "Radarr"
# Attendu: Présence de "Radarr"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface se charge.
3. Vérifier les connexions (Prowlarr, Download Client).

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :**
    - NFS Storage
    - `Prowlarr`
    - Download Clients
- **Particularités :** Gestionnaire de films.
---
> ⚠️ **HIBERNATION DEV**
> L'environnement `dev` peut être hiberné pour économiser les ressources ; vérifiez l'overlay dev courant avant de conclure qu'il est actif ou inactif.
> Pour réactiver durablement l'application, suivez `docs/procedures/dev-hibernation.md` via une branche/PR ; ne décommentez pas directement l'Application ArgoCD comme mécanisme de réveil.
