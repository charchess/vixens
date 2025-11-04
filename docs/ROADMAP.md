# Roadmap Vixens - Infrastructure Kubernetes GitOps

## Vision Globale

Construire une plateforme Kubernetes multi-cluster (dev/test/staging/prod) gérée en GitOps, suivant les best practices cloud-native, avec approche itérative et destruction/reconstruction simplifiée.

---

## Phases du Projet

### 📦 Phase 1 : Infrastructure as Code (Terraform)
**Scope** : Provisioning automatisé des clusters Talos via Terraform

- Configuration nodes Talos (dual-VLAN)
- Bootstrap Kubernetes
- Déploiement CNI (Cilium)
- Déploiement ArgoCD
- Validation end-to-end

**Deliverable** : `terraform apply` → cluster fonctionnel avec ArgoCD

---

### 🚀 Phase 2 : GitOps Core Services
**Scope** : Services infrastructure gérés par ArgoCD

- MetalLB (LoadBalancer)
- Traefik (Ingress Controller)
- cert-manager (TLS/HTTPS)
- Synology CSI Driver (iSCSI storage)
- Authelia (SSO/Authentication)
- Monitoring (Prometheus/Grafana)

**Deliverable** : Stack infrastructure complète en GitOps

---

### 🎯 Phase 3 : Applications & Utilisation (Hors Scope Initial)
**Scope** : Déploiement applications utilisateur

- Media apps (Radarr, Sonarr, Lidarr)
- Downloads (SABnzbd)
- Home automation (Home Assistant)
- Password manager (Vaultwarden)
- NFS storage (bas priorité)

**Note** : Cette phase fera l'objet d'un projet séparé

---

## Approche Sprint

### Principes

1. **Itératif** : Chaque sprint livre une fonctionnalité validable
2. **Incrémental** : Partir d'1 node → 3 nodes → multi-cluster
3. **Destructible** : Clusters dev/test peuvent être reconstruits à tout moment
4. **Validable** : Chaque sprint a des critères d'acceptation clairs

### Stratégie de Validation

**Par Sprint** :
- Tests manuels (commandes documentées)
- Validation terraform (plan = no changes)
- Validation ArgoCD (all apps synced & healthy)

**Promotion entre Environnements** :
```
DEV (branche dev)
  └─> Validation complète
      └─> PR dev → test
          └─> TEST (branche test)
              └─> Validation complète
                  └─> PR test → staging
                      └─> STAGING (branche staging)
                          └─> Validation complète
                              └─> PR staging → main
                                  └─> PROD (branche main)
```

---

## Phase 1 - Détail des Sprints

### Sprint 0 : Préparation (1-2h)
**Objectif** : Structure projet et documentation

**Livrables** :
- ✅ Architecture réseau documentée
- ✅ ADRs (Architecture Decision Records)
- ✅ Structure Git initialisée
- ✅ Projet Archon créé

---

### Sprint 1 : Terraform Module Talos (4-6h)
**Objectif** : Cluster dev à 1 node (obsy) fonctionnel

**Statut** : ✅ Terminé

**Tâches** :
1. Créer module `terraform/modules/talos/`
   - Variables validation (control planes impair)
   - Configuration dual-VLAN
   - Bootstrap automatique
2. Configurer `terraform/environments/dev/` pour 1 node
3. `terraform apply` → node provisionné
4. Valider Kubernetes API accessible
5. Valider node Ready

**Validation** :
```bash
talosctl --nodes 192.168.111.162 version
kubectl --kubeconfig kubeconfig-dev get nodes
# Résultat attendu : 1 node Ready
```

**Definition of Done** :
- [ ] Module Terraform validé (fmt, validate)
- [ ] Cluster 1 node accessible
- [ ] Kubeconfig fonctionnel
- [ ] Documentation module complète

---

### Sprint 2 : Déploiement Cilium (2-3h)
**Objectif** : CNI opérationnel sur cluster 1 node

**Statut** : ✅ Terminé

**Tâches** :
1. Configurer Helm provider Terraform
2. Déployer Cilium via Terraform
   - kube-proxy replacement
   - Hubble enabled
3. Valider pods Cilium running
4. Valider connectivité réseau

**Validation** :
```bash
kubectl get pods -n kube-system -l k8s-app=cilium
cilium status
cilium connectivity test
```

**Definition of Done** :
- [ ] Cilium pods running (DaemonSet sur 1 node)
- [ ] `cilium connectivity test` = success
- [ ] Hubble relay accessible

---

### Sprint 3 : Scale à 3 Nodes Dev (3-4h)
**Objectif** : Cluster HA 3 control planes

**Statut** : ✅ Terminé

**Tâches** :
1. Ajouter onyx, opale dans `terraform.tfvars`
2. `terraform plan` → vérifier 2 nouveaux nodes
3. `terraform apply`
4. Valider 3 nodes Ready
5. Valider etcd quorum (3 membres)
6. Tester cas erreur (variable validation control planes impair)

**Validation** :
```bash
kubectl get nodes
# Résultat attendu : 3 nodes Ready

talosctl --nodes 192.168.111.160 etcd members
# Résultat attendu : 3 membres
```

**Definition of Done** :
- [ ] 3 nodes Ready
- [ ] etcd quorum fonctionnel
- [ ] Cilium distribué sur 3 nodes
- [ ] Validation variable Terraform (nombre impair)

---

### Sprint 4 : ArgoCD Bootstrap (3-4h)
**Objectif** : ArgoCD auto-géré via GitOps

**Statut** : ✅ Terminé

**Tâches** :
1. Déployer ArgoCD via Terraform (Helm chart)
2. Créer structure `argocd/base/` + `overlays/dev/`
3. Créer root-app (App-of-Apps)
4. Commit + push branche `dev`
5. Appliquer root-app manuellement
6. Valider ArgoCD self-managed

**Validation** :
```bash
kubectl get pods -n argocd
argocd app list
# Résultat attendu : argocd app synced & healthy
```

**Definition of Done** :
- [ ] ArgoCD UI accessible
- [ ] Root app sync automatique
- [ ] ArgoCD se gère lui-même via GitOps

---

## Phase 2 - Détail des Sprints

### Sprint 5 : Cilium L2 LoadBalancer (2-3h)
**Objectif** : LoadBalancer opérationnel via Cilium L2 Announcements

**Statut** : ✅ Terminé

**Tâches** :
1. Créer `apps/metallb/base/` + `overlays/dev/`
2. Définir IPAddressPool (VLAN 208)
   - Pool assigned : .70-.79
   - Pool auto : .80-.89
3. Commit + push dev
4. Valider ArgoCD sync MetalLB

**Validation** :
```bash
kubectl get ipaddresspool -n metallb-system
kubectl get svc -n metallb-system
```

**Definition of Done** :
- [ ] MetalLB pods running
- [ ] IPAddressPool configuré
- [ ] LoadBalancer service obtient IP du pool

---

### Sprint 6 : Traefik (3-4h)
**Objectif** : Ingress controller exposé

**Statut** : [ ] En cours

**Tâches** :
1. Créer `apps/traefik/base/` + `overlays/dev/`
2. Traefik LoadBalancer avec IP fixe (192.168.208.70)
3. Déployer app test (whoami) avec Ingress
4. Valider accès HTTP externe

**Validation** :
```bash
kubectl get svc -n traefik traefik
# EXTERNAL-IP = 192.168.208.70

curl http://whoami.dev.local
# Résultat attendu : réponse whoami
```

**Definition of Done** :
- [ ] Traefik accessible sur 192.168.208.70
- [ ] Ingress whoami fonctionnel
- [ ] HTTP routing validé

---

### Sprint 7 : cert-manager (Self-Signed Dev) (2-3h)
**Objectif** : TLS automatique en dev

**Tâches** :
1. Créer `apps/cert-manager/base/`
2. ClusterIssuer `selfsigned` pour dev
3. Annoter Ingress whoami avec cert-manager
4. Valider certificat généré

**Validation** :
```bash
kubectl get certificate -n default
# Résultat attendu : whoami-tls Ready

curl https://whoami.dev.local -k
# Résultat attendu : HTTPS avec self-signed cert
```

**Definition of Done** :
- [ ] cert-manager pods running
- [ ] ClusterIssuer selfsigned créé
- [ ] Certificat auto-généré pour Ingress

---

### Sprint 8 : Synology CSI Driver (4-5h)
**Objectif** : Stockage iSCSI dynamique

**Tâches** :
1. Créer Secret Synology (DSM credentials)
2. Déployer Synology CSI via ArgoCD
3. Créer StorageClass `synelia-iscsi`
4. Tester PVC (MariaDB test)

**Validation** :
```bash
kubectl get sc synelia-iscsi
kubectl get pvc -n test
# Résultat attendu : PVC bound

kubectl exec -it mariadb-0 -n test -- df -h /var/lib/mysql
# Résultat attendu : iSCSI volume monté
```

**Definition of Done** :
- [ ] Synology CSI pods running
- [ ] StorageClass créé
- [ ] PVC dynamique fonctionnel
- [ ] Test DB avec données persistantes

---

### Sprint 9 : Réplication Cluster Test (6-8h)
**Objectif** : Valider Terraform + Kustomize sur 2e cluster

**Statut** : [ ] En cours

**Tâches** :
1. Créer `terraform/environments/test/`
2. Variables : carny, celesty, citrine (VLAN 209)
3. `terraform apply` → cluster test complet
4. Créer branche `test`
5. Créer overlays test pour toutes les apps
6. Valider ArgoCD test sync depuis branche `test`

**Validation** :
```bash
kubectl --kubeconfig kubeconfig-test get nodes
# Résultat attendu : 3 nodes Ready (test)

argocd app list --kubeconfig kubeconfig-test
# Résultat attendu : toutes apps synced
```

**Definition of Done** :
- [ ] Cluster test opérationnel (3 nodes)
- [ ] Toutes apps Phase 2 déployées sur test
- [ ] Kustomize overlays fonctionnels
- [ ] GitOps test autonome

---

### Sprint 10 : Authelia (4-5h)
**Objectif** : SSO devant Traefik

**Tâches** :
1. Créer `apps/authelia/base/` + overlays
2. Backend flatfile (users.yaml)
3. Middleware Traefik pour Authelia
4. Protéger ArgoCD + whoami avec auth

**Validation** :
```bash
curl https://argocd.dev.local
# Résultat attendu : redirect vers Authelia login
```

**Definition of Done** :
- [ ] Authelia pods running
- [ ] Login page accessible
- [ ] Middleware Traefik configuré
- [ ] Services protégés par auth

---

### Sprint 11 : Monitoring (Prometheus/Grafana) (4-6h)
**Objectif** : Observabilité cluster

**Tâches** :
1. Déployer kube-prometheus-stack
2. Configurer ServiceMonitors (Cilium, Traefik, ArgoCD)
3. Dashboards Grafana
4. Alertes basiques (node down, pod crash)

**Validation** :
```bash
kubectl get prometheus -n monitoring
kubectl get grafana -n monitoring

# Accès Grafana UI → voir métriques cluster
```

**Definition of Done** :
- [ ] Prometheus scrape targets OK
- [ ] Grafana dashboards visibles
- [ ] Alertes configurées

---

## Phase 3 - Applications (Projet Séparé)

**Sprints futurs** :
- Sprint 12 : NFS Storage (PV statiques)
- Sprint 13 : Media apps (Radarr, Sonarr)
- Sprint 14 : Downloads (SABnzbd)
- Sprint 15 : Vaultwarden
- Sprint 16 : Home Assistant

---

## Timeline Estimée

| Phase      | Sprints       | Temps Estimé | Status     |
|------------|---------------|--------------|------------|
| Phase 0    | Sprint 0      | 1-2h         | ✅ En cours|
| Phase 1    | Sprints 1-4   | 12-17h       | ⏳ Pending |
| Phase 2    | Sprints 5-11  | 23-32h       | ⏳ Pending |
| Phase 3    | Sprints 12+   | TBD          | 📅 Future  |

**Total Phase 1+2** : ~35-50h (répartis sur plusieurs semaines)

---

## Dépendances entre Sprints

```
Sprint 1 (Terraform 1 node)
  └─> Sprint 2 (Cilium)
      └─> Sprint 3 (3 nodes)
          └─> Sprint 4 (ArgoCD)
              ├─> Sprint 5 (MetalLB)
              │   └─> Sprint 6 (Traefik)
              │       └─> Sprint 7 (cert-manager)
              └─> Sprint 8 (Synology CSI)
              └─> Sprint 9 (Cluster Test)
                  └─> Sprint 10 (Authelia)
                      └─> Sprint 11 (Monitoring)
```

---

## Critères de Succès Globaux

### Phase 1 (Infrastructure as Code)
- [x] Cluster dev créé via `terraform apply`
- [ ] Cluster destructible et recréable en < 30min
- [ ] 3 control planes HA fonctionnels
- [ ] CNI (Cilium) opérationnel
- [x] ArgoCD auto-géré

### Phase 2 (GitOps Services)
- [ ] Tous les services gérés via Git
- [ ] `git push` = déploiement automatique
- [ ] HTTPS fonctionnel (self-signed dev, Let's Encrypt prod)
- [ ] Storage dynamique (iSCSI)
- [ ] Authelia SSO protège tous les services
- [ ] Monitoring complet (Prometheus/Grafana)
- [ ] Cluster test réplique dev avec succès (en cours)

---

## Risques & Mitigations

| Risque | Impact | Probabilité | Mitigation |
|--------|--------|-------------|------------|
| Complexité Terraform Talos | Bloque Phase 1 | Moyenne | Tests unitaires, validation progressive |
| Réseau VLAN mal configuré | Bloque accès services | Faible | Documentation détaillée, tests connectivité |
| ArgoCD sync issues | Bloque GitOps | Faible | Validation dry-run, logs détaillés |
| Storage iSCSI performance | Dégrade apps | Faible | Tests bench, tuning NAS |
| Learning curve Cilium | Ralentit debugging | Moyenne | Documentation Hubble, Slack community |

---

## Next Steps

1. ✅ Review documentation architecture (en cours)
2. ⏳ Créer sprints/tâches dans Archon (après validation)
3. ⏳ Démarrer Sprint 1 (Terraform module)

---

**Version** : 1.0
**Date** : 2025-10-30
**Auteur** : Infrastructure Team
