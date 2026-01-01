# Stirling-PDF

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 1.3.2   |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [x]     | [ ]       | [ ]   | 1.3.2   |

## Validation
**URL :** https://pdf.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://pdf.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://pdf.dev.truxonline.com | grep "Stirling PDF"
# Attendu: Présence de "Stirling PDF"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que la page d'accueil avec les différents outils PDF s'affiche.
3. Tester un outil simple (ex: "Get ALL Info on PDF") avec un fichier PDF de test.

## Notes Techniques
- **Namespace :** `tools`
- **Chart Helm :** `stirling-pdf/stirling-pdf-chart`
- **Particularités :**
  - Application Java (Spring Boot) qui peut être lente au démarrage.
  - Sondes de liveness/readiness configurées avec des délais augmentés (60s/30s) pour éviter les boucles de redémarrage.
  - Déployé initialement dans le mauvais namespace (`stirling-pdf`), corrigé pour être dans `tools`.
- **Ressources :**
  - Requests: 100m CPU / 256Mi RAM
  - Limits: 1000m CPU / 1Gi RAM

---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
