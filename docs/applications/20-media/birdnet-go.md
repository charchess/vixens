# BirdNET-Go

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [x]     | [x]       | [x]   | nightly-20251223 |

## Validation
**URL :** https://birdnet.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://birdnet.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://birdnet.dev.truxonline.com | grep "BirdNET-Go"
# Attendu: Présence de "BirdNET-Go"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'analyse audio fonctionne (si flux audio configuré) et que le spectrogramme s'affiche.

## Notes Techniques
- **Namespace :** `birdnet-go`
- **Dépendances :** Flux audio RTSP (frigate.media)
- **Configuration :**
  - Locale: Configurée via l'interface web (persistée dans config.yaml)
  - Coordonnées: latitude=46.923, longitude=-0.353
- **Stockage :**
  - `/config` : PVC `birdnet-go-config` (iSCSI)
  - `/data` : PVC `birdnet-go-data` (iSCSI)
  - `/data/clips` : NFS `192.168.111.69:/volume3/Internal/birdnet/clips`
- **PriorityClass :** `vixens-medium` (antipréemption)
- **Particularités :** Analyse et identification de chants d'oiseaux en temps réel
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
