# Mail Gateway

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [ ]   | -       |
| Prod          | [x]     | [x]       | [ ]   | -       |

## Description
Service de passerelle mail exposant le service de messagerie Synology MailPlus via Ingress Kubernetes.
Gère les redirections webmail (`/mail`) et l'exposition des services SMTP/IMAP si nécessaire.

## Validation
**URL :** https://mail.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://mail.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier la redirection vers /mail/
curl -I -L -k https://mail.dev.truxonline.com
# Attendu: Redirection vers https://mail.dev.truxonline.com/mail/
```

### Méthode Manuelle
1. Accéder à `https://mail.dev.truxonline.com`
2. Vérifier la redirection automatique vers le webmail (`/mail/`).
3. Vérifier que l'interface de login MailPlus s'affiche.

## Notes Techniques
- **Namespace :** `mail-gateway`
- **Dépendances :** Synology MailPlus Server
- **Particularités :** 
  - Utilise un Middleware Traefik `redirectRegex` pour rediriger la racine `/` vers `/mail/`.
  - Service de type `ExternalName` ou proxy vers l'IP du NAS.
