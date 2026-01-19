# ADR-013: Layered Configuration & Disaster Recovery Strategy

**Date:** 2026-01-05
**Status:** Accepted
**Deciders:** User, Coding Agent
**Tags:** backup, disaster-recovery, persistence

---

## Context

### The Problem

Le 2026-01-05, le cluster production a été accidentellement réinitialisé **deux fois**, entraînant:
- Perte de tous les PVCs depuis Kubernetes
- Perte de toutes les applications déployées
- Nécessité de reconstruction manuelle complète

**Malgré cela, certaines données ont survécu:**
- ✅ LUNs iSCSI sur Synology NAS (storage externe)
- ✅ Secrets dans Infisical (externe au cluster)
- ✅ Configurations Terraform (IaC préservé)
- ❌ Configurations applicatives stockées uniquement sur PVCs (perdues)

### Key Insight

**Les applications ont 3 types de données avec des besoins différents:**

1. **Configuration Statique** (reverse proxy, réseau)
   - Ne change jamais ou rarement
   - Peut être versionnée dans Git
   - Doit être restaurée automatiquement

2. **Configuration Dynamique** (personnalisation, secrets intégrés)
   - Modifiée par l'utilisateur ou l'application
   - Contient des secrets/info personnelle (pas dans Git)
   - Doit être backupée et restaurable

3. **État Applicatif** (bases de données, cache)
   - Évolue constamment
   - Critique pour le fonctionnement
   - Doit être backupé en continu

**Sans stratégie unifiée, chaque perte de PVC = intervention manuelle.**

---

## Decision

Nous adoptons une **stratégie de configuration en couches (Layered Configuration)** avec 3 tiers de persistance:

### Tier 1: Configuration Statique (Git)
- **Source de vérité:** Git (ConfigMap/Kustomize)
- **Caractéristiques:** Safe pour versioning, pas de secrets
- **Exemples:** Reverse proxy, réseau, structure applicative de base
- **Pattern:** ConfigMap → InitContainer copie vers PVC

### Tier 2: Configuration Dynamique (Backup/Restore)
- **Source de vérité:** Backup externe (MinIO/S3)
- **Caractéristiques:** Contient secrets/info personnelle, modifiable
- **Exemples:** HomeAssistant config complète, Frigate config.yml
- **Pattern:** MinIO backup → InitContainer restore → PVC writable → Backup continu

### Tier 3: État Applicatif (Backup Continu)
- **Source de vérité:** Backup temps réel (Litestream/rclone)
- **Caractéristiques:** Haute fréquence de changement
- **Exemples:** SQLite databases, cache applicatif
- **Pattern:** PVC → Sidecar backup continu vers MinIO/S3

---

## Architecture Pattern

### Layered Initialization

```yaml
initContainers:
- name: smart-init
  image: alpine:latest
  command:
  - /bin/sh
  - -c
  - |
    set -e
    echo "=== Layered Configuration Init ==="

    # LAYER 1: Base vanilla (TOUJOURS)
    echo "Layer 1: Deploying vanilla base from Git..."
    cp /defaults/configuration-base.yaml /config/configuration.yaml

    # LAYER 2: Restore backup si disponible (OVERRIDE Layer 1)
    if [ -f /backup/configuration.yaml ]; then
      echo "Layer 2: Restoring user configuration from backup..."
      cp /backup/configuration.yaml /config/configuration.yaml
      echo "✅ Configuration restored"
    else
      echo "⚠️  No backup found, using vanilla base"
    fi

    # LAYER 3: Merge secrets depuis Infisical (COMPLEMENT Layer 2)
    if [ -f /secrets/api-keys.env ]; then
      echo "Layer 3: Injecting secrets from Infisical..."
      # Merge/inject secrets into config
      envsubst < /config/configuration.yaml > /tmp/merged.yaml
      mv /tmp/merged.yaml /config/configuration.yaml
      echo "✅ Secrets injected"
    fi

    echo "=== Initialization complete ==="
  volumeMounts:
  - name: defaults          # ConfigMap (Git)
    mountPath: /defaults
  - name: backup-volume     # EmptyDir (PreSync job remplit)
    mountPath: /backup
  - name: secrets           # InfisicalSecret
    mountPath: /secrets
  - name: config-pvc        # PVC writable
    mountPath: /config
```

### Continuous Backup (Sidecar)

```yaml
containers:
- name: app
  # Application principale
  volumeMounts:
  - name: config-pvc
    mountPath: /config

- name: backup-sidecar
  image: rclone/rclone:latest
  command:
  - /bin/sh
  - -c
  - |
    while true; do
      echo "Backing up configuration..."
      rclone sync /config s3:minio/app/config \
        --config /rclone/rclone.conf \
        --exclude "*.log" \
        --exclude ".cache/**"
      sleep 300  # Toutes les 5 minutes
    done
  volumeMounts:
  - name: config-pvc
    mountPath: /config
    readOnly: true
  - name: rclone-config
    mountPath: /rclone
```

### Disaster Recovery Flow

```
┌─────────────────────────────────────────────────────┐
│ Cluster Reset / PVC Loss                            │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────┐
│ ArgoCD PreSync Hook: Restore Job                    │
│ - Fetch backup from MinIO/S3                        │
│ - Place in EmptyDir volume                          │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────┐
│ InitContainer: Layered Init                         │
│ Layer 1: Copy vanilla from ConfigMap                │
│ Layer 2: Restore backup (override vanilla)          │
│ Layer 3: Inject secrets from Infisical              │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────┐
│ Application Starts                                  │
│ - Config complète disponible                        │
│ - Secrets injectés                                  │
│ - État restauré depuis backup                       │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────┐
│ Backup Sidecar: Continuous Backup                   │
│ - Sync config vers MinIO toutes les 5 min           │
│ - Sync DB vers MinIO en temps réel (Litestream)     │
└─────────────────────────────────────────────────────┘
```

---

## Decision Matrix

### Choix du Tier par Type de Données

| Type de Données | Git (Tier 1) | Backup (Tier 2) | Continu (Tier 3) |
|-----------------|--------------|-----------------|------------------|
| **Reverse proxy config** | ✅ ConfigMap | - | - |
| **Network settings** | ✅ ConfigMap | - | - |
| **API keys extractibles** | - | ⚠️ Infisical preferred | MinIO fallback |
| **Config monolithique + secrets** | Base vanilla | ✅ Full backup | - |
| **Info personnelle** (noms, zones) | ❌ Jamais | ✅ Backup only | - |
| **SQLite databases** | - | Initial restore | ✅ Litestream |
| **Fichiers volumineux** (media) | - | - | NFS/iSCSI direct |

### Exemples par Application

| Application | Tier 1 (Git) | Tier 2 (Backup) | Tier 3 (État) |
|-------------|--------------|-----------------|---------------|
| **HomeAssistant** | `http` reverse proxy | `configuration.yaml` complet | `home-assistant_v2.db` |
| **Frigate** | Network defaults | `config.yml` (RTSP URLs) | Clips cache |
| **Hydrus** | - | Settings files | SQLite DB (Litestream) |
| **Mosquitto** | `mosquitto.conf` base | ACL avec users | `persistence.db` |
| **Jellyfin** | Network config | Library metadata | (NFS media) |

---

## Consequences

### Positive

✅ **Disaster Recovery Automatique**
- Cluster reset = apps redéployées avec config complète
- Aucune intervention manuelle (sauf cas extrême)
- RPO < 5 minutes pour configurations
- RPO < 1 minute pour bases de données (Litestream)

✅ **Sécurité Améliorée**
- Zero secrets dans Git public
- Séparation info personnelle / config publique
- Backup chiffré possible (rclone crypt)

✅ **Developer Experience**
- Config vanilla fonctionnelle immédiatement
- Test facile (destroy PVC = test recovery)
- Documentation claire (3 tiers)

✅ **Operational Excellence**
- Pattern réutilisable pour toutes les apps
- Monitoring possible (alertes si backup échoue)
- Conformité GitOps maintenue

### Negative

⚠️ **Complexité Initiale**
- InitContainer plus complexe
- Backup sidecar pour chaque app
- Besoin MinIO/S3 opérationnel

⚠️ **Dépendances Externes**
- Si MinIO down = pas de restore (vanilla fallback)
- Besoin credentials rclone/Litestream
- Storage backup doit être dimensionné

⚠️ **Premier Déploiement**
- Sans backup = vanilla seulement
- Nécessite backup initial manuel
- Migration apps existantes = effort

### Mitigations

**Dépendance MinIO:**
- Vanilla base toujours fonctionnelle
- MinIO HA (déploiement distribué)
- Backup multiple destinations possible

**Premier Déploiement:**
- Script migration pour apps existantes
- Documentation procédure backup initial
- Validation pattern sur apps non-critiques d'abord

**Complexité:**
- Templates réutilisables (Helm charts)
- Documentation claire et exemples
- Validation via dry-run avant déploiement

---

## Implementation Plan

### Phase 1: Documentation & Standards ✅ (Ce document)
- [x] ADR-013 créé
- [ ] Reference: Application Deployment Standard
- [ ] Guide: Backup/Restore Pattern Implementation
- [ ] Templates: InitContainer, Backup Job, Restore Job

### Phase 2: Infrastructure
- [ ] Valider MinIO disponibilité et capacité
- [ ] Créer namespace `backup` pour jobs
- [ ] Déployer rclone config (Secret)
- [ ] Tester backup/restore cycle

### Phase 3: Apps Critiques (Proof of Concept)
- [ ] HomeAssistant: Pattern complet
- [ ] Frigate: Config avec RTSP secrets
- [ ] Mosquitto: ACL + persistence.db

### Phase 4: Rollout Progressif
- [ ] Migrer apps existantes
- [ ] Valider chaque migration
- [ ] Update `docs/applications/*.md`

### Phase 5: Monitoring & Alerting
- [ ] Prometheus metrics (backup success/failure)
- [ ] Alertmanager règles (backup manquant > 1h)
- [ ] Grafana dashboard (backup status)

---

## Success Metrics

**Objectifs à 3 mois:**
- ✅ 100% apps critiques avec pattern layered config
- ✅ Test disaster recovery mensuel (destroy dev cluster)
- ✅ RPO < 5 minutes pour config, < 1 minute pour DBs
- ✅ RTO < 30 minutes pour recovery complet
- ✅ Zero intervention manuelle lors incident

**KPIs:**
- Temps de recovery (target: < 30 min)
- Pourcentage apps avec backup (target: 100% critiques)
- Fréquence backup failures (target: 0%)
- Couverture tests DR (target: monthly)

---

## References

- **Incident:** [INCIDENT-2026-01-05-cluster-reset.md](../INCIDENT-2026-01-05-cluster-reset.md)
- **Pattern inspiration:** Docker layers, Linux distributions
- **Kubernetes patterns:** Init Containers, Sidecars
- **12-Factor App:** Config in environment, stateless processes

---

## Related ADRs

- **ADR-010:** Shared Resources Organization (Proposed)
- **ADR-011:** Namespace Ownership Strategy (Proposed)
- **ADR-012:** Middleware Management (Proposed)

---

**Maintainers:** Infrastructure Team
**Reviewers:** All team members
**Next Review:** 2026-04-05 (3 months)
