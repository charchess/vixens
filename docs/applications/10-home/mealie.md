# Mealie

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v3.9.2  |
| Prod          | [x]     | [x]       | [x]   | v3.9.2  |

Mealie est un gestionnaire de recettes auto-hébergé.

## Infrastructure

- **Namespace**: `mealie`
- **Port**: `9000` (HTTP)
- **Image**: `ghcr.io/mealie-recipes/mealie:v3.9.2`
- **Storage**: 1Gi PVC via `synelia-iscsi-retain` (RWO)
- **Strategy**: `Recreate`

## Configuration des secrets

Les valeurs secrètes sont stockées dans OpenBao et projetées dans Kubernetes par External Secrets Operator via `ClusterSecretStore/openbao`.

- **ExternalSecret** : `mealie-secrets-sync`
- **Secret Kubernetes cible** : `mealie-secrets`
- **Chemin dev** : `vixens/dev/apps/10-home/mealie`
- **Chemin prod** : `vixens/prod/apps/10-home/mealie` via l'overlay prod

| Clé | Description | Valeur conseillée |
| --- | --- | --- |
| `ALLOW_SIGNUP` | Autoriser l'inscription | `true` ou `false` |
| `MEALIE_SECRET_KEY` | Clé secrète de l'application | Chaîne aléatoire |
| `BASE_URL` | URL de base de l'application | `https://mealie.truxonline.com` |

Ne pas recréer d'`InfisicalSecret` : l'ancienne intégration Infisical est retirée.

## Ingress & Accès

- **Dev**: `https://mealie.dev.truxonline.com` (LetsEncrypt Staging)
- **Prod**: `https://mealie.truxonline.com` (LetsEncrypt Prod)

## Validation

### Technique
- Vérifier que le pod est `Running`: `kubectl get pods -n mealie`
- Vérifier la synchro des secrets: `kubectl get externalsecret -n mealie`
- Vérifier le certificat: `kubectl get certificate -n mealie`

### Fonctionnelle
- Accès à l'interface web via l'URL configurée.
- Connexion/Inscription fonctionnelle.
- Création d'une recette test.
