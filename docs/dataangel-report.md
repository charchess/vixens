# DataAngel — Rapport de Test d'Intégration

**Date** : 2026-03-19  
**Testeur** : Sisyphus (Claude Code)  
**Version testée** : `charchess/dataangel:12fb2a9` (dernière image au moment du test)  
**Application cible** : mealie (dev cluster, emptyDir, MinIO synelia)  
**Issue de suivi** : #2264

---

## Résumé Exécutif

DataAngel est **fonctionnel pour les cas nominaux** et résilient aux pannes S3 en cours de réplication. Trois bugs ont été identifiés, signalés et corrigés pendant la session. Trois points restent ouverts : métriques non fonctionnelles, rclone hang sur bucket vide, et perte de données silencieuse en cas de corruption sans backup.

**Verdict : Prêt pour tests élargi sur d'autres apps, pas encore production.**

---

## Contexte Technique

| Élément | Valeur |
|---------|--------|
| Backend S3 | MinIO (`http://synelia.internal.truxonline.com:9000`) |
| Bucket dev | `vixens-dev-mealie` |
| Storage pod | `emptyDir` (iSCSI indisponible sur dev) |
| App uid | 911 (mealie) |
| Namespace | mealie |

---

## Bugs Identifiés et Corrigés

### Bug #1 — flag `-endpoint` inexistant dans litestream restore
**Issue** : https://github.com/truxonline/dataAngel/issues/1  
**Symptôme** : `flag provided but not defined: -endpoint` → CrashLoopBackOff  
**Cause** : `restoreSQLite()` passait `-endpoint` comme flag CLI à `litestream restore`, ce flag n'existe pas.  
**Fix** : Génération d'un fichier de config litestream temporaire `/tmp/litestream-restore-N.yml` avec l'endpoint.  
**Statut** : ✅ Corrigé et validé

### Bug #2 — `/etc/litestream` non writable (sidecar)
**Issue** : https://github.com/truxonline/dataAngel/issues/2  
**Symptôme** : `mkdir /etc/litestream: permission denied` → sidecar crash immédiat  
**Cause** : Le sidecar tentait d'écrire sa config dans `/etc/litestream/` (root-only).  
**Fix** : Utilisation de `/tmp/litestream-*.yml` pour les configs générées.  
**Statut** : ✅ Corrigé et validé

### Bug #3 — Pas de détection de corruption SQLite
**Issue** : https://github.com/truxonline/dataAngel/issues/3  
**Symptôme** : DB corrompue présente → `database already exists, skipping` → exit 0 → app crash  
**Cause** : Flag `-if-db-not-exists` de litestream vérifie seulement la présence, pas l'intégrité.  
**Fix** : Vérification SQLite `PRAGMA integrity_check` avant de décider du skip. Si corrompue : suppression + restore.  
**Statut** : ✅ Corrigé et validé

---

## Matrice de Tests

### Init Container

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

### Sidecar

| # | Cas | Résultat | Notes |
|---|-----|---------|-------|
| 9 | Graceful shutdown (SIGTERM) | ✅ WAL flushé avant mort | Nouveau pod restaure les WAL poussés avant SIGTERM |
| 10 | S3 inaccessible pendant réplication | ✅ Sidecar reste UP | `rclone sync error: exit status 1` loggué, pas de crash |
| 11 | Métriques Prometheus (:9090) | ⚠️ Endpoint up, valeurs incorrectes | `litestream_up 0`, `rclone_syncs_failed 0` malgré erreurs — voir point ouvert #3 |

---

## Points Ouverts

### 🔴 Point ouvert #1 — Perte de données silencieuse (DB corrompue + pas de backup)

**Scenario** : DB locale corrompue, aucun backup en S3.  
**Comportement actuel** : DataAngel supprime la DB corrompue, litestream ne trouve pas de backup (`-if-replica-exists` → exit 0), l'app démarre avec une DB vierge. **Perte totale des données, sans erreur.**  
**Comportement attendu** : Exit 1 pour signaler l'impossibilité de restaurer, forcer l'intervention opérateur.  
**Suggestion** :
```go
// Si DB corrompue ET pas de backup → exit 1, ne pas démarrer l'app avec rien
if corruptedAndNoBackup {
    return fmt.Errorf("DB corrupted and no S3 backup available - manual intervention required")
}
```

### 🟡 Point ouvert #2 — rclone hang sur prefix S3 vide

**Scenario** : `DATA_GUARD_FS_PATHS` défini, bucket existe mais prefix (`/data`) absent.  
**Comportement** : `rclone copy :s3:bucket/data /local` hang indéfiniment (>20s, probablement indéfini).  
**Impact** : Init container jamais terminé → pod bloqué au premier démarrage sur app sans historique FS.  
**Suggestion** : Ajouter un timeout rclone (`--timeout 30s`, `--contimeout 10s`) ou vérifier l'existence du prefix avant de lancer rclone.

### 🟡 Point ouvert #3 — Métriques non reflétées par les subprocesses

**Scenario** : Sidecar up, litestream et rclone en erreur (S3 down).  
**Comportement** : `dataguard_litestream_up 0`, `dataguard_rclone_up 0`, `dataguard_rclone_syncs_failed_total 0` — toujours 0 quel que soit l'état réel.  
**Impact** : Alerting Prometheus inopérant. On ne peut pas détecter une panne de réplication via métriques.  
**Cause probable** : Les métriques sont déclarées mais les goroutines litestream/rclone ne remontent pas leur état au collecteur.  
**Suggestion** : Instrumenter les goroutines subprocess pour updater les gauges en temps réel.

---

## Points de Design à Suggérer

### `runAsUser` configurable par annotation

**Problème rencontré** : DataAngel tourne en `uid=1000` mais mealie crée ses fichiers en `uid=911`. Litestream échoue avec `attempt to write a readonly database`.  
**Fix actuel** (vixens) : `securityContext.runAsUser: 911` hardcodé dans le patch dev.  
**Suggestion** : Ajouter une annotation `data-guard.io/run-as-user: "911"` lisible par le kustomize component pour injecter le securityContext automatiquement, sans toucher aux patches individuels.

```yaml
annotations:
  data-guard.io/run-as-user: "911"
  data-guard.io/run-as-group: "911"
```

### Timeout S3 configurable

**Problème** : S3 inaccessible = timeout DNS ~30s bloquant. Aucun paramètre pour réduire ce délai.  
**Suggestion** : `DATA_GUARD_S3_TIMEOUT` (défaut `10s`) pour contrôler le timeout de connexion S3 dans l'init.

### Note sur `DATA_GUARD_FS_PATHS` et le nom du répertoire S3

Le chemin S3 est calculé via `filepath.Base(fsPath)`. Pour `fsPath=/app/data`, le backup ira dans `s3://bucket/data/`. L'ancien setup rclone de vixens utilisait `s3://bucket/config/`. Migration non transparente — à documenter clairement.

---

## État du Cluster Dev Post-Tests

- Pod mealie : 2/2 Running (mealie + data-guard-sidecar)
- Bucket `vixens-dev-mealie` : actif, WAL streaming en cours
- Config DataAngel : init + sidecar (SQLite + FS paths)
- Credentials MinIO dev : dans Infisical dev `/apps/10-home/mealie`

---

## Recommandation pour la Suite

1. **Ouvrir les issues #1 et #2** sur truxonline/dataAngel (perte silencieuse + rclone hang)
2. **Ouvrir issue #3** sur les métriques (observabilité cassée)
3. **Tester sur une app avec PVC** (pas emptyDir) pour valider le comportement avec données persistantes
4. **Tester le cas "S3 down au boot + DB locale saine"** (non testable en emptyDir, nécessite PVC)
5. Une fois #1 et #2 corrigés : envisager le rollout sur d'autres apps vixens
