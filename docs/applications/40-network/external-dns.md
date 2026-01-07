# ExternalDNS

Gestion automatisée des enregistrements DNS pour les Ingress du cluster Kubernetes.

## Architecture

Le déploiement utilise une stratégie de **Split DNS** avec deux instances distinctes d'`external-dns` dans le namespace `networking`.

| Instance | `external-dns-unifi` (Interne) | `external-dns-gandi` (Public) |
| :--- | :--- | :--- |
| **Rôle** | Résolution DNS locale (LAN) | Résolution DNS publique (WAN) |
| **Domaines** | `*.dev.truxonline.com`, `*.truxonline.com` | `*.truxonline.com` |
| **Provider** | Webhook UniFi (Sidecar) | Gandi (API) |
| **Cible par défaut** | IP de l'Ingress (A record) | CNAME (si configuré) |

## Configuration

### Instance UniFi (Interne)
- **Image :** `registry.k8s.io/external-dns/external-dns:v0.14.2`
- **Sidecar :** `ghcr.io/kashalls/external-dns-unifi-webhook:v0.8.0`
- **Secrets (Infisical) :** `/apps/40-network/external-dns/unifi`
- **Comportement :** Scanne tous les Ingress. Si l'entrée n'existe pas dans l'UDM Pro SE, elle est créée. Elle utilise des enregistrements `TXT` pour marquer la propriété.

### Instance Gandi (Public)
- **Image :** `registry.k8s.io/external-dns/external-dns:v0.14.2`
- **Secrets (Infisical) :** `/apps/40-network/external-dns/gandi` (Utilise un Personal Access Token - PAT)
- **Filtrage :** Uniquement les Ingress avec l'annotation `external-dns.alpha.kubernetes.io/public: "true"`.

## Utilisation

### Forcer un CNAME
Pour que l'entrée DNS soit un CNAME pointant vers un hostname spécifique (ex: le domaine racine ou un LoadBalancer) plutôt qu'un record A :
```yaml
annotations:
  external-dns.alpha.kubernetes.io/target: "vixens-dev.truxonline.com"
```

### Exposer publiquement (Gandi)
Pour qu'un Ingress soit synchronisé sur Gandi :
```yaml
annotations:
  external-dns.alpha.kubernetes.io/public: "true"
```

## Validation

### Commandes de vérification
- **Logs UniFi :** `kubectl logs -n networking -l app.kubernetes.io/instance=external-dns-unifi -c external-dns`
- **Logs Webhook :** `kubectl logs -n networking -l app.kubernetes.io/instance=external-dns-unifi -c unifi-webhook`
- **Logs Gandi :** `kubectl logs -n networking -l app.kubernetes.io/instance=external-dns-gandi`

## Notes Techniques (Gold)
- **Priorité :** `vixens-critical`
- **Ressources :** Profil Micro (20m/64Mi)
- **Secrets :** Gérés par Infisical (`external-dns-gandi-secret`) via une Application ArgoCD dédiée pour plus de stabilité.

### Test de résolution
```bash
# Interne (si connecté au LAN)
dig contacts.dev.truxonline.com

# Public (Google DNS)
dig @8.8.8.8 contacts.truxonline.com
```