# Post-Mortem : ArgoCD Controller OOMKilled (peach node)

**Date :** 2026-04-18  
**Durée de l'incident :** ~20:00 UTC → ~21:00 UTC (~1h)  
**Cluster :** prod

---

## Résumé

Le contrôleur ArgoCD a été OOMKilled 18 fois sur le nœud `peach` (107% overcommit mémoire). Cela a bloqué la synchronisation de tous les ArgoCD Apps (openclaw n'était pas réconcilié), et par cascade, frigate restait en `Pending` (VPA à 6Gi bloquait le scheduling).

---

## Chronologie

| Heure UTC | Événement |
|-----------|-----------|
| ~20:00 | ArgoCD controller OOMKilled sur peach (exit 137, 18 restarts) |
| ~20:10 | Frigate Pending détecté — VPA requests = 6Gi (trop haut pour les nœuds) |
| ~20:15 | Investigation : peach à 107% overcommit mémoire réel |
| ~20:20 | Taint `memory=pressure:NoSchedule` posé sur peach |
| ~20:25 | Pod controller replanifié sur pearl (3.4 GiB libre), stable |
| ~20:30 | Taint retiré de peach |
| ~20:40 | ArgoCD force-sync déclenché — openclaw réconcilié (PR #3027) |
| ~21:00 | Frigate débloqué (dataAngel sha-2957234, VPA 2Gi) |

---

## Cause racine

**Deux causes combinées :**

1. **Overcommit mémoire sur peach** — 107% d'utilisation réelle. Le nœud avait trop de pods sans `limits` mémoire. L'OOM killer a ciblé argocd-controller (plus gros consommateur sans limits).

2. **VPA openclaw à 6Gi** — Le VPA avait poussé la request mémoire d'openclaw à 6Gi (basé sur l'utilisation de mempalace). Cela saturait peach et bloquait frigate (aucun nœud ne pouvait accommoder 6Gi).

---

## Impact

- ArgoCD ne réconciliait plus aucune application (controller crashait en boucle)
- frigate en `Pending` depuis plusieurs heures
- openclaw bloqué sur une ancienne révision (mempalace encore présent)

---

## Résolution

1. **Taint peach** (`memory=pressure:NoSchedule`) → force la migration du controller vers pearl
2. **Force-sync ArgoCD** pendant la courte fenêtre de stabilité → openclaw réconcilié
3. **PR #3027** — suppression mempalace + VPA réduit à 2Gi
4. **PR #3025** — dataAngel sha-2957234 (fix nil pointer panic iSCSI) → frigate débloqué

---

## Actions correctives

| Action | Statut |
|--------|--------|
| Supprimer mempalace d'openclaw | ✅ PR #3027 mergé |
| Réduire VPA cap openclaw à 2Gi | ✅ PR #3027 mergé |
| Ajouter limits mémoire sur argocd-controller | ⬜ À faire |
| Surveiller overcommit peach | ⬜ Alerte Grafana à créer |

---

## Leçons

- Les pods sans `limits` mémoire sont vulnérables à l'OOM killer en cas de pression
- Le VPA sans `maxAllowed` peut exploser les requests et bloquer le scheduling
- Un taint temporaire est une technique rapide et réversible pour forcer la migration d'un pod critique
