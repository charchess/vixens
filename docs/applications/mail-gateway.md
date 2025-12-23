# Mail Gateway

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | External|
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Service Interne)

### Méthode Automatique (Command Line)
```bash
kubectl get endpoints mail-gateway-svc -n mail-gateway
# Attendu: IP 192.168.111.69 configurée
```

### Méthode Manuelle
1. Telnet depuis un pod vers le service sur le port 5000 (ou port configuré).

## Notes Techniques
- **Namespace :** `mail-gateway`
- **Dépendances :** Synology MailPlus (Service Externe)
- **Particularités :** Service "Headless" (via Endpoints manuels) pointant vers le serveur mail externe (Synology NAS). Permet aux applications du cluster d'envoyer des mails.
