# Mealie

Mealie est un gestionnaire de recettes auto-hébergé.

## Infrastructure

- **Namespace**: `mealie`
- **Port**: `9000` (HTTP)
- **Image**: `ghcr.io/mealie-recipes/mealie:v1.12.0`
- **Storage**: 1Gi PVC via `synelia-iscsi-retain` (RWO)
- **Strategy**: `Recreate`

## Configuration (Infisical)

Les secrets sont gérés dans Infisical sous le chemin `/apps/10-home/mealie`.

| Clé | Description | Valeur conseillée |
| --- | --- | --- |
| `ALLOW_SIGNUP` | Autoriser l'inscription | `true` ou `false` |
| `MEALIE_SECRET_KEY` | Clé secrète de l'application | Chaîne aléatoire |
| `BASE_URL` | URL de base de l'application | `https://mealie.truxonline.com` |

## Ingress & Accès

- **Dev**: `https://mealie.dev.truxonline.com` (LetsEncrypt Staging)
- **Prod**: `https://mealie.truxonline.com` (LetsEncrypt Prod)

## Validation

### Technique
- Vérifier que le pod est `Running`: `kubectl get pods -n mealie`
- Vérifier la synchro des secrets: `kubectl get infisicalsecret -n mealie`
- Vérifier le certificat: `kubectl get certificate -n mealie`

### Fonctionnelle
- Accès à l'interface web via l'URL configurée.
- Connexion/Inscription fonctionnelle.
- Création d'une recette test.
