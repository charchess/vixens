# Contacts

Service de redirection DNS pour l'accès aux contacts (Synology).

## Architecture

Il s'agit d'un service "ExternalName" ou d'un Service/Endpoints pointant vers une IP externe au cluster (Synology NAS).

- **Namespace :** `contacts`
- **Backend :** `192.168.111.69:5000` (DSM Synology)

## Ingress

Le service est exposé via deux Ingress :
1.  **Redirection HTTP :** Redirige le port 80 vers 443.
2.  **Ingress HTTPS :** Gère le TLS via `cert-manager`.

### DNS
L'Ingress utilise `external-dns` pour la gestion automatique du DNS :
- **Interne :** Créé sur l'UDM Pro SE comme un CNAME vers `vixens-dev.truxonline.com` (en dev) ou `truxonline.com` (en prod).
- **Public :** Créé sur Gandi (en prod) grâce à l'annotation `external-dns.alpha.kubernetes.io/public: "true"`.

## Validation

### URL de test
- **Dev :** `https://contacts.dev.truxonline.com`
- **Prod :** `https://contacts.truxonline.com`

### Commande de validation
```bash
curl -I https://contacts.truxonline.com
```

---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
