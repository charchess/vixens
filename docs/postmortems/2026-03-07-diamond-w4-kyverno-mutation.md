# Post-Mortem : Diamond W4 Kyverno Mutation Policy

**Date de l'incident :** 2026-03-07  
**Durée :** ~3h (16:27 → 19:30)  
**Sévérité :** P1 — Multiple apps critiques down  
**Auteur :** Agent Sisyphus  

---

## Résumé exécutif

Le déploiement de la policy Kyverno `mutate-security-context` (Diamond Wave 4) a provoqué une cascade de failures affectant ~20 applications. La policy avait 3 bugs qui se sont cumulés : (1) suppression de CAP_CHOWN sur les init containers root, (2) crash du webhook sur les pods sans init containers, (3) blocage de s6-overlay par `allowPrivilegeEscalation: false`.

**Impact :** HomeAssistant, Jellyfin, les *arr apps, Mealie, et d'autres en CrashLoopBackOff ou ProgressDeadlineExceeded pendant ~3h.

**Résolution :** 6 PRs correctifs (#1912-#1917). La policy Diamond reste active avec les corrections appropriées.

---

## Timeline

| Heure (UTC) | Événement |
|-------------|-----------|
| 16:27:54 | Policy `mutate-security-context` déployée via ArgoCD |
| 16:29:21 | Premiers pods bloqués par le webhook (jellyfin, traefik) |
| 16:35:xx | Apps s6-overlay (mealie, lazylibrarian) commencent à CrashLoop |
| ~16:45 | Init containers `fix-permissions` commencent à échouer (*arr apps) |
| 17:55 | Début investigation incident par agent |
| 18:28 | InfisicalSecrets forcés à resync (73/74 bloqués) |
| 18:36 | PR #1912 mergée — fix init containers root |
| 18:55 | PR #1913 mergée — fix lost+found HA |
| 19:05 | PR #1914 mergée — fix redis-shared NetworkPolicy |
| 19:12 | PR #1915 mergée — fix webhook null crash |
| 19:22 | PR #1916 mergée — bypass s6-overlay HA |
| 19:35 | PR #1917 mergée — bypass s6-overlay batch (4 apps) |
| 19:40 | HomeAssistant 3/3 Running, cluster stabilisé |

---

## Cause racine

**Une seule policy, trois bugs :**

### Bug 1 : Init containers root privés de CAP_CHOWN

```yaml
# La mutation appliquée à TOUS les init containers :
securityContext:
  capabilities:
    drop: [ALL]
```

Les init containers `fix-permissions` tournent en root (`runAsUser: 0`) pour exécuter `chown -R 1000:1000 /config`. Sans `CAP_CHOWN`, même root ne peut pas chowner → `Operation not permitted`.

**Apps affectées :** sonarr, radarr, lidarr, mylar, prowlarr, whisparr, homepage, booklore, vaultwarden

### Bug 2 : Webhook crash sur pods sans init containers

```yaml
# Précondition mal écrite :
key: "{{ request.object.spec.initContainers[] | length(@) }}"
```

Quand `spec.initContainers` est `null` (pas une liste vide, `null`), JMESPath lève une exception. Comme le webhook a `failurePolicy: Fail`, l'exception bloque la création du pod.

**Apps affectées :** jellyfin, traefik, trivy, qbittorrent, pyload, redis-shared, penpot, netbox — tous les pods sans init containers

### Bug 3 : s6-overlay incompatible avec allowPrivilegeEscalation=false

```yaml
# La mutation appliquée à TOUS les containers :
securityContext:
  allowPrivilegeEscalation: false
```

Les images LinuxServer.io utilisent `s6-applyuidgid` (setuid) pour passer de root à l'utilisateur service. Avec `allowPrivilegeEscalation: false`, le setuid échoue.

**Apps affectées :** homeassistant, mealie, lazylibrarian, qbittorrent, pyload

---

## Impact

| Métrique | Valeur |
|----------|--------|
| Apps totalement down | ~15 |
| Apps dégradées | ~7 |
| Durée d'indisponibilité HA | ~3h |
| InfisicalSecrets bloqués | 73/74 |
| PRs correctifs | 6 |

**Apps critiques affectées :**
- HomeAssistant (domotique)
- Jellyfin (media)
- Mealie (recettes)
- *arr stack (media automation)

---

## Résolution

| PR | Description | Approche |
|----|-------------|----------|
| #1912 | Skip mutation init containers root | Précondition `runAsUser != 0` sur le forEach |
| #1913 | HA fix-perms lost+found | Exclure `lost+found` du chown récursif |
| #1914 | redis-shared NetworkPolicy | Ajouter ingress rules manquantes |
| #1915 | Webhook null-safe | `length(initContainers \|\| '[]')` |
| #1916 | HA bypass s6-overlay | `explicitly-allow-root: "true"` |
| #1917 | Batch bypass s6-overlay | Même annotation sur 4 apps |

**La policy Diamond reste ACTIVE.** On n'a pas reculé — on a corrigé les edge cases.

---

## Ce qui a bien fonctionné

1. **Le mécanisme de bypass existait déjà** — `vixens.io/explicitly-allow-root` était documenté dans la policy, on l'a juste appliqué aux apps concernées

2. **ArgoCD + GitOps** — Chaque fix était une PR, revue, mergée, déployée automatiquement. Rollback facile si nécessaire

3. **Infisical operator resilient** — Les secrets existants en cluster n'ont pas été supprimés malgré les erreurs de sync. Les apps déjà démarrées ont continué à fonctionner

4. **Diagnostic rapide** — Les messages d'erreur Kyverno étaient explicites dans les Events des pods

---

## Ce qui a mal fonctionné

1. **Pas de dry-run/staging** — La policy a été déployée directement en prod sans test sur un subset d'apps

2. **Tests insuffisants** — La policy n'a pas été testée contre :
   - Pods sans init containers
   - Init containers root (fix-permissions pattern)
   - Images s6-overlay (LinuxServer.io)

3. **Monitoring/alerting absent** — Pas d'alerte sur le nombre de pods en CrashLoopBackOff ou sur les webhook denials Kyverno

4. **NetworkPolicy oubliée** — `redis-shared` avait `ingress: []` (bloquer tout) — erreur de copier-coller probable

---

## Actions préventives

### Court terme (cette semaine)

- [ ] **Audit des autres apps s6-overlay** — Identifier toutes les images LinuxServer.io et ajouter le bypass proactivement
- [ ] **Documenter le pattern** — ADR sur quand utiliser `explicitly-allow-root`
- [ ] **Fix Kyverno policies similaires** — Vérifier les autres policies pour des JMESPath null-unsafe

### Moyen terme (ce mois)

- [ ] **Alerting Kyverno** — PrometheusRule sur `kyverno_policy_results_total{result="fail"}` spike
- [ ] **Alerting pods** — Alerte si >5 pods en CrashLoopBackOff cluster-wide
- [ ] **Staging policy** — Tester les nouvelles ClusterPolicies en mode `Audit` pendant 24h avant `Enforce`

### Long terme

- [ ] **CI policy testing** — Tests unitaires des policies Kyverno avec `kyverno test` dans la CI
- [ ] **Canary namespace** — Déployer les policies d'abord sur un namespace canary (e.g., `tools`) avant cluster-wide

---

## Leçons apprises

1. **Les mutations Kyverno sont puissantes mais dangereuses** — Elles s'appliquent à la création du pod, pas au YAML. Un bug bloque TOUS les nouveaux pods.

2. **JMESPath est fragile** — `null` vs `[]` fait crasher des expressions qui semblent correctes. Toujours utiliser des fallbacks (`|| '[]'`).

3. **Les images tierces ont leurs contraintes** — LinuxServer.io (s6-overlay), HomeAssistant, et d'autres ont besoin de privilege escalation. C'est légitime et documenté.

4. **Le bypass mechanism doit être planifié** — `explicitly-allow-root` existe, mais on ne l'avait pas appliqué proactivement aux apps connues.

---

## Annexe : Apps nécessitant explicitly-allow-root

| App | Raison |
|-----|--------|
| homeassistant | s6-overlay + hostNetwork |
| frigate | GPU access + hostNetwork |
| gluetun | NET_ADMIN capability |
| mealie | s6-overlay user switching |
| lazylibrarian | s6-overlay user switching |
| qbittorrent | s6-overlay user switching |
| pyload | s6-overlay user switching |
| (à compléter) | ... |

---

## Statut final

- **Cluster :** Stabilisé, 10 apps en Progressing (démarrage) ou Degraded (iSCSI)
- **Policy Diamond :** Active et fonctionnelle
- **Dette technique :** 5 apps bloquées par iSCSI (action utilisateur requise côté Synology)

---

*Post-mortem rédigé le 2026-03-07 par Agent Sisyphus*
