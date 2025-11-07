# Sprint 6 - cert-manager + TLS Implementation Plan

**Objectif :** Sécuriser tous les services web avec HTTPS et certificats automatiques Let's Encrypt

**Date :** 2025-11-07
**Environnements :** dev, test
**Dépendances :** Traefik fonctionnel (Sprint 5 ✅)

---

## 📋 Vue d'ensemble

### Services à sécuriser
- ✅ Traefik Dashboard (traefik.{env}.truxonline.com)
- ✅ ArgoCD UI (argocd.{env}.truxonline.com)
- ✅ Whoami App (whoami.{env}.truxonline.com)

### Architecture cible
```
Internet (HTTPS)
    ↓
Traefik Ingress (TLS termination)
    ↓ (HTTP)
Services internes
```

---

## 🏗️ Architecture technique

### Composants

**1. cert-manager** (v1.14.x)
- Contrôleur Kubernetes pour gérer les certificats
- Renouvellement automatique (30 jours avant expiration)
- Support Let's Encrypt ACME protocol

**2. ClusterIssuer** (2 isseurs)
- `letsencrypt-staging` : Tests (rate limit élevé)
- `letsencrypt-prod` : Production (rate limit strict : 50 certs/semaine/domain)

**3. Certificate CRDs**
- Création automatique via annotations Ingress
- Stockage dans Secrets Kubernetes

### Flux ACME HTTP-01 Challenge
```
1. Ingress créé avec annotation cert-manager
2. cert-manager détecte et crée Certificate CRD
3. Let's Encrypt envoie HTTP challenge
4. Traefik route /.well-known/acme-challenge/ vers cert-manager
5. Validation OK → Certificat émis
6. cert-manager stocke cert dans Secret
7. Traefik utilise le Secret pour TLS
```

---

## 📦 Structure des fichiers

```
apps/cert-manager/
├── base/
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── cluster-issuer-staging.yaml
│   └── cluster-issuer-prod.yaml
└── overlays/
    ├── dev/
    │   ├── kustomization.yaml
    │   └── cluster-issuer-patch.yaml  # Email contact Let's Encrypt
    └── test/
        ├── kustomization.yaml
        └── cluster-issuer-patch.yaml

argocd/overlays/{dev,test}/
├── cert-manager-app.yaml              # ArgoCD Application

apps/traefik-dashboard/base/
├── ingress.yaml                       # + annotations TLS
└── certificate.yaml                   # Certificate CRD (optionnel)

apps/argocd/overlays/{dev,test}/
└── ingress.yaml                       # + annotations TLS

apps/whoami/base/
└── ingress.yaml                       # + annotations TLS
```

---

## 🔧 Étapes d'implémentation

### Phase 1 : Installation cert-manager (Dev)

**Tâche 1.1 : Créer la structure cert-manager**
```bash
mkdir -p apps/cert-manager/base
mkdir -p apps/cert-manager/overlays/{dev,test}
```

**Tâche 1.2 : Déployer cert-manager via Helm**
- Chart Helm officiel : `jetstack/cert-manager`
- Version : v1.14.4
- CRDs : Installées automatiquement
- Configuration :
  - Namespace : `cert-manager`
  - Replicas : 1 (suffisant pour homelab)
  - Tolerations : control-plane

**Tâche 1.3 : Créer ClusterIssuer staging**
```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: admin@truxonline.com  # À adapter
    privateKeySecretRef:
      name: letsencrypt-staging
    solvers:
    - http01:
        ingress:
          class: traefik
```

**Tâche 1.4 : Tester avec un Ingress**
- Modifier whoami Ingress pour activer TLS staging
- Vérifier émission du certificat
- Valider accès HTTPS (certificat staging non-trusted = normal)

### Phase 2 : Activer TLS sur tous les services (Dev)

**Tâche 2.1 : Traefik Dashboard**
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-staging
    traefik.ingress.kubernetes.io/router.entrypoints: websecure
spec:
  tls:
  - hosts:
    - traefik.dev.truxonline.com
    secretName: traefik-dashboard-tls
```

**Tâche 2.2 : ArgoCD UI**
- Même pattern que Traefik

**Tâche 2.3 : Whoami App**
- Même pattern

**Tâche 2.4 : Configurer redirect HTTP → HTTPS**
- Middleware Traefik pour redirection automatique
- Ou annotation Ingress

### Phase 3 : Passage en production (Dev)

**Tâche 3.1 : Créer ClusterIssuer prod**
```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@truxonline.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: traefik
```

**Tâche 3.2 : Basculer tous les Ingress vers prod**
- Changer annotation : `letsencrypt-staging` → `letsencrypt-prod`
- Supprimer anciens Secrets staging
- Vérifier nouveaux certificats émis

**Tâche 3.3 : Valider certificats**
```bash
openssl s_client -connect traefik.dev.truxonline.com:443 -servername traefik.dev.truxonline.com
```

### Phase 4 : Extension à Test

**Tâche 4.1 : Créer overlay test**
- Copier structure dev → test
- Adapter hostnames (*.test.truxonline.com)

**Tâche 4.2 : Commit dev → PR → test**
- Suivre workflow habituel

**Tâche 4.3 : Valider test**
- Vérifier tous les certificats émis
- Tester accès HTTPS

---

## ⚠️ Considérations importantes

### Rate Limits Let's Encrypt

**Staging (recommandé pour tests) :**
- Pas de rate limit strict
- Certificats non-trusted (normal)

**Production :**
- 50 certificats/semaine/registered domain
- 5 duplicate certificates/semaine/domain
- ⚠️ **IMPORTANT** : Tester en staging avant prod !

### DNS et domaines

**Prérequis :**
- DNS public pointant vers LoadBalancer externe
- Ou DNS interne si Let's Encrypt peut atteindre les services

**Validation :**
```bash
# Vérifier résolution DNS
nslookup traefik.dev.truxonline.com

# Vérifier accessibilité HTTP (challenge)
curl -v http://traefik.dev.truxonline.com/.well-known/acme-challenge/test
```

### Traefik Configuration

**EntryPoints requis :**
```yaml
ports:
  web:
    port: 80
    expose: true
  websecure:
    port: 443
    expose: true
    tls:
      enabled: true
```

**Middleware redirect HTTP → HTTPS :**
```yaml
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: redirect-https
spec:
  redirectScheme:
    scheme: https
    permanent: true
```

---

## 📊 Métriques de succès

- [ ] cert-manager déployé et healthy dans dev
- [ ] ClusterIssuer staging fonctionnel
- [ ] 3 certificats staging émis (traefik, argocd, whoami)
- [ ] ClusterIssuer prod fonctionnel
- [ ] 3 certificats prod valides (trusted)
- [ ] Tous les services accessibles en HTTPS
- [ ] Redirect HTTP → HTTPS fonctionnel
- [ ] Renouvellement automatique testé (< 30 jours)
- [ ] Extension à test réussie

---

## 🚀 Ordre d'exécution recommandé

1. Installer cert-manager (dev)
2. Créer ClusterIssuer staging
3. Tester avec 1 service (whoami)
4. Valider certificat staging émis
5. Étendre aux 3 services
6. Créer ClusterIssuer prod
7. Basculer les 3 services en prod
8. Valider certificats prod
9. Activer redirect HTTP → HTTPS
10. Créer overlay test
11. PR dev → test
12. Valider test

---

## 📚 Ressources

- [cert-manager docs](https://cert-manager.io/docs/)
- [Let's Encrypt rate limits](https://letsencrypt.org/docs/rate-limits/)
- [Traefik + cert-manager](https://doc.traefik.io/traefik/https/acme/)
- [HTTP-01 Challenge](https://letsencrypt.org/docs/challenge-types/)

---

## 🔄 Rollback plan

Si problème :
1. Supprimer annotation `cert-manager.io/cluster-issuer`
2. Supprimer section `tls:` dans Ingress
3. Supprimer Secrets certificats
4. Services reviennent en HTTP

**Checkpoint git avant Sprint 6 :** `88e80f1`
