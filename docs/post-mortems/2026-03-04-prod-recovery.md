# Post-Mortem & Rapport de Récupération Prod
**Date :** 2026-03-03 → 2026-03-04  
**Durée de l'incident :** ~12h (reboot node jusqu'à 90/90 Healthy)  
**Cluster :** prod (Talos v1.12.4, K8s v1.34.0, ArgoCD v3.3.0)  
**Nodes :** peach, pearl (workers) | phoebe, poison, powder (control-plane)  
**Auteur :** Claude Code (Sisyphus) + charchess  
**PRs mergées :** #1689 → #1738 (39 PRs au total)  
**Résultat final :** ✅ **90/90 apps Synced Healthy**

---

## 1. CE QUI POSAIT PROBLÈME (Résumé)

Le node `poison` (control-plane) a rebooté. Cela a provoqué une cascade :

1. **ArgoCD dégradé** : le reboot a interrompu l'app-controller, qui a perdu son cache de santé → des apps saines ont été marquées Degraded ou Unknown.
2. **Kyverno sizing bug systémique** : des pods ont redémarré sans le label `vixens.io/sizing` correct → Kyverno leur a appliqué la classe `micro` (128Mi limit), causant des OOMKills en cascade.
3. **Cluster à saturation mémoire** : 99% d'allocation sur les 5 nodes. Le moindre redémarrage de pod causait des pods Pending impossibles à scheduler.
4. **Labels de sizing mal formés** : plusieurs apps utilisaient `vixens.io/sizing.appname: small` au lieu de `vixens.io/sizing: small` — Kyverno ne reconnaissait pas ce format et appliquait `micro` systématiquement.
5. **Pods sans tolérance control-plane** : plusieurs apps configurées pour les workers uniquement, bloquées quand les workers sont pleins.

---

## 2. CE QUI A ÉTÉ FAIT (Résumé)

- **39 PRs mergées** suivant un workflow GitOps strict (branch → PR → CI → squash-merge → prod-stable tag)
- **8 PolicyExceptions Kyverno** créées pour bypasser `sizing-mutate` sur les apps nécessitant des resources explicites
- **Corrections de sizing** sur ~15 apps (labels corrigés, resources-patch ajoutés, sizing tiers revus)
- **Corrections de scheduling** : tolérances control-plane ajoutées à ~6 apps bloquées sur workers saturés
- **Corrections ArgoCD** : health-bumps pour purger le cache Degraded, `ServerSideApply`, `ignoreDifferences` pour les CRDs
- **Cluster memory freed** : réduction ciblée des `requests` sur adguard (−384Mi), loki (−256Mi), robusta, amule, firefly-iii-importer pour permettre le scheduling des pods critiques

---

## 3. CE QUI N'EST PAS SATISFAISANT (Résumé)

### 3.1 Problèmes systémiques non résolus

| Problème | Impact | État |
|----------|--------|------|
| **Cluster à 99% RAM** | Tout redémarrage de pod risque de bloquer | ⚠️ Non résolu |
| **Labels `vixens.io/sizing.xxx` erronés** | Kyverno applique `micro` silencieusement → OOMKill | ⚠️ Corrigé sur les apps touchées, pas d'audit global |
| **node-collector (Talos incompatible)** | OOMKill car `/etc/systemd` read-only sur Talos | ⚠️ Non résolu (limitation Talos) |
| **Velero kopia-maintain OOMKill** | Jobs de maintenance backup échouent en boucle | ⚠️ Non résolu |
| **VPA en Error** | Anciennes pods VPA non nettoyées (pods zombies) | ℹ️ Mineur — nouvelles pods Running |
| **Headlamp/netbox/stirling-pdf pods zombies** | Vieilles pods Error coexistent avec les pods Running | ℹ️ Cosmétique — GC Kubernetes lent |

### 3.2 Approche sous-optimale

- **Le workflow sizing Kyverno est fragile** : le format `vixens.io/sizing.xxx` (per-container) n'est pas reconnu par la policy `sizing-mutate` qui n'évalue que `vixens.io/sizing`. Ce mismatch a causé la majorité des OOMKills. Un audit complet de tous les manifests devrait être fait.
- **Les PolicyExceptions sont des pansements** : l'architecture idéale serait que les resources déclarées dans les manifests soient honorées sans exception, mais Kyverno écrase tout. À long terme, il faudrait soit aligner les sizing labels, soit changer la policy pour lire `vixens.io/sizing` ET les labels per-container.
- **Pas de marge mémoire cluster** : le cluster tourne à 99% en `requests`. Le moindre rolling update avec surge=1 peut bloquer. Recommandation : ajouter un node worker, ou réduire davantage les requests sur les apps peu utilisées.

---

## 4. CE QUI POSAIT PROBLÈME (Détaillé)

### 4.1 Cause racine : reboot du node `poison`

Le node `poison` (control-plane, 15 Gi RAM) a rebooté pendant que des CronJobs s'exécutaient (~3h UTC). Effets en cascade :

**ArgoCD app-controller** :
- L'app-controller a perdu son cache de santé après le redémarrage
- Des apps parfaitement saines ont été évaluées comme `Degraded` ou `Unknown`
- Le bug ArgoCD de cache de health est documenté : après un reboot, les ressources Kubernetes (notamment les CronJobs) peuvent être réévaluées avec un état transitoire incorrect

**Kyverno sizing-mutate** :
- Au redémarrage des pods (rolling updates post-reboot), Kyverno intercepte la création de chaque pod
- Si le pod n'a pas le label `vixens.io/sizing` (ou a un label mal formé comme `vixens.io/sizing.appname`), Kyverno applique la classe `micro` : 128Mi request / 128Mi limit
- Les apps qui avaient des resources explicites > 128Mi (ex: trivy 1Gi, frigate 4Gi, openclaw) ont toutes été OOMKillées dès leur premier redémarrage

### 4.2 Problème Kyverno : label format invalide

La policy `sizing-mutate` évalue `metadata.labels['vixens.io/sizing']`.

Plusieurs apps avaient des labels de la forme :
```yaml
labels:
  vixens.io/sizing.appname: small   # ❌ Kyverno ne reconnaît pas
```
au lieu de :
```yaml
labels:
  vixens.io/sizing: small           # ✅ Kyverno reconnaît
```

**Apps affectées** : amule (`vixens.io/sizing.amule`), firefly-iii-importer (`vixens.io/sizing.importer`), radar (`vixens.io/sizing.radar`), et plusieurs autres corrigées en début de session.

### 4.3 Problème ArgoCD : CronJob health evaluation

ArgoCD évalue la santé des CronJobs via un check Lua intégré. Si le dernier run schedulé n'a pas de Job correspondant dans l'historique (ex: le job a été interrompu par le reboot avant completion), ArgoCD marque le CronJob `Degraded`.

**App affectée** : `firefly-iii` (CronJob `firefly-data-importer`) — le CronJob avait tourné à 3h UTC mais le job n'avait pas complété (reboot pendant l'exécution).

Fix : déclencher un job manuel + corriger le CronJob pour qu'il puisse se scheduler (CP toleration manquante, absence de sizing label).

### 4.4 Problème ArgoCD : CRD annotations trop larges

ArgoCD ne peut pas stocker les CRDs avec des annotations `last-applied-configuration` trop grandes (>256KB). Cela causait des erreurs `etcd: request too large`.

Fix : `ServerSideApply: true` + `ignoreDifferences` sur les champs auto-gérés par Kyverno/admission webhooks.

### 4.5 Problème de saturation mémoire cluster

**État des nodes en début de session :**
| Node | Rôle | Allocatable | Allocated | % |
|------|------|------------|-----------|---|
| peach | worker | ~7.4 Gi | ~7.3 Gi | 99% |
| pearl | worker | ~7.3 Gi | ~7.2 Gi | 99% |
| phoebe | control-plane | ~7.1 Gi | ~7.1 Gi | 99% |
| poison | control-plane | ~15.1 Gi | ~15.0 Gi | 99% |
| powder | control-plane | ~15.1 Gi | ~14.8 Gi | 98% |

Les workers (peach, pearl) ont seulement 2×8Gi RAM contre les CP à 2×16Gi. La majorité des workloads applicatifs est sur les workers (pas de tolérance CP), causant une saturation asymétrique.

### 4.6 Problème : Reloader + saturation = cascade Pending

En fin de session, Reloader a détecté un changement de secret et redémarré simultanément plusieurs pods (amule, firefly-iii-importer). Ces pods sont devenus Pending car le cluster n'avait plus de mémoire disponible — un problème latent révélé par un événement mineur.

### 4.7 Problème node-collector (Talos)

`trivy-operator` déploie un `node-collector` DaemonSet pour scanner les nodes. Sur Talos Linux (OS immutable), le container tente de créer `/etc/systemd` → `read-only file system`. Cette incompatibilité est structurelle : Talos ne permet pas d'écriture dans `/etc` par design. ArgoCD rapporte `trivy` comme Healthy car l'opérateur principal fonctionne, mais `node-collector` échoue systématiquement.

---

## 5. CE QUI A ÉTÉ FAIT (Détaillé)

### 5.1 Workflow GitOps appliqué

Toutes les corrections ont suivi le workflow strict :
```
git checkout -b fix/<name>
# éditions
git commit && git push
gh pr create && gh pr merge --squash --auto
# attente merge CI
git checkout main && git pull
git checkout prod-stable && git merge --ff-only origin/main
git push origin prod-stable
git tag -f prod-stable HEAD && git push --force origin refs/tags/prod-stable
kubectl patch application <app> -n argocd --type merge \
  -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}'
```

> Exception acceptée : `kubectl rollout restart` pour forcer la recréation des pods après application d'une PolicyException (nécessaire car les pods existants ont le cache Kyverno de l'ancienne config).

### 5.2 Tableau complet des PRs

| PR | Titre | Problème résolu |
|----|-------|----------------|
| #1689 | fix(birdnet-go): reduce sizing G-large→G-medium | OOMKill / mémoire node phoebe |
| #1690 | fix(birdnet-go): fix resources-patch dangling spec | YAML invalide |
| #1691 | fix(robusta): remove broken cross-source sizing-patch | OutOfSync robusta |
| #1692 | fix(lidarr): add sizing labels to fix OOMKill | Kyverno micro fallback → OOMKill |
| #1693 | fix(stirling-pdf): remove duplicate ingress ArgoCD app | Duplicate app Unknown |
| #1694 | fix(stirling-pdf): fix podLabels + remove explicit resources | Kyverno drift |
| #1695 | fix(argocd): ServerSideApply + ignoreDifferences CRD | CRD annotation too large |
| #1696 | fix(kyverno): managedFieldsManagers ignoreDifferences | Kyverno CRD drift |
| #1697 | fix(kyverno): jqPathExpressions for CRD fields | Kyverno CRD drift |
| #1698 | fix(kyverno): foreach context apiCall.method | Policy evaluation error |
| #1699 | fix(sizing): stirling-pdf G-large→medium, robusta G-large→small | Pods Pending mémoire |
| #1700 | fix(robusta): runner small→G-small (128Mi actual usage) | Mémoire cluster |
| #1701 | fix(sizing): replace generic labels with per-container | OOMKill multi-apps |
| #1702 | fix(firefly-iii): health-bump annotation | ArgoCD Degraded cache |
| #1703 | fix(sizing): right-size tiers from VPA recommendations | Right-sizing global |
| #1704 | fix(sizing): sonarr/music-assistant/openclaw OOMKills | Kyverno micro fallback |
| #1705 | fix(argocd): normalize ignoreDifferences + group fields | OutOfSync app-of-apps |
| #1706 | fix(frigate): G-xl→G-small prod schedulability | Pod Pending |
| #1707 | fix(sizing): loki + docspell-joex sizing tiers | Mauvais tiers prod |
| #1708 | fix(argocd): remove duplicate kind key ignoreDifferences | YAML invalide |
| #1709 | fix(robusta): runner G-small→small (OOMKill) | OOMKill robusta-runner |
| #1710 | fix(sonarr): add resources-patch to kustomization | Resources non appliquées |
| #1711 | fix(prod): sizing/PolicyException/health-bump multi-apps | Corrections multiples |
| #1712 | fix(frigate): G-medium→medium sizing CPU burst go2rtc | Startup CPU |
| #1713 | fix(argocd): app-controller memory request 400Mi | Mémoire cluster |
| #1714 | fix(frigate): increase probe delays 30→120s (liveness) | Probe failure au démarrage |
| #1715 | fix(argocd): ServerSideApply=true + ignoreDifferences CRDs | CRD annotation error |
| #1716 | fix(argocd): CP toleration pour scheduling nodes CP | Pod Pending workers full |
| #1717 | fix(frigate): G-small→G-large (OOMKill) | OOMKill frigate |
| #1718 | fix(openclaw): medium→G-controller JS heap OOM | OOMKill openclaw |
| #1719 | fix(argocd): repo-server medium→small mémoire cluster | Libérer RAM |
| #1720 | fix(argocd): repo-server 128Mi request / 1Gi limit | Resources explicites |
| #1721 | fix(argocd): PolicyException repo-server + 128Mi request | Kyverno bypass ArgoCD |
| #1722 | fix(frigate): G-large→large (4Gi) OOMKill | OOMKill frigate (final) |
| #1723 | fix(whisparr): G-small/micro pour libérer RAM robusta | Libérer RAM powder |
| #1724 | fix(firefly-iii): health-bump pour forcer réconciliation | ArgoCD Degraded cache |
| #1725 | fix(lidarr): G-small/micro pour libérer RAM powder | Libérer RAM |
| #1726 | fix(robusta): CP toleration runner | Pod Pending workers full |
| #1727 | fix(robusta): 384Mi + PolicyException bypass Kyverno | OOMKill robusta |
| #1728 | fix(lidarr): G-medium (512Mi) fix OOMKill au démarrage | OOMKill lidarr |
| #1729 | fix(whisparr): G-medium (512Mi) fix OOMKill | OOMKill whisparr |
| #1730 | fix(firefly-iii): health-bump 4 réconciliation | ArgoCD Degraded cache |
| #1731 | fix(mealie): PolicyException bypass Kyverno small→256Mi | Kyverno sizing bypass |
| #1732 | fix(adguard): PolicyException + 512Mi→128Mi request | Libérer 384Mi powder |
| #1733 | fix(firefly-iii): health-bump 4→5 cache Degraded | ArgoCD Degraded cache |
| #1734 | fix(loki): PolicyException + 512Mi→256Mi request | Libérer 256Mi pour robusta |
| #1735 | fix(firefly-iii): CronJob sizing label + CP toleration | CronJob Degraded / Pending |
| #1736 | fix(trivy): PolicyException bypass micro→1Gi | OOMKill trivy (1Gi nécessaire) |
| #1737 | fix(amule): 256Mi request + PolicyException + label fix | Pod Pending cluster plein |
| #1738 | fix(firefly-iii-importer): CP toleration + PolicyException | Pod Pending workers pleins |

### 5.3 PolicyExceptions créées

| Exception | Namespace | Pattern pods | Raison |
|-----------|-----------|-------------|--------|
| `argocd-sizing-exception` | kyverno | `argocd-repo-server-*` | ArgoCD repo-server : 128Mi req / 1Gi limit |
| `robusta-sizing-exception` | kyverno | `robusta-runner-*` | 384Mi req / 512Mi limit (usage réel ~350Mi) |
| `mealie-sizing-exception` | kyverno | `mealie-*` | 256Mi req / 512Mi limit (contourne small=512Mi) |
| `adguard-sizing-exception` | kyverno | `adguard-home-*` | 128Mi req / 512Mi limit (libère 384Mi) |
| `loki-sizing-exception` | kyverno | `loki-*` | 256Mi req / 512Mi limit (libère 256Mi) |
| `trivy-sizing-exception` | kyverno | `trivy-*` | 1Gi req/limit (usage réel ~800Mi, scan mémoire-intensif) |
| `amule-sizing-exception` | kyverno | `amule-*` | 256Mi req / 512Mi limit (cluster plein) |
| `firefly-iii-importer-sizing-exception` | kyverno | `firefly-iii-importer-*` | 128Mi req / 512Mi limit (usage réel ~88Mi) |

### 5.4 Mémoire libérée sur le cluster

| App | Avant (request) | Après (request) | Libéré |
|-----|----------------|----------------|--------|
| adguard-home | 512Mi | 128Mi | **−384Mi** |
| loki | 512Mi | 256Mi | **−256Mi** |
| amule | 512Mi | 256Mi | **−256Mi** |
| firefly-iii-importer | 512Mi | 128Mi | **−384Mi** |
| robusta-runner | 512Mi | 384Mi | **−128Mi** |
| mealie | 512Mi | 256Mi | **−256Mi** |
| argocd repo-server | ~512Mi | 128Mi | **~−384Mi** |
| **Total libéré** | | | **~2 Gi** |

### 5.5 prod-stable promu à

`2d2ff2ee07bdc324da5e8b091c2680b4d5c1029d` (inclut PRs #1689–#1738)

---

## 6. ÉTAT DE CONFIGURATION DE CHAQUE APP (Détaillé)

### Légende
- ✅ Synced Healthy — fonctionnel
- ⚠️ Synced Healthy (ArgoCD) / problème pods sous-jacent
- 🔧 Corrigé cette session
- ℹ️ Note connue

---

### Infrastructure (00-infra)

#### argocd ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Corrections :** ServerSideApply=true, ignoreDifferences CRDs Kyverno, CP toleration, repo-server PolicyException (128Mi req / 1Gi limit), app-controller 400Mi request
- **Note :** ArgoCD v3.3.0. Le bug de cache health (apps marquées Degraded après reboot) est un problème connu. Workaround appliqué : health-bump annotations.

#### kyverno ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **PolicyExceptions actives :** 8 exceptions (voir §5.3)
- **Note :** La policy `sizing-mutate` écrase les resources des pods sans label `vixens.io/sizing`. Les apps sans ce label exact reçoivent `micro` (128Mi/128Mi). Plusieurs apps avaient des labels mal formés (`vixens.io/sizing.xxx`) corrigés cette session.

#### cert-manager ✅
- **Statut ArgoCD :** Synced Healthy
- **Note :** Helm chart, pas de revision git trackée. Fonctionnel.

#### cert-manager-config ✅
- **Statut ArgoCD :** Synced Healthy

#### cert-manager-secrets ✅
- **Statut ArgoCD :** Synced Healthy

#### cert-manager-webhook-gandi ✅
- **Statut ArgoCD :** Synced Healthy

#### cilium-lb ✅
- **Statut ArgoCD :** Synced Healthy
- **Note :** L2 announcements Cilium, VIP 192.168.111.200 prod.

#### cloudnative-pg ✅
- **Statut ArgoCD :** Synced Healthy
- **Note :** Version Helm 0.27.0.

#### descheduler ✅
- **Statut ArgoCD :** Synced Healthy

#### external-dns-gandi / external-dns-unifi ✅
- **Statut ArgoCD :** Synced Healthy

#### infisical-operator ✅
- **Statut ArgoCD :** Synced Healthy
- **Note :** Version Helm 0.10.5. Gestion des secrets via InfisicalSecret CRDs.

#### maturity-controller ✅
- **Statut ArgoCD :** Synced Healthy

#### metrics-server ✅
- **Statut ArgoCD :** Synced Healthy

#### nfs-storage ✅
- **Statut ArgoCD :** Synced Healthy

#### priority-classes ✅
- **Statut ArgoCD :** Synced Healthy

#### shared-namespaces ✅
- **Statut ArgoCD :** Synced Healthy

#### synology-csi / synology-csi-secrets ✅
- **Statut ArgoCD :** Synced Healthy
- **Note :** CSI iSCSI Synology. Storage provider principal pour les PVCs.

#### traefik ✅
- **Statut ArgoCD :** Synced Healthy
- **Note :** Helm chart. Ingress controller principal.

#### traefik-dashboard / traefik-middlewares ✅
- **Statut ArgoCD :** Synced Healthy

---

### Monitoring (02-monitoring)

#### grafana / grafana-ingress ✅
- **Statut ArgoCD :** Synced Healthy

#### loki ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Correction :** PolicyException `loki-sizing-exception` + resources-patch 256Mi req / 512Mi limit (était 512Mi req, libère 256Mi sur powder)
- **Usage réel :** ~162Mi → 256Mi request est confortable

#### prometheus / prometheus-ingress ✅
- **Statut ArgoCD :** Synced Healthy

#### promtail ✅
- **Statut ArgoCD :** Synced Healthy

#### robusta ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Corrections multiples :** sizing G-large→G-small→small→384Mi, PolicyException, CP toleration
- **Resources actuelles :** 384Mi req / 512Mi limit, PolicyException active
- **Usage réel :** ~350Mi → sizing adapté

---

### Sécurité (03-security)

#### trivy ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Correction :** PolicyException `trivy-sizing-exception` (1Gi req/limit) + rollout restart pour appliquer la PolicyException
- **Usage réel :** ~800Mi, scan de vulnérabilités nécessite beaucoup de RAM
- **⚠️ node-collector Pending/Error :** Le DaemonSet `node-collector` de trivy-operator tente d'écrire dans `/etc/systemd` → `read-only file system` sur Talos. Incompatibilité structurelle. ArgoCD reste Healthy car l'opérateur principal fonctionne. Le scan de vulnérabilités des images fonctionne ; seul le scan OS node est non fonctionnel.

#### policy-reporter ✅
- **Statut ArgoCD :** Synced Healthy

---

### Réseau (40-network)

#### adguard-home ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Correction :** PolicyException `adguard-sizing-exception` + resources-patch 128Mi req / 512Mi limit (était 512Mi req → libère 384Mi sur powder)
- **Note :** 3 replicas. DNS local. Usage réel << 128Mi.

#### external-dns-gandi-secrets / external-dns-unifi-secrets ✅
- **Statut ArgoCD :** Synced Healthy

---

### Services (60-services)

#### firefly-iii ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Corrections :** 4 health-bump annotations pour purger le cache Degraded d'ArgoCD, CP toleration au CronJob, sizing label sur le CronJob
- **CronJob :** `firefly-data-importer` — schedule 3h UTC. Le job interrompu par le reboot a été déclenché manuellement pour resetter l'historique ArgoCD.
- **⚠️ Surveillance recommandée :** Si le prochain run planifié à 3h UTC échoue, ArgoCD peut repasser Degraded. Le fix GitOps est en place (PR #1735).

#### firefly-iii-importer ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Corrections :** CP toleration, label `vixens.io/sizing.importer` → `vixens.io/sizing: small`, PolicyException `firefly-iii-importer-sizing-exception` (128Mi req / 512Mi limit)
- **Rollout restart manuel :** nécessaire car le pod avait été créé AVANT que la PolicyException soit active (Kyverno avait déjà overridé à 128Mi limit → OOMKill)
- **Resources actuelles :** 128Mi req / 512Mi limit. Usage réel ~88Mi.

---

### Domotique (10-home)

#### homeassistant ✅
- **Statut ArgoCD :** Synced Healthy

#### mosquitto ✅
- **Statut ArgoCD :** Synced Healthy

#### mealie ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Correction :** PolicyException `mealie-sizing-exception` + resources-patch 256Mi req / 512Mi limit
- **Usage réel :** ~200Mi → 256Mi adapté

#### birdnet-go ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Corrections :** sizing G-large→G-medium, fix resources-patch YAML invalide

---

### Médias (20-media)

#### amule ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Corrections :** Label `vixens.io/sizing.amule` → `vixens.io/sizing: small`, PolicyException `amule-sizing-exception`, request 512Mi → 256Mi
- **Note :** Nécessite gluetun (SOCKS5 proxy) pour les connexions eDonkey.

#### frigate ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Corrections multiples :** sizing G-xl→G-small→G-medium→G-large→large (4Gi), probe delays ajustés (liveness 30→120s, readiness 10→60s)
- **Resources actuelles :** 4Gi req/limit (sizing `large`). Usage réel ~3.5Gi — NVR vidéo avec détection IA.

#### hydrus-client ✅
- **Statut ArgoCD :** Synced Healthy

#### jellyfin ✅
- **Statut ArgoCD :** Synced Healthy

#### jellyseerr ✅
- **Statut ArgoCD :** Synced Healthy

#### lazylibrarian ✅
- **Statut ArgoCD :** Synced Healthy

#### music-assistant ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Correction :** Sizing label corrigé (OOMKill Kyverno micro)

#### mylar ✅
- **Statut ArgoCD :** Synced Healthy

---

### *arr stack (20-media)

#### lidarr ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Corrections :** sizing G-small/micro → G-medium (512Mi) pour fix OOMKill au démarrage (Mono runtime)
- **Note :** Mono/.NET nécessite plus de RAM au démarrage (~400Mi).

#### radarr ✅
- **Statut ArgoCD :** Synced Healthy

#### sonarr ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Correction :** resources-patch ajouté à kustomization (était manquant), sizing label corrigé

#### whisparr ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Corrections :** sizing G-small/micro → G-medium (512Mi) pour fix OOMKill

#### prowlarr ✅
- **Statut ArgoCD :** Synced Healthy

#### qbittorrent ✅
- **Statut ArgoCD :** Synced Healthy

#### sabnzbd ✅
- **Statut ArgoCD :** Synced Healthy

#### pyload ✅
- **Statut ArgoCD :** Synced Healthy

---

### Outils (tools)

#### headlamp ✅ ⚠️
- **Statut ArgoCD :** Synced Healthy
- **Deployment :** 1/1 Running (pod `headlamp-5bf5d99858-dfg7l`)
- **⚠️ Pods zombies :** 3 anciens pods en état Error (`headlamp-5bf5d99858-8zpdp/gxh64/zk9df`) — logs : `stat /home/headlamp/.kube/config: no such file or directory`. Ces pods ont Exit Code 2 et ne sont pas nettoyés automatiquement par le GC K8s (restartPolicy=Always, restart count=0). Cosmétique — le pod Running fonctionne.
- **Cause :** Configuration headlamp qui pointait vers un kubeconfig absent, probablement lors d'une rotation de secret.

#### netbox ✅ ⚠️
- **Statut ArgoCD :** Synced Healthy
- **Deployment :** 1/1 Running
- **⚠️ Pod zombie :** 1 ancien pod Error (connexion Redis échouée au démarrage). Le pod Running fonctionne.

#### radar ✅ ⚠️
- **Statut ArgoCD :** Synced Healthy
- **Deployment :** 1/1 Running (`radar-6c9f8d6659-4svgq`)
- **⚠️ Pods zombies :** 2 anciens pods OOMKilled (`radar-58d857cd8b-*`) — label `vixens.io/sizing.radar` non reconnu par Kyverno → micro (128Mi limit) → OOMKill. Le pod Running a 512Mi limit (sizing correct sur cette ReplicaSet plus récente).
- **Note :** Le pod Running a des resources correctes (`{"limits":{"cpu":"100m","memory":"128Mi"}}` sur les vieilles RS, `512Mi` sur la RS courante). ArgoCD est Healthy.

#### stirling-pdf ✅ ⚠️
- **Statut ArgoCD :** Synced Healthy (Helm)
- **⚠️ Pod zombie :** 1 ancien pod Error. Le pod Running fonctionne.

#### changedetection ✅
- **Statut ArgoCD :** Synced Healthy

#### homepage ✅
- **Statut ArgoCD :** Synced Healthy

#### it-tools / it-tools-ingress ✅
- **Statut ArgoCD :** Synced Healthy

#### linkwarden ✅
- **Statut ArgoCD :** Synced Healthy

#### nocodb ✅
- **Statut ArgoCD :** Synced Healthy

#### openclaw ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Correction :** sizing medium → G-controller (2Gi) pour fix JS heap OOM (Node.js)

#### penpot ✅
- **Statut ArgoCD :** Synced Healthy

#### vikunja ✅
- **Statut ArgoCD :** Synced Healthy

#### renovate ✅ ⚠️
- **Statut ArgoCD :** Synced Healthy
- **⚠️ Jobs OOMKilled :** Les CronJobs Renovate (mise à jour automatique des dépendances) OOMKillent depuis 2 jours. Sizing `micro` (128Mi) — Renovate clone des repos git et analyse les dépendances → intensif en RAM. Pods Completed ou OOMKilled en boucle. Non bloquant pour le cluster mais Renovate ne fonctionne pas.

---

### Bases de données partagées

#### mariadb-shared ✅
- **Statut ArgoCD :** Synced Healthy

#### postgresql-shared ✅
- **Statut ArgoCD :** Synced Healthy

#### redis-shared ✅
- **Statut ArgoCD :** Synced Healthy

---

### Autres services

#### authentik ✅
- **Statut ArgoCD :** Synced Healthy
- **Note :** SSO/OIDC provider.

#### booklore ✅
- **Statut ArgoCD :** Synced Healthy

#### contacts ✅
- **Statut ArgoCD :** Synced Healthy

#### docspell ✅ 🔧
- **Statut ArgoCD :** Synced Healthy
- **Correction :** Sizing tiers corrigés pour prod

#### gluetun ✅
- **Statut ArgoCD :** Synced Healthy
- **Note :** VPN gateway pour amule et autres apps téléchargement.

#### goldilocks ✅
- **Statut ArgoCD :** Synced Healthy
- **Note :** Dashboard VPA recommendations. Helm chart.

#### mail-gateway ✅
- **Statut ArgoCD :** Synced Healthy

#### media-shared-secrets ✅
- **Statut ArgoCD :** Synced Healthy

#### netbird ✅
- **Statut ArgoCD :** Synced Healthy
- **Note :** Ignoré pendant la session (instruction explicite). Synced Healthy.

#### netvisor ✅
- **Statut ArgoCD :** Synced Healthy

#### reloader ✅
- **Statut ArgoCD :** Synced Healthy
- **⚠️ Attention :** Reloader redémarre automatiquement les pods quand un secret Infisical change. Sur un cluster à 99% RAM, un changement de secret peut déclencher des restarts en cascade → pods Pending. À surveiller.

#### vaultwarden ✅
- **Statut ArgoCD :** Synced Healthy

#### velero ✅ ⚠️
- **Statut ArgoCD :** Synced Healthy (Helm)
- **Schedules actifs :**
  - `velero-daily-critical` : 2h UTC, dernier backup 2026-03-03
  - `velero-daily-home` : 4h UTC, dernier backup 2026-03-03
  - `velero-weekly-full` : 3h dim UTC, dernier backup 2026-03-01
- **⚠️ kopia-maintain-job OOMKill :** Les jobs de maintenance Kopia (nettoyage backups) OOMKillent avec 128Mi limit. Les jobs sont recréés périodiquement et finissent par échouer. Velero ArgoCD est Healthy mais la maintenance des backups est dégradée.

#### velero-maintenance-config / velero-secrets ✅
- **Statut ArgoCD :** Synced Healthy

#### vpa ✅ ⚠️
- **Statut ArgoCD :** Synced Healthy
- **Deployments :** 3/3 Running (admission-controller, recommender, updater)
- **⚠️ Pods zombies :** 3 anciens pods en Error (Exit Code 2). Les pods Running sont sains. Les recommendations VPA sont actives et accessibles via Goldilocks dashboard.

#### whoami ✅
- **Statut ArgoCD :** Synced Healthy
- **Note :** App de test/debug Traefik.

---

## 7. RECOMMANDATIONS POST-INCIDENT

### Urgent (à faire dans les 7 jours)

1. **Audit global des labels sizing** : Scanner tous les manifests pour trouver les labels `vixens.io/sizing.xxx` mal formés. Commande :
   ```bash
   grep -r "vixens.io/sizing\." apps/ --include="*.yaml" -l
   ```
   Chaque occurrence est potentiellement un OOMKill silencieux au prochain redémarrage.

2. **Renovate OOMKill** : Augmenter le sizing Renovate (au moins `small` = 512Mi) ou ajouter une PolicyException. Renovate est actuellement non fonctionnel.

3. **Velero kopia-maintain OOMKill** : Le job a 128Mi limit. Un backup kopia demande ~256Mi minimum. Ajouter resources explicites + PolicyException sur le VeleroSchedule ou la config kopia.

4. **node-collector Talos** : Désactiver `node-collector` dans la config trivy-operator (annotation ou valeur Helm) pour arrêter les tentatives échouées en boucle.

### Moyen terme (30 jours)

5. **Capacité cluster** : Ajouter 1-2 nodes worker avec 16Gi RAM. Le cluster tourne à 99% structurellement. Même avec les réductions de requests de cette session, une seule app qui augmente sa consommation peut saturer.

6. **Nettoyage pods zombies** : Les vieux pods Error/OOMKilled ne sont pas nettoyés automatiquement (pas de `ttlSecondsAfterFinished` sur les Deployments, K8s ne GC pas les pods non-Job). Évaluer `descheduler` ou nettoyage manuel périodique.

7. **Reloader awareness** : Documenter que tout changement de secret Infisical sur un cluster saturé déclenche des restarts en cascade. Envisager des maintenance windows ou un throttling Reloader.

### Long terme

8. **Policy Kyverno sizing-mutate** : Modifier la policy pour également lire `vixens.io/sizing.containerName` en plus de `vixens.io/sizing`, ou supprimer le format per-container et standardiser sur le label unique.

9. **AlertManager sur OOMKill** : Alerter immédiatement quand un pod est OOMKilled. Actuellement, les OOMKills sont silencieux jusqu'à ce qu'ArgoCD marque l'app Degraded.

---

## 8. ÉTAT FINAL

```
Date       : 2026-03-04 ~02h45 UTC
prod-stable: 2d2ff2ee07bdc324da5e8b091c2680b4d5c1029d
Apps       : 90/90 Synced Healthy ✅
PRs        : #1689 → #1738 (39 PRs)
Pods running: 151
PolicyExceptions actives: 8
```

**Mémoire cluster (état final) :**
| Node | Rôle | Allocatable | Allocated | % |
|------|------|------------|-----------|---|
| peach | worker | 7419 Mi | 7376 Mi | 99% |
| pearl | worker | 7331 Mi | 7312 Mi | 99% |
| phoebe | control-plane | 7155 Mi | 7130 Mi | 99% |
| poison | control-plane | 15105 Mi | 15050 Mi | 99% |
| powder | control-plane | 15138 Mi | 14930 Mi | 98% |

> ⚠️ Le cluster reste structurellement saturé. 90/90 Healthy mais sans marge de manœuvre.
