# 📝 POST-MORTEM : Cascade Failure suite au changement de mot de passe DSM (PROD)

**Date :** 07 Février 2026
**Statut :** EN COURS DE RÉSOLUTION
**Sévérité :** Critique (S1) - Cascade failure affectant 50+ applications

---

## 1. 📝 RÉSUMÉ EXÉCUTIF

Le changement du mot de passe du compte administrateur Synology DSM sans mise à jour immédiate des credentials du CSI driver a déclenché une cascade de défaillances touchant l'ensemble du cluster de production. L'incident a provoqué des échecs d'authentification iSCSI, des erreurs Multi-Attach sur les volumes, des défaillances temporaires du webhook Kyverno, et a finalement affecté plus de 50 applications dépendantes du stockage persistant ou des bases de données partagées.

**Durée estimée de l'incident :** 2+ heures (en cours)
**Applications impactées :** 27 applications OutOfSync/Degraded/Progressing
**Pods affectés :** 54 pods en état non-Running

---

## 2. ⏳ CHRONOLOGIE DES ÉVÉNEMENTS

- **~10:00** : Changement du mot de passe DSM (compte administrateur ou compte CSI)
- **~10:05** : **Premiers symptômes** : CSI driver commence à échouer l'authentification iSCSI
- **~10:10** : Erreurs de montage de volumes : `Failed to login with target iqn`
- **~10:15** : Cascade sur les bases de données : PostgreSQL, MariaDB, Redis ne peuvent plus monter leurs PVC
- **~10:20** : Applications dépendantes commencent à crasher (authentik, mealie, nocodb, etc.)
- **~10:30** : **Kyverno webhook failure** : Webhook validation échoue avec `connect: operation not permitted`
- **~10:35** : ArgoCD sync failures en cascade : AdGuard, Traefik, external-dns passent OutOfSync
- **~11:00** : Détection de l'incident, début investigation par Gemini
- **~12:00** : Gemini identifie le problème CSI, travaille sur la résolution
- **~12:50** : **Secret Infisical mis à jour** (synology-csi-credentials-sync)
- **~13:00** : CSI driver redémarré, credentials rechargés
- **~13:15** : Kyverno se stabilise automatiquement
- **~13:30** : Début de récupération progressive des applications
- **~14:00** : AdGuard revient Running (3/3), Traefik stabilisé
- **~14:10** : **État actuel** : 27 apps avec problèmes, 54 pods non-Running, récupération en cours

---

## 3. 🔍 ANALYSE DES CAUSES RACINES (Root Causes)

### Cause Racine #1 : Credentials non synchronisés (HUMAIN)

**Problème :** Le mot de passe DSM a été changé sans mise à jour immédiate du secret Infisical.

**Impact :** Le CSI driver a continué à utiliser l'ancien mot de passe, causant des échecs d'authentification iSCSI sur toutes les opérations de montage/démontage de volumes.

**Erreurs observées :**
```
rpc error: code = Internal desc = Failed to login with target iqn
[iqn.2000-01.com.synology:Synelia.pvc-xxx], err: iscsiadm: Timeout on
acquiring lock on DB: /run/lock/iscsi/lock.write: 17: File exists
```

### Cause Racine #2 : Cascade de dépendances (ARCHITECTURAL)

**Problème :** Les applications ont des dépendances hiérarchiques non documentées :
```
CSI Driver
    ↓
Volumes iSCSI (RWO)
    ↓
Bases de données (PostgreSQL, MariaDB, Redis)
    ↓
Applications métier (authentik, mealie, nocodb, linkwarden, etc.)
    ↓
Services frontend/ingress
```

**Impact :** La défaillance du CSI s'est propagée en cascade :
1. CSI ne peut plus attacher/détacher volumes
2. Volumes bloqués avec erreurs Multi-Attach
3. Bases de données ne démarrent pas (I/O errors)
4. Applications dépendantes restent en Progressing/Degraded
5. Contention de ressources (trop de restarts simultanés)

### Cause Racine #3 : Kyverno webhook collatéral (TIMING)

**Problème :** Pendant la cascade de redémarrages, le webhook Kyverno est devenu temporairement indisponible.

**Impact :** ArgoCD ne pouvait plus synchroniser les applications :
```
failed calling webhook "validate.kyverno.svc-fail": failed to call webhook:
Post "https://kyverno-svc.kyverno.svc:443/validate/fail?timeout=10s":
dial tcp 10.100.54.145:443: connect: operation not permitted
```

**Aggravation :** Même après résolution du problème CSI, les applications sont restées OutOfSync à cause du webhook.

### Cause Racine #4 : Absence de procédure documentée (PROCESS)

**Problème :** Aucune procédure documentée pour le changement de mot de passe DSM.

**Impact :** L'opérateur n'a pas été conscient de l'impact critique de cette opération sur le cluster Kubernetes.

---

## 4. 🛡️ ACTIONS CORRECTIVES & PRÉVENTION

### Immédiat (Fait) :
- ✅ **Secret Infisical mis à jour** avec nouveau mot de passe DSM
- ✅ **CSI driver redémarré** pour charger nouveaux credentials
- ✅ **Kyverno stabilisé** automatiquement après quelques minutes
- ✅ **AdGuard récupéré** : Pod Running (3/3)
- ✅ **Traefik récupéré** : 3 pods Running, LoadBalancer fonctionnel

### En cours (14:10) :
- 🔄 **27 applications** encore en OutOfSync/Degraded/Progressing
- 🔄 **54 pods** en état non-Running (Pending, ContainerCreating, Failed)
- 🔄 **Bases de données** : Certaines avec I/O errors, récupération progressive
- 🔄 **Monitoring stack** : Prometheus, Grafana instables

### À court terme (Prochaines 2h) :
1. **Nettoyage pods zombies** : Supprimer pods Failed/Terminated
2. **Force sync ArgoCD** : Forcer synchronisation apps OutOfSync
3. **Résolution ressources** : Identifier et résoudre pods Pending (limites CPU/RAM?)
4. **Stabilisation monitoring** : Prometheus, Grafana, Goldilocks
5. **Validation finale** : Tests ingress, accès WebUI, health checks

### À moyen terme (Cette semaine) :
1. ✅ **Procédure documentée** : Créée dans `docs/procedures/dsm-password-change.md`
2. 📝 **Documentation dépendances** : Cartographie CSI→DB→Apps
3. 📝 **Runbook cascade failure** : Procédure de récupération générique
4. 🔧 **Monitoring amélioré** : Alertes sur CSI authentication failures
5. 🧪 **Test DR** : Valider procédure de récupération en environnement dev

### À long terme (Mois prochain) :
1. **Secret rotation automatisée** : Automation du changement de credentials avec sync Kubernetes
2. **Health checks améliorés** : Détection précoce des failures CSI
3. **Circuit breakers** : Mécanismes pour limiter les cascades
4. **Backup validation** : Tests réguliers de restore depuis Velero
5. **Chaos engineering** : Simulations d'incidents pour valider résilience

---

## 5. 💡 LEÇONS APPRISES

### 1. L'effet domino est réel
Un simple changement de mot de passe sans procédure peut mettre à genoux tout un cluster de production. La profondeur des dépendances (CSI → Storage → Databases → Apps) crée un effet domino difficile à arrêter une fois lancé.

### 2. Les credentials sont critiques
Le secret Synology CSI est un **Single Point of Failure** pour tout le stockage persistant du cluster. Un changement non coordonné équivaut à un crash du storage backend.

### 3. Les webhooks sont fragiles
Les admission webhooks comme Kyverno sont sensibles aux perturbations du cluster. Pendant une cascade failure, ils peuvent aggraver la situation en bloquant la récupération via ArgoCD.

### 4. La documentation sauve des vies
Sans procédure documentée, chaque opérateur doit redécouvrir les impacts et la séquence de récupération. C'est du temps perdu et des risques supplémentaires.

### 5. Le monitoring est aveugle
Nous n'avions pas d'alerte sur :
- CSI authentication failures
- Kyverno webhook availability
- ArgoCD sync failure patterns
- Cascade failure detection

### 6. La récupération est lente
Même après résolution de la cause racine, le cluster met du temps à se stabiliser (30-60 min) à cause des retries, timeouts, et resource contention.

---

## 6. 🎯 INDICATEURS D'IMPACT

| Métrique | Valeur | Note |
|----------|--------|------|
| Durée totale incident | 2h+ (en cours) | Détection à résolution complète |
| Applications impactées | 27/90 (30%) | OutOfSync/Degraded/Progressing |
| Pods affectés | 54 | Non-Running au pic |
| Services critiques down | 3 | AdGuard, Traefik (temporaire), Monitoring |
| Perte de données | 0 | Aucune (volumes préservés) |
| Intervention manuelle requise | Oui | Mise à jour Infisical + restarts |

---

## 7. 📚 DOCUMENTATION CRÉÉE

Suite à cet incident :

1. **[Procédure de changement DSM password](../../procedures/dsm-password-change.md)** ✅
   - Étapes détaillées avec vérifications
   - Impact analysis
   - Troubleshooting guide
   - Rollback procedure

2. **[Infrastructure Dependencies Map](../../reference/infrastructure-dependencies.md)** 🚧
   - Diagramme de dépendances CSI → DB → Apps
   - Impact analysis par composant
   - Points de défaillance critiques

3. **[Cascade Failure Recovery Runbook](../cascade-failure-recovery.md)** 🚧
   - Détection des cascade failures
   - Procédure de récupération générique
   - Checklist de validation

---

## 8. 📞 COMMUNICATION

| Stakeholder | Message | Timing |
|-------------|---------|--------|
| Utilisateurs finaux | ⚠️ Interruption services non-critiques | Début incident |
| Équipe ops | 🚨 Incident S1 en cours, investigation | +30 min |
| Management | 📊 Status update, ETA récupération | +1h |
| Post-mortem | 📝 Rapport complet, actions préventives | Fin résolution |

---

## 9. ✅ VALIDATION DE RÉSOLUTION

L'incident sera considéré **RÉSOLU** quand :

- [ ] Toutes les applications ArgoCD `Synced` et `Healthy`
- [ ] Tous les pods `Running` ou `Succeeded` (sauf jobs terminés)
- [ ] Tests ingress fonctionnels sur services critiques
- [ ] Logs CSI sans erreurs d'authentification
- [ ] Monitoring stack stable (Prometheus, Grafana)
- [ ] Documentation complète publiée

**Date résolution estimée :** 2026-02-07 16:00 (si pas de complications)

---

**Dernière mise à jour :** 2026-02-07 14:15 - Incident en cours de résolution
**Prochain update :** Après Phase 1 du plan (nettoyage + sync)

---

**Fin du rapport provisoire.**
