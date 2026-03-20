# DataAngel — Rapport de Test d'Intégration

## Session 1 — Tests initiaux

**Date** : 2026-03-19
**Testeur** : Sisyphus (Claude Code)
**Version testée** : `charchess/dataangel:12fb2a9` (pre-v1.0.0)
**Application cible** : mealie (dev cluster, emptyDir, MinIO synelia)
**Issue de suivi** : #2264

---

### Résumé Exécutif

DataAngel est **fonctionnel pour les cas nominaux** et résilient aux pannes S3 en cours de réplication. Trois bugs ont été identifiés, signalés et corrigés pendant la session. Trois points restent ouverts : métriques non fonctionnelles, rclone hang sur bucket vide, et perte de données silencieuse en cas de corruption sans backup.

**Verdict : Prêt pour tests élargis sur d'autres apps, pas encore production.**

---

### Contexte Technique

| Élément | Valeur |
|---------|--------|
| Backend S3 | MinIO (`http://synelia.internal.truxonline.com:9000`) |
| Bucket dev | `vixens-dev-mealie` |
| Storage pod | `emptyDir` (iSCSI indisponible sur dev) |
| App uid | 911 (mealie) |
| Namespace | mealie |

---

### Bugs Identifiés et Corrigés

#### Bug #1 — flag `-endpoint` inexistant dans litestream restore
**Issue** : https://github.com/truxonline/dataAngel/issues/1
**Symptôme** : `flag provided but not defined: -endpoint` → CrashLoopBackOff
**Cause** : `restoreSQLite()` passait `-endpoint` comme flag CLI à `litestream restore`, ce flag n'existe pas.
**Fix** : Génération d'un fichier de config litestream temporaire `/tmp/litestream-restore-N.yml` avec l'endpoint.
**Statut** : ✅ Corrigé et validé

#### Bug #2 — `/etc/litestream` non writable (sidecar)
**Issue** : https://github.com/truxonline/dataAngel/issues/2
**Symptôme** : `mkdir /etc/litestream: permission denied` → sidecar crash immédiat
**Cause** : Le sidecar tentait d'écrire sa config dans `/etc/litestream/` (root-only).
**Fix** : Utilisation de `/tmp/litestream-*.yml` pour les configs générées.
**Statut** : ✅ Corrigé et validé

#### Bug #3 — Pas de détection de corruption SQLite
**Issue** : https://github.com/truxonline/dataAngel/issues/3
**Symptôme** : DB corrompue présente → `database already exists, skipping` → exit 0 → app crash
**Cause** : Flag `-if-db-not-exists` de litestream vérifie seulement la présence, pas l'intégrité.
**Fix** : Vérification SQLite `PRAGMA integrity_check` avant de décider du skip. Si corrompue : suppression + restore.
**Statut** : ✅ Corrigé et validé

---

### Matrice de Tests

#### Init Container

| # | Cas | Résultat | Exit Code | Notes |
|---|-----|---------|-----------|-------|
| 1 | Boot sans backup S3 (bucket vide) | ✅ Skip propre | 0 | `no matching backups found` |
| 2 | Boot avec backup S3 (données présentes) | ✅ Restore WAL complet | 0 | snapshot + WAL segments appliqués |
| 3 | DB corrompue locale + backup S3 | ✅ Détecte, supprime, restaure | 0 | `WARNING: Database exists but is corrupted` |
| 4 | DB corrompue locale + **pas** de backup S3 | ⚠️ Supprime DB, skip restore | 0 | **Perte de données silencieuse** — voir point ouvert #1 |
| 5 | S3 inaccessible au démarrage | ✅ Échoue visiblement | 1 | Timeout DNS ~30s puis exit 1 → CrashLoopBackOff |
| 6 | Multi-paths SQLite (plusieurs chemins) | ✅ Traitement indépendant | 0 | Chaque path itéré séparément |
| 7 | FS paths restore (rclone) | ✅ rclone copy --s3-provider Minio | 0 | Endpoint MinIO correctement appliqué |
| 8 | FS paths + bucket vide (prefix absent) | ⚠️ **Hang** (timeout >20s) | ? | rclone copy ne retourne pas — voir point ouvert #2 |

#### Sidecar

| # | Cas | Résultat | Notes |
|---|-----|---------|-------|
| 9 | Graceful shutdown (SIGTERM) | ✅ WAL flushé avant mort | Nouveau pod restaure les WAL poussés avant SIGTERM |
| 10 | S3 inaccessible pendant réplication | ✅ Sidecar reste UP | `rclone sync error: exit status 1` loggué, pas de crash |
| 11 | Métriques Prometheus (:9090) | ⚠️ Endpoint up, valeurs incorrectes | `litestream_up 0`, `rclone_syncs_failed 0` malgré erreurs — voir point ouvert #3 |

---

## Session 2 — Test v1.0.0 (image `dev`)

**Date** : 2026-03-20
**Testeur** : Claude Code (Opus 4.6)
**Version testée** : `charchess/dataangel:dev` (sha256:20eb50ba..., post-v1.0.0 avec renommage annotations)
**Application cible** : mealie (dev cluster, emptyDir, MinIO synelia)
**PR vixens** : #2317 (renommage annotations `data-guard.io` → `dataangel.io`)

---

### Objectif

Valider le fonctionnement de dataangel avec :
- Le breaking change v1.0.0 : annotations `data-guard.io/*` → `dataangel.io/*`
- L'image `dev` (dernière build main)
- Les fixes précédents (#13 AWS_REGION, #14 rclone provider, #15 annotation rename)

### Changements appliqués (vixens)

| Fichier | Changement |
|---------|------------|
| `apps/_shared/components/dataangel/kustomization.yaml` | `fieldRef` annotations: `data-guard.io/*` → `dataangel.io/*` |
| `apps/10-home/mealie/overlays/dev/patches/dataangel-test.yaml` | Pod annotations: `data-guard.io/*` → `dataangel.io/*` |

Les variables d'environnement (`DATA_GUARD_*`) restent inchangées côté Go.

### Contexte Technique

| Élément | Valeur |
|---------|--------|
| Image | `charchess/dataangel:dev` (sha256:20eb50ba...) |
| Image précédente (pod ancien) | sha256:6e2bd85d... |
| Annotations | `dataangel.io/*` (v1.0.0) |
| Env vars | `DATA_GUARD_*` (inchangées) |
| Kubernetes | v1.34.0 |
| Backend S3 | MinIO (`http://synelia.internal.truxonline.com:9000`) |
| Bucket | `vixens-dev-mealie` |
| Storage pod | `emptyDir` |

### Contenu du bucket S3

```
# Sous data/ (ce que rclone FS tente de restaurer) :
       17 data/.mealie.db-litestream/generation
     4152 data/.mealie.db-litestream/generations/.../wal/00000000.wal
       64 data/.secret
       64 data/.session_secret
    21580 data/mealie.log

# Litestream snapshots/WAL :
   ~27KB × 7 generations (snapshots + WAL segments)

# Ancien format (config/, filesystem/) :
   ~52KB  fichiers config hérités des anciens setups rclone
```

Total bucket : ~30 objets, ~250 KB.

---

### Résultats des Tests

#### Test 1 — Déploiement avec nouvelles annotations

| Étape | Résultat | Notes |
|-------|----------|-------|
| ArgoCD détecte le changement | ✅ | Sync automatique après merge PR #2317 |
| Annotations `dataangel.io/*` sur le pod | ✅ | Vérifiées via `kubectl get pod -o jsonpath` |
| Image `dev` (sha256:20eb50ba) tirée | ✅ | `imagePullPolicy: Always`, nouvelle image |
| Pod rollout (strategy Recreate) | ✅ | Ancien pod terminé proprement |

#### Test 2 — Phase Restore (init container)

| Étape | Résultat | Notes |
|-------|----------|-------|
| Phase restore démarre | ✅ | `phase=restore starting` |
| Config litestream générée | ✅ | `/tmp/litestream-restore-1.yml` correct |
| Litestream restore SQLite | ✅ | DB 1.3MB restaurée depuis snapshot + WAL |
| Rclone FS restore (1er essai) | 🔴 **KILLED après 120s** | `signal: killed` — timeout hardcodé |
| Litestream restore (2ème essai) | ✅ | `database already exists, skipping` |
| Rclone FS restore (2ème essai) | 🔴 **KILLED après 120s** | Même cause |
| Restore (3ème essai) | ✅ | Données accumulées des 2 essais précédents, skip rapide |

**Diagnostic détaillé du kill rclone** : voir [Bug #4](#bug-4--restore-rclone-killed-par-context-timeout-hardcodé-de-2-minutes) ci-dessous.

#### Test 3 — Phase Backup (sidecar)

| Étape | Résultat | Notes |
|-------|----------|-------|
| Lock S3 acquis | ✅ | `Lock acquired, ready for traffic` |
| Litestream replication démarrée | ✅ | Snapshot + WAL écrits |
| Lock renewal | 🟡 **Échecs intermittents** | `context deadline exceeded` sur PutObject S3 pendant ~6 min |
| Rclone backup sync | 🟡 | `signal: terminated` (timeout 3 min sidecar) puis retry OK |
| Stabilisation | ✅ | Après ~6 min, lock renewal et rclone fonctionnent |

**Diagnostic des lock renewals** : voir [Bug #5](#bug-5--lock-renewal-échoue-avec-context-deadline-exceeded-au-démarrage) ci-dessous.

#### Test 4 — Application mealie

| Étape | Résultat | Notes |
|-------|----------|-------|
| Pod final 2/2 Running | ✅ | Après 2 restarts du sidecar natif |
| API mealie accessible | ✅ | `GET /api/app/about` → `{"version":"v3.13.1"}` |
| Données restaurées | ✅ | DB + fichiers utilisateurs présents |

---

### Nouveaux Bugs Identifiés

#### Bug #4 — Restore rclone killed par context timeout hardcodé de 2 minutes

**Sévérité** : 🔴 Haute
**Issue** : https://github.com/truxonline/dataAngel/issues/16
**Reproductible** : Oui (100% sur premier boot avec emptyDir + FS paths)

**Symptôme** :
```
06:49:10 Running: rclone copy :s3:vixens-dev-mealie/data /app/data --s3-env-auth --exclude *.db* --timeout 60s --contimeout 15s --s3-provider Minio
06:51:10 phase=restore failed: failed to restore filesystem /app/data: rclone copy failed: signal: killed
```
Exactement **120 secondes** entre le lancement et le kill.

**Root cause** : Dans `cmd/dataangel/restore.go`, `restoreFilesystem()` utilise :
```go
restoreCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)  // ← hardcodé
defer cancel()
cmd := exec.CommandContext(restoreCtx, "rclone", args...)
// PAS de cmd.Cancel → Go default = SIGKILL (pas SIGTERM)
// PAS de cmd.WaitDelay → kill immédiat
```

**Comparaison avec la phase backup** (qui fonctionne correctement) :

| Propriété | Phase Restore (cassée) | Phase Backup (correcte) |
|-----------|----------------------|------------------------|
| Context timeout | **2 min (hardcodé)** | 3 min (hardcodé) |
| Signal à l'expiration | **SIGKILL** (défaut Go) | SIGTERM (graceful via `cmd.Cancel`) |
| Délai de grâce | **Aucun** | 15s (`cmd.WaitDelay`) |
| Configurable | **Non** | Non |
| rclone --timeout | 60s | 120s |
| rclone --contimeout | 15s | 30s |

**Impact** :
- CrashLoopBackOff au premier boot quand FS restore est configuré
- Le pod finit par se stabiliser après 2-3 restarts (données accumulées entre les tentatives via emptyDir persistant entre restarts du native sidecar)
- Avec PVC, l'impact est limité au tout premier boot
- Non idempotent : chaque restart re-télécharge des fichiers déjà restaurés

**Note** : Le même pattern affecte `restoreSQLite()` (même `context.WithTimeout(ctx, 2*time.Minute)` sans `cmd.Cancel`), mais la DB est plus petite et se restaure en <120s.

**Fix suggéré** :
1. **Timeout configurable** : `DATA_GUARD_RESTORE_TIMEOUT` (défaut 10 min)
2. **Shutdown graceful** : `cmd.Cancel = SIGTERM` + `cmd.WaitDelay = 15s` (comme la phase backup)
3. **Augmenter le défaut** : 2 min est trop agressif pour des restores FS sur S3 lent

#### Bug #5 — Lock renewal échoue avec context deadline exceeded au démarrage

**Sévérité** : 🟡 Moyenne
**Issue** : https://github.com/truxonline/dataAngel/issues/17
**Reproductible** : Oui (au démarrage de la phase backup)

**Symptôme** :
```
06:54:01 Lock acquired, ready for traffic
06:54:01 Starting litestream replicator + rclone sync loop
06:54:41 Failed to renew lock: operation error S3: PutObject, StatusCode: 0, canceled, context deadline exceeded
06:55:11 Failed to renew lock: [même erreur]
... (toutes les 30s pendant ~6 minutes)
06:58:01 [rclone] Command failed: signal: terminated
07:00:05 wal segment written (elapsed=5m1.49s)  ← litestream finit par écrire
```

Les lock renewals échouent systématiquement pendant les premières ~6 minutes, puis se stabilisent.

**Hypothèse** : Au démarrage, litestream snapshot + rclone sync + lock renewal sont tous concurrents sur le même endpoint S3/MinIO. Le PutObject du lock renewal a un timeout de 10s (context interne) qui est insuffisant quand les autres goroutines saturent la bande passante ou les connexions MinIO.

**Impact** :
- Lock non renouvelé pendant 6 min → risque de perte du lock (TTL défaut 60s)
- Si un second pod démarre pendant cette fenêtre, il pourrait acquérir le lock → **split-brain potentiel**
- Le pod reste Ready malgré le lock expiré (readiness probe ne vérifie pas le lock)

**Fix suggéré** :
1. Séquencer le démarrage : acquérir le lock, puis démarrer litestream/rclone, pas en parallèle
2. Augmenter le timeout de renewal (10s → 30s) ou le rendre configurable
3. Considérer un readiness probe qui vérifie aussi la validité du lock

---

### Points Ouverts (mis à jour depuis Session 1)

| # | Problème | Sévérité | Statut Session 1 | Statut Session 2 |
|---|----------|----------|-------------------|-------------------|
| 1 | Perte de données silencieuse (DB corrompue + pas de backup) | 🔴 | Identifié | **Inchangé** — toujours pas corrigé |
| 2 | rclone hang sur prefix S3 vide | 🟡 | Identifié | **Mieux compris** — c'est le timeout hardcodé 2 min (Bug #4) |
| 3 | Métriques non reflétées par les subprocesses | 🟡 | Identifié | **Non testé** cette session |
| 4 | Restore rclone SIGKILL après 2 min | 🔴 | — | **Nouveau** — root cause identifiée dans le code |
| 5 | Lock renewal failures au démarrage | 🟡 | — | **Nouveau** — 6 min d'échecs puis stabilisation |

---

### Points de Design à Suggérer (mis à jour)

#### `runAsUser` configurable par annotation
**Problème** : DataAngel tourne en `uid=1000` mais mealie crée ses fichiers en `uid=911`.
**Fix actuel** (vixens) : `securityContext.runAsUser: 911` hardcodé dans le patch dev.
**Suggestion** : Annotation `dataangel.io/run-as-user: "911"` lisible par le kustomize component.

```yaml
annotations:
  dataangel.io/run-as-user: "911"
  dataangel.io/run-as-group: "911"
```

#### Timeout S3 configurable
**Suggestion** : `DATA_GUARD_S3_TIMEOUT` (défaut `10s`) pour le timeout de connexion S3.

#### Restore timeout configurable
**Suggestion** : `DATA_GUARD_RESTORE_TIMEOUT` (défaut `10m`) pour le timeout de la phase restore.

#### Note sur `DATA_GUARD_FS_PATHS` et le nom du répertoire S3
Le chemin S3 est calculé via `filepath.Base(fsPath)`. Pour `fsPath=/app/data`, le backup va dans `s3://bucket/data/`. L'ancien setup rclone de vixens utilisait `s3://bucket/config/`. Migration non transparente.

---

### État du Cluster Dev Post-Tests

- Pod mealie : **2/2 Running** (mealie + dataangel sidecar natif)
- Restarts : 2 (dus au Bug #4, stabilisé)
- Bucket `vixens-dev-mealie` : actif, litestream WAL streaming OK
- Annotations : `dataangel.io/*` (v1.0.0 compatible)
- Image : `charchess/dataangel:dev` (sha256:20eb50ba...)
- API mealie : **fonctionnelle** (`v3.13.1`)
- Credentials MinIO dev : dans Infisical dev `/apps/10-home/mealie`

---

### Recommandations

#### Pour les développeurs dataangel (issues à ouvrir)

1. **🔴 Bug #4** : Restore timeout hardcodé 2 min + SIGKILL → rendre configurable + SIGTERM graceful
2. **🟡 Bug #5** : Lock renewal failures au démarrage → séquencer ou augmenter timeouts
3. **🔴 Point ouvert #1** : Perte de données silencieuse (DB corrompue + pas de backup) → exit 1
4. **🟡 Point ouvert #3** : Métriques Prometheus non fonctionnelles → instrumenter les goroutines
5. **Design** : `dataangel.io/run-as-user` annotation pour le securityContext
6. **Design** : `DATA_GUARD_RESTORE_TIMEOUT` env var

#### Pour la suite des tests vixens

1. **Tester sur une app avec PVC** (pas emptyDir) pour valider le comportement avec données persistantes
2. **Tester le cas "S3 down au boot + DB locale saine"** (nécessite PVC)
3. **Tester avec `fs-paths` désactivé** (SQLite seul) pour contourner Bug #4 en attendant le fix
4. **Attendre les fixes upstream** avant de considérer un rollout production
5. **Tester les métriques** une fois Bug #4 et #5 corrigés

---

### Verdict Session 2

**Annotations v1.0.0 (`dataangel.io/*`)** : ✅ Fonctionnelles après mise à jour vixens.

**Image `dev`** : ⚠️ Fonctionnelle mais 2 bugs identifiés (restore timeout + lock renewal). Le pod se stabilise après 2-3 restarts, ce qui est acceptable en dev mais **inacceptable en production**.

**Blockers production** :
- Bug #4 (restore SIGKILL) — affecte le premier boot et tout restart avec emptyDir
- Bug #5 (lock renewal) — risque de split-brain pendant la fenêtre de 6 min
- Point ouvert #1 (perte de données silencieuse) — risque de données perdues sans alerte
