# Application Status Dashboard

**Quick reference for application deployment status across environments.**

Last Updated: 2026-09-17

---

## Global Cluster Status

| Cluster | Nodes | Version | Status |
|---------|-------|---------|--------|
| **Prod** | 5 (peach, pearl, phoebe, poison, powder) | Talos v1.12.4 / K8s v1.34.0 | ✅ Active |
| **Dev** | 1 (daphne — diva/dulce offline intentionnellement) | - | ⚠️ ArgoCD OK, 1 nœud |

| Component | Status | Description |
|-----------|--------|-------------|
| **ArgoCD Apps** | ✅ ~90 Healthy | Quelques apps Progressing (rolling updates post-session) |
| **Kustomize Build** | ✅ PASSING | |
| **CI/CD Pipelines** | ✅ ACTIVE | |
| **Quality Gates** | ✅ ENFORCED | Pre-commit + CI (secrets, YAML style, K8s validation) |

---

## Désactivations demandées — 2026-09-14

- L'app-of-apps production cesse de déclarer `lidarr`, `mongodb-shared`, `mariadb-shared`, `redis-shared` et `postgresql-shared` à la demande du propriétaire.
- La promotion déclenche le prune Argo des workloads et ressources gérées. Les volumes TrueNAS sont conservés selon leurs politiques de rétention ; aucune suppression directe de PV, snapshot ZFS ou objet de sauvegarde n'est effectuée par ce changement.
- L'opérateur `cloudnative-pg` reste déclaré : seule l'instance applicative `postgresql-shared` et ses dépendances de données sont retirées.

---

## Changements en préparation — 2026-09-17

### CoreDNS GitOps runtime

- 🚧 Une Application `coredns` est préparée pour rendre le Corefile de production déclaratif après bootstrap Terraform.
- Le routage reste inchangé : zone `internal.truxonline.com` vers les DC, `truxonline.com` vers l’UDM, et résolution générale vers AdGuard Home via son FQDN Kubernetes.
- Le changement n’est pas encore publié ni promu ; aucune prise de contrôle du ConfigMap live n’a eu lieu.

### Mail — validation DMS privée

- 🚧 Le premier rsync Maildir non destructif depuis fuu a terminé avec succès vers le PVC TrueNAS retenu.
- La prochaine étape GitOps active uniquement un replica DMS privé ; Roundcube demeure à zéro et aucune route SMTP/IMAP publique, DNS/MX, NAT ou cutover source n’est déclaré.
- ⚠️ Le DMS privé est Ready, mais Dovecot v2.3.19.1 expire durant son `auth_bind` DN lookup alors que la même recherche LDAP depuis le Pod réussit sur les deux DC en environ 100 ms. L’essai supporté `blocking = yes` n’a pas modifié ce résultat ; les hooks temporaires de diagnostic sont retirés. Prochaine étape : revue explicite d’un passdb Dovecot à bind UPN direct Active Directory.

---

## Incidents & Changements notables — 2026-09-14

### Data durability: DataAngel retiré des overlays prod

- DataAngel est retiré de 32 overlays de production. Les démarrages applicatifs ne dépendent plus d'une restauration automatique S3 ni de son endpoint d'init.
- La protection durable est assurée par snapshots ZFS TrueNAS avec rétention GFS et réplication `zfs send` vers une cible indépendante, complétée par les backups natifs applicatifs quand disponibles.
- Les PVC/PV retenus, objets S3 historiques et chemins de recovery non-prod sont conservés. Une restauration est désormais une opération explicite, sélectionnant un point ZFS validé.

---

## Incidents & Changements notables — 2026-09-13

### OpenBao / External Secrets

- Le `ClusterSecretStore` canonique est `openbao` et pointe vers OpenBao sur TrueNAS (`nas.truxonline.com:8200`, moteur `kv` v2). Le suffixe historique `-umi` est retiré de tous les manifests ; UMI n'est pas une dépendance de production.
- Les secrets Home Assistant sont synchronisés depuis `kv/vixens/prod/apps/10-home/homeassistant`. Les valeurs ne sont jamais stockées dans Git.
- Le PVC Home Assistant est créé avant le Deployment (wave 5 avant wave 6) sur `truenas-iscsi-xfs-retain`. La remise en service est un démarrage vierge explicitement autorisé ; la restauration applicative provient du backup séparé.
- Les configurations CSI TrueNAS de production sont lues depuis `kv/vixens/prod/apps/01-storage/truenas-csi/{iscsi,nfs}` et utilisent le FQDN canonique `nas.truxonline.com`, sans identifiant UMI.
- Le CSI iSCSI reçoit une egress Cilium minimale : contrôleur vers TrueNAS API/SSH (`443`, `22`) et nœuds CSI vers la cible iSCSI (`3260`). Elle cible uniquement `192.168.200.244/32`, adresse vérifiée du FQDN NAS.


### Incident résolu
| Problème | Root cause | Fix |
|----------|-----------|-----|
| `argocd.truxonline.com` répondait 404 / Traefik n'avait aucun backend | La CiliumNetworkPolicy du namespace `argocd` appliquait le default-deny sans autoriser les Pods Traefik | Ingress explicite Traefik → ArgoCD server dans la CNP GitOps |
| Le routage HTTPS Traefik dépendait globalement de CrowdSec | Le plugin/middleware CrowdSec était injecté sur tout `websecure`, alors que la LAPI est indisponible | Retrait de la dépendance globale ; CrowdSec reste isolé de Traefik |
| Workloads non-core redémarrés après reprise | La reprise globale avait retiré leur composant `prod-hibernate` | Hibernation GitOps réappliquée uniquement à la liste demandée ; socle, stockage et bases restent actifs |
| CrowdSec et VictoriaMetrics restaient actifs malgré `prod-hibernate` | Les workloads sont rendus par une source Helm séparée, non patchable par le composant Kustomize | Valeurs Helm prod dédiées : désactivation explicite des contrôleurs et composants de collecte |
| Applications non-core en dette de PVC/secrets legacy | Les garder dans l'app-of-apps entretenait des synchronisations en échec alors qu'elles étaient volontairement arrêtées | 41 Applications demandées par le propriétaire sont commentées dans l'overlay prod ; ArgoCD les prune au lieu de tenter de les réparer |
| Overlays UMI inclus dans l'app-of-apps prod | Des Applications prod rendaient les valeurs et chemins de l'hôte de développement UMI (`*-umi`) | Remplacées par les Applications/valeurs prod ; les overlays UMI sont explicitement exclus de prod |

## Incidents & Changements notables — 2026-03-28

### Incidents résolus
| Problème | Root cause | Fix |
|----------|-----------|-----|
| loki-0 + g4f ContainerCreating 3h+ | iSCSI lock zombie (phoebe) — processus iscsiadm dead holdait le fd | FORCE_THRESHOLD=3600s dans iscsi-lock-cleanup (#2538) |
| external-dns-gandi CrashLoop | Cilium eBPF egress cassé (poison puis powder) après incident nœud | Restart cilium pod sur chaque nœud affecté |
| ArgoCD dev dex CreateContainerConfigError | Image dex v2.45.1 username non-numérique + runAsNonRoot | Patch runAsUser:1001 dans dev overlay (#2538) |
| booklore dataangel Pending forever | startupProbe 5min trop court pour restore S3 | failureThreshold 150→600 (#2538) |
| victoria-metrics OutOfSync permanent | Webhook cert auto-géré par VM operator driftait vs git | ignoreDifferences sur cert Secret + WebhookConfig (#2540) |

### Changements de sizing/infra
| Changement | PR | Raison |
|-----------|-----|--------|
| Nouveau tier V-3xlarge (1000m/8Gi → 4000m/32Gi) | #2542 | Frigate VPA target >2000m lim du SB-xlarge |
| Suppression media-xlarge | #2542 | Hors nomenclature B/G/SB/V-* |
| frigate: SB-xlarge → V-3xlarge | #2542 | CPU throttlé, VPA target 2406m > lim 2000m |
| pyload: V-micro → V-nano + dataangel V-small → V-nano | #2538/#2542 | Pearl 99% requests |
| grafana: B-large → V-medium + 2Gi lim | #2542 | OOMKilled 12x à 512Mi lim |

### Maturity — PriorityClass fixes
| PR | Changement |
|----|-----------|
| #2543 | `vixens.io/priority-class: vixens-low` ajouté sur 19 déploiements sans priorityClass (keda, kyverno, policy-reporter, victoria-metrics-operator, redis-shared, local-path-provisioner) |
| #2545 | Niveaux corrigés : redis-shared→high, local-path-provisioner→critical, kyverno-admission→high, keda→medium. Rule 1 mutate-priority-class rendue générique (apiCall + spec.priority) |

---

## Système de Maturité (ADR-023)

> **Référence:** [ADR-023: 7-Tier Goldification System v2](adr/023-7-tier-goldification-system-v2.md)

| Niveau | Nom | Description | Count |
|--------|-----|-------------|-------|
| 🥉 1 | **Bronze** | Déployée | 7 |
| 🥈 2 | **Silver** | Production Ready | 41 |
| 🥇 3 | **Gold** | Observable | 34 |
| 💎 4 | **Platinum** | Reliable | 10 |
| 🟢 5 | **Emerald** | Data Durability | 0 |
| 💠 6 | **Diamond** | Secure & Integrated | 0 |
| 🌟 7 | **Orichalcum** | Parfaite | 0 |
| ⚫ | **none** | Non labellisé | 0 |

**Total déploiements labellisés:** ~100 (tous labellisés après session 2026-03-28)

### Violations Kyverno top-5 (à traiter)
| Policy | Violations | Action recommandée |
|--------|-----------|-------------------|
| check-restore-init | 357 | Apps Emerald sans init container restore |
| check-vulnerability-scan | 274 | Scans Trivy manquants |
| check-pdb | 218 | PDB manquants sur apps Platinum+ |
| check-backup | 215 | Label backup-profile manquant |
| sizing-audit | 110 | Labels sizing non conformes v2 |

### Dettes techniques identifiées
| Dette | Impact | Effort |
|-------|--------|--------|
| local-path PVCs node-locked (amule, mealie, vikunja, pyload…) | Apps bloquées si nœud saturé | Migration iSCSI (moyen) |
| Nœuds 16GB (poison/powder) saturés en requests quand frigate VPA se déclenche | 4-6 apps Pending lors rolling updates | Soit plus de RAM, soit partitionnement workloads |
| grok-proxy hors repo | Pas traitable via GitOps | Intégrer dans le repo |
| hubble-relay hors repo (Talos bootstrap) | PriorityClass non gérée | Intégrer config Cilium dans le repo |

---

## Legend

| Symbol | Status | Description |
|--------|--------|-------------|
| ✅ | **Healthy** | Synced, Healthy, pas d'issues |
| ⚠️ | **Degraded** | Fonctionne mais restarts élevés ou policy violations |
| ❌ | **Broken** | Unhealthy ou OutOfSync |
| 🚧 | **Progressing** | Sync ou rollout en cours |
| 💤 | **Paused** | Intentionnellement non déployé |

---

## Infrastructure (00-infra/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| argocd | ✅ | 🥇 Gold | v3.x - Self-Managed |
| kyverno | ✅ | 💎 Platinum | 4 controllers |
| velero | ✅ | 🥇 Gold | v1.17.x + Infisical |
| traefik | ✅ | 🥇 Gold | v3.x Ingress controller |
| cert-manager | ✅ | 🥈 Silver | TLS via Let's Encrypt |
| cert-manager-webhook-gandi | ✅ | 🥇 Gold | DNS-01 challenge |
| cilium-operator | ✅ | 💎 Platinum | CNI |
| synology-csi | ✅ | - | iSCSI storage |
| infisical-operator | ✅ | 🥇 Gold | Secrets management |
| reloader | ✅ | 🥈 Silver | ConfigMap/Secret reload |
| vpa | ✅ | 💎 Platinum | Vertical Pod Autoscaler |
| metrics-server | ✅ | 💎 Platinum | Resource metrics |

---

## Monitoring (02-monitoring/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| prometheus | ✅ | 🥇 Gold | |
| grafana | ✅ | 🥉 Bronze | Needs upgrade |
| loki | ✅ | - | Log aggregation |
| promtail | ✅ | - | Fixed: Probe timeout 1s→5s (PR #1980) |
| goldilocks | ✅ | 🥇 Gold | VPA recommendations |
| descheduler | ✅ | - | Pod rebalancing |
| policy-reporter | ✅ | 🥇 Gold | Kyverno reporting |

---

## Security (03-security/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| authentik | ✅ | 🥈 Silver (server) / 🥇 Gold (worker) | SSO Provider |
| trivy | ✅ | 🥇 Gold | Vulnerability scanning |

---

## Databases (04-databases/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| postgresql-shared | ✅ | - | CloudNativePG |
| redis-shared | ✅ | 🥇 Gold | |
| mariadb-shared | ✅ | - | Fixed: Storage permissions (fsGroup) |
| cloudnative-pg | ✅ | 🥇 Gold | Operator |

---

## Home Automation (10-home/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| homeassistant | ✅ | 🥈 Silver | Fixed: OOM & Probes. Resources adjusted. |
| mealie | ✅ | 🥈 Silver | |
| mosquitto | ✅ | - | MQTT broker |

---

## Media (20-media/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| jellyfin | ✅ | 🥇 Gold | Media server |
| jellyseerr | ✅ | 🥇 Gold | Request management |
| sabnzbd | ✅ | 🥈 Silver | Usenet downloader |
| radarr | ⚠️ | 🥈 Silver | **Restarts**, DB verrous iSCSI |
| sonarr | ✅ | 🥈 Silver | TV Shows |
| prowlarr | ✅ | 🥈 Silver | Indexer manager |
| lidarr | ✅ | 🥈 Silver | Music |
| mylar | ✅ | 🥈 Silver | Comics |
| whisparr | ✅ | 🥈 Silver | Adult content |
| lazylibrarian | ✅ | 🥈 Silver | Books/Audiobooks |
| music-assistant | ✅ | 🥇 Gold | |
| frigate | ✅ | 🥈 Silver | NVR - **DB Migrée en RAM** |
| hydrus-client | ✅ | 🥈 Silver | |
| booklore | ✅ | 🥇 Gold | |
| birdnet-go | ✅ | 🥉 Bronze | Bird detection |
| qbittorrent | ✅ | 💎 Platinum | Torrent client |
| pyload | ✅ | 💎 Platinum | Download manager |
| amule | ✅ | ⚫ none | ED2K client |

---

## Network (40-network/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| external-dns-unifi | ✅ | 🥇 Gold | Internal DNS |
| external-dns-gandi | ✅ | 🥇 Gold | Public DNS |
| netbird | ⚠️ | 🥇 Gold | **42 restarts**, SecurityContext manquant |
| netvisor | 🚧 | 🥇 Gold | Progressing |
| adguard-home | ✅ | - | Known: DNS dependency cycle (Litestream→MinIO) |
| contacts | ✅ | - | Redirection |

---

## Services (60-services/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| mail-gateway | ✅ | - | Email gateway |
| vaultwarden | ✅ | 🥈 Silver | Password manager |
| docspell | ✅ | - | Document management |
| gluetun | ✅ | 🥇 Gold | VPN container |
| firefly-iii | ✅ | 🥉 Bronze | Finance - needs upgrade |
| firefly-iii-importer | ✅ | 💎 Platinum | |
| OpenClaw | 💤 | - | Retired from GitOps; Argo CD will prune after the approved merge |

---

## Tools (70-tools/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| whoami | ✅ | 🥇 Gold | Test app |
| homepage | ✅ | 💎 Platinum | Dashboard |
| netbox | ✅ | 🥇 Gold | IPAM |
| changedetection | ✅ | 🥇 Gold | Website monitoring |
| stirling-pdf | ✅ | 🥈 Silver | PDF tools |
| it-tools | ✅ | 🥇 Gold | Dev tools |
| headlamp | ✅ | 🥇 Gold | K8s dashboard |
| linkwarden | ✅ | 🥇 Gold | Bookmark manager |
| vikunja | ⚠️ | 💎 Platinum | **37 restarts** |
| penpot | ✅ | 🥇 Gold | Design platform |
| renovate | ✅ | - | Fixed: OOMKilled - resources 512Mi→2Gi (PR #1980) |
| trilium | ✅ | 💎 Platinum | Notes |
| nocodb | ✅ | 💎 Platinum | Airtable alternative |
| radar | ✅ | 💎 Platinum | |

---

## Applications avec Issues

| Application | Restarts | Issue principale |
|-------------|----------|------------------|
| netbird-management | 42 | SecurityContext non durci |
| vikunja | 37 | À investiguer |

| homeassistant | 16 | PDB manquant |

---

## Top Policy Failures (Kyverno)

| Policy | Failures | Impact |
|--------|----------|--------|
| check-backup | 317 | Emerald bloqué |
| check-pdb | 237 | Platinum bloqué |
| require-resources | 237 | Silver bloqué |
| require-probes | 198 | Silver bloqué |
| sizing-audit | 126 | Gold bloqué |
| check-security-context | 121 | Diamond bloqué |

---

## Environment Information

### Prod Cluster

- **Nodes:** peach, pearl, phoebe, poison, powder (3 CP + 2 workers)
- **VIP:** 192.168.111.190
- **IP Range:** 192.168.111.191 - 192.168.111.195
- **Status:** ✅ Active

### Dev Cluster

- **Nodes:** daphne, diva, dulce (3 CP HA)
- **VIP:** 192.168.111.160
- **Status:** ❌ Certificat invalide (kubeconfig à regénérer)

---

## Related Documentation

- **[ADR-023: 7-Tier Goldification System v2](adr/023-7-tier-goldification-system-v2.md)** - Source de vérité pour les niveaux de maturité
- **[Application Documentation](applications/)** - Documentation par app
- **[Maturity Standards Matrix](reference/maturity-standards-matrix.md)** - Matrice des exigences par niveau

---

**Last Updated:** 2026-03-11
