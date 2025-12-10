# Plan de Recette Fonctionnelle - Vixens Homelab

Ce document décrit les étapes de validation fonctionnelle à haut niveau pour s'assurer que les services principaux du homelab sont opérationnels après un déploiement ou une modification. L'objectif est de confirmer la disponibilité et l'accès aux interfaces principales sans vérifier en profondeur la configuration technique.

## Prérequis

- Le cluster Kubernetes pour l'environnement cible (ex: `dev`, `test`) est déployé.
- Les enregistrements DNS sont configurés et propagés.
- Vous disposez d'un accès réseau au VLAN des services.

## 1. Validation des Services Web

Pour chaque service listé ci-dessous, ouvrez l'URL correspondante dans un navigateur web et confirmez que la page se charge sans erreur majeure (e.g., erreur 404 ou 503).

| Service | Environnement `dev` | Environnement `test` | Résultat Attendu | Statut (OK/KO) |
| :--- | :--- | :--- | :--- | :--- |
| **ArgoCD** | `https://argocd.dev.truxonline.com` | `https://argocd.test.truxonline.com` | La page de connexion d'ArgoCD s'affiche. | |
| **Traefik Dashboard** | `https://traefik.dev.truxonline.com/dashboard/` | `https://traefik.test.truxonline.com/dashboard/` | Le dashboard Traefik s'affiche avec la liste des routeurs et services. | |
| **Whoami** | `https://whoami.dev.truxonline.com` | `https://whoami.test.truxonline.com` | Une page simple affichant des informations sur la requête et le conteneur s'affiche. | |
| **Home Assistant** | `https://homeassistant.dev.truxonline.com` | `https://homeassistant.test.truxonline.com` | La page de connexion ou le tableau de bord de Home Assistant s'affiche. | |
| **Mail Gateway** | `https://mail.dev.truxonline.com` | `https://mail.test.truxonline.com` | La page de Roundcube ou du service mail s'affiche. | |
| **Linkwarden** | `https://linkwarden.dev.truxonline.com` | `https://linkwarden.test.truxonline.com` | La page de connexion ou le tableau de bord de Linkwarden s'affiche. | |

## 2. Validation de la Sécurité (HTTPS)

Pour chaque URL testée à l'étape 1 :
1. Cliquez sur l'icône de cadenas dans la barre d'adresse du navigateur.
2. Vérifiez que le certificat est émis par **"(STAGING) Let's Encrypt"** (pour `dev`/`test`) ou **"Let's Encrypt"** (pour `prod`).
3. Vérifiez que la connexion est décrite comme "sécurisée".

**Résultat Attendu :** Tous les services sont servis en HTTPS avec un certificat valide.

## 2.1. Validation des Redirections HTTP → HTTPS

Pour chaque service, testez que les requêtes HTTP sont automatiquement redirigées vers HTTPS :

```bash
# Test ArgoCD
curl -I http://argocd.dev.truxonline.com

# Test Traefik
curl -I http://traefik.dev.truxonline.com

# Test Home Assistant
curl -I http://homeassistant.dev.truxonline.com

# Test Mail Gateway
curl -I http://mail.dev.truxonline.com
```

**Résultat Attendu :**
- Code de réponse: `HTTP/1.1 308 Permanent Redirect`
- Header `Location:` pointe vers l'URL HTTPS correspondante
- Exemple: `Location: https://homeassistant.dev.truxonline.com/`

## 3. Validation du "Self-Heal" d'ArgoCD (Test Superficiel)

1. Choisissez une application simple comme `whoami`.
2. Supprimez manuellement l'Ingress de l'application :
   ```bash
   kubectl -n whoami delete ingress whoami-ingress --kubeconfig <chemin-vers-kubeconfig>
   ```
3. Attendez environ 3 minutes (le cycle de réconciliation par défaut d'ArgoCD).
4. Vérifiez que l'Ingress a été automatiquement recréé :
   ```bash
   kubectl -n whoami get ingress --kubeconfig <chemin-vers-kubeconfig>
   ```

**Résultat Attendu :** L'Ingress `whoami-ingress` réapparaît automatiquement, démontrant que le "self-healing" est actif.

## Bilan de la Recette

- **Succès :** Toutes les étapes ci-dessus sont validées avec succès. Le système est considéré comme fonctionnellement opérationnel.
- **Échec :** Au moins une des étapes a échoué. Une investigation plus approfondie via la recette technique est nécessaire.
