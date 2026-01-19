# Mail Gateway

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | External|
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Service Interne)

### Méthode Automatique (Command Line)
```bash
# Vérifier la configuration des endpoints
kubectl get endpoints mail-gateway-svc -n mail-gateway
# Attendu: IP 192.168.111.69 configurée sur le port 5000
```

### Méthode Manuelle
1. Depuis un pod de test (netshoot), tenter une connexion telnet : `nc -v mail-gateway-svc.mail-gateway 5000`.
2. Envoyer un mail de test depuis une application (ex: Vaultwarden, Alertmanager) et vérifier la réception.

## Notes Techniques
- **Namespace :** `mail-gateway`
- **Dépendances :** Synology MailPlus (Service Externe)
- **Particularités :** Service "Headless" (via Endpoints manuels) pointant vers le serveur mail externe (Synology NAS). Permet aux applications du cluster d'envoyer des mails.

