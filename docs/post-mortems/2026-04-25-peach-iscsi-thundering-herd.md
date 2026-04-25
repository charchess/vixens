# Post-Mortem: peach iSCSI thundering herd deadlock

**Date:** 2026-04-25  
**Durée:** ~7h (début investigation → cluster fully recovered)  
**Sévérité:** Haute — plusieurs apps critiques indisponibles (mongodb, sakapuss, g4f, netbird)  
**Nœud affecté:** peach (192.168.111.191)

---

## Résumé

Suite à l'incident du 2026-04-24 (crash iSCSI homeassistant/mongodb sur peach), des interventions manuelles dans la session précédente ont déclenché un thundering herd d'appels `iscsiadm sendtargets` sur peach. Les processus iscsiadm se sont accumulés en D-state (bloqués sur I/O réseau), paralysant le plugin CSI pendant plusieurs heures. La résolution a nécessité le reboot de peach, la dépose d'une session NAS stale pour netbird, et le reprovisionnement de son PVC.

---

## Timeline

| Heure | Événement |
|-------|-----------|
| 2026-04-24 23:41 | Crash initial sur peach (incident séparé) |
| Session précédente | Suppression forcée de VolumeAttachments + pods pour migrer les workloads de peach → thundering herd déclenché |
| ~14h00 | Découverte du deadlock : 3+ processus `sendtargets` bloqués sur peach |
| ~14h30 | Identification de la nodeAffinity `required hostname=peach` sur mongodb et sakapuss |
| ~15h00 | PR #3078 mergé : suppression des affinités hardcodées |
| ~15h00 | Taint `iscsi-broken=true:NoSchedule` ajouté sur peach → **aggrave la situation** (mémoire saturée sur autres nœuds) |
| ~15h30 | Taint retiré après constat que le taint crée plus de problèmes qu'il n'en résout |
| ~15h45-16h00 | mongodb, sakapuss, g4f, netbird finissent par se scheduler et démarrer |
| ~16h00 | netbird bloqué : erreur 19 `non-retryable iSCSI login failure` sur pvc-92467dab |
| ~16h30 | Reboot de peach pour remettre iscsid à zéro |
| ~17h00 | Thundering herd rechargé après reboot (13 processus simultanés) |
| ~17h30-20h30 | Processus D-state indestructibles (kill -9 ignoré) — attente du timeout TCP NAS (~30-60min) |
| ~20h55 | Processus D-state se débloquent spontanément (timeout NAS) |
| ~21h00 | Erreur 19 persistante sur netbird → LUN supprimé manuellement dans DSM |
| ~21h10 | PVC/PV netbird-data reprovisionnés → netbird Running ✅ |
| ~21h15 | Nettoyage de 37 PVs local-path orphelins |
| ~21h20 | Cluster fully recovered — 5 nœuds Ready, 0 pod non-Running |

---

## Cause racine

### Cause directe
La suppression simultanée de plusieurs VolumeAttachments + force-delete de pods sur peach dans la session précédente a provoqué plusieurs NodeStageVolume concurrents. Chacun lance un `iscsiadm -m discoverydb --type sendtargets`, déclenchant un **thundering herd** : le NAS ne peut pas répondre à N connexions de discovery simultanées depuis le même initiateur → les processus se bloquent en D-state.

### Cause aggravante
Deux apps (mongodb-shared, sakapuss-backend) avaient une `nodeAffinity: required hostname=peach` hardcodée en Git. Cette concentration volontaire de volumes iSCSI sur un seul nœud le rend particulièrement vulnérable au thundering herd dès qu'une perturbation force la reconnexion simultanée.

### Processus D-state
Une fois en D-state (bloqués sur un socket TCP sans réponse du NAS), les processus `iscsiadm` sont **indestructibles même avec `kill -9`**. La seule issue est :
- Que le NAS réponde (timeout TCP, ~30-60min avec les timeouts iSCSI par défaut)
- Ou reboot du nœud (mais les processus respawnent immédiatement si les pods retentent en masse)

---

## Actions de remédiation

| Action | Résultat |
|--------|----------|
| PR #3078 : suppression `required hostname=peach` de mongodb et sakapuss | ✅ Permanent — évite la concentration future |
| Reboot peach | ✅ Remet iscsid à zéro, mais thundering herd repart si trop de pods reconntectent en même temps |
| Attente timeout TCP NAS (~30-60min) | ✅ Seul moyen de débloquer les processus D-state |
| Suppression LUN netbird + reprovisionnement PVC | ✅ Résout l'erreur 19 de session NAS stale |
| Nettoyage 37 PVs local-path orphelins | ✅ Housekeeping |

---

## Erreurs commises pendant l'incident

1. **Taint prématuré de peach** sans comprendre la cause racine → a concentré la charge sur les autres nœuds jusqu'à saturation mémoire (98-102% requests)
2. **Tentatives de kill répétées** sur des processus D-state → inefficace, perte de temps
3. **Commande `iscsiadm --discover` manuelle** depuis le pod CSI → a ajouté un processus D-state de plus
4. **DSM disable/enable du LUN** sans comprendre que ça ne clôt pas la session iSCSI du Target → n'a pas résolu l'erreur 19

---

## Leçons

- **Ne jamais concentrer plusieurs volumes iSCSI sur un nœud via nodeAffinity** — en cas de perturbation, tous reconntectent simultanément
- **La suppression forcée de VolumeAttachments doit être séquentielle**, un par un, jamais en masse
- **Les processus iscsiadm en D-state ne peuvent pas être killés** — seul le timeout NAS ou le reboot nœud les libère
- **Le reboot d'un nœud avec beaucoup de volumes iSCSI** peut recréer le thundering herd si tous les pods tentent de reconnecter au même moment
- **L'erreur iSCSI 19 "non-retryable"** signale une session stale côté NAS — nécessite une intervention DSM au niveau du Target (pas du LUN)
- **Confirmer les actions avec l'utilisateur** avant d'intervenir sur le cluster en production

---

## Actions préventives

- [ ] Vérifier qu'aucune autre app n'a de `nodeAffinity` hardcodée sur un nœud spécifique
- [ ] Documenter la procédure de recovery iSCSI thundering herd dans `docs/troubleshooting/`
- [ ] Considérer un délai de démarrage aléatoire (`startupDelay`) dans le CSI pour éviter le thundering herd au reboot
