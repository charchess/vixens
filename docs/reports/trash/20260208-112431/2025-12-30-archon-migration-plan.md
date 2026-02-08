# Plan de Migration et Nettoyage des Tâches Archon
**Date:** 2025-12-30
**Status:** 📋 REVIEW REQUIS
**Tâches totales:** 50 todo, 0 doing

---

## 📊 Statistiques

| Catégorie | Nombre | Action |
|-----------|--------|--------|
| Architecture Cleanup | 8 | ✅ Migrer (nouvellement créées) |
| Déploiements Apps | 8 | ✅ Migrer au nouveau format |
| Fixes Production | 3 | ✅ Migrer |
| Infrastructure | 5 | ✅ Migrer |
| Monitoring/Observability | 7 | ✅ Migrer |
| Security | 3 | ✅ Migrer |
| Documentation/Chore | 4 | ✅ Migrer |
| Research/Evaluation | 3 | ✅ Migrer |
| À Clarifier | 9 | ⚠️ REVIEW + Migrer |
| **TOTAL** | **50** | |

---

## ✅ PHASE 1: Migrations Prioritaires (P0-P1)

### 🔥 Architecture Cleanup (8 tâches - Nouvellement créées)

**Action:** Migrer au nouveau format

| Ancien Titre | Nouveau Titre (Proposé) | task_order | priority | assignee |
|--------------|-------------------------|------------|----------|----------|
| optimization: centralize HTTP redirect middleware | `refactor(tech-debt): centralize http redirect middleware` | 95 | p0 | User |
| optimization: move media namespace out of sabnzbd | `refactor(tech-debt): move media namespace to shared structure` | 90 | p0 | User |
| optimization: factorize *arr config-patcher | `refactor(tech-debt): factorize arr config-patcher scripts` | 88 | p0 | User |
| optimization: factorize *arr deployment-patch | `refactor(tech-debt): factorize arr deployment patches` | 85 | p0 | User |
| optimization: standardize overlay environment strategy | `chore: standardize overlay environment strategy` | 75 | p1 | User |
| optimization: create apps/_shared/ structure | `feat: create apps shared components structure` | 70 | p1 | User |
| optimization: document architecture patterns | `docs: create architecture patterns guide` | 68 | p1 | User |
| optimization: refactor ArgoCD apps structure | `refactor: clarify argocd apps granularity strategy` | 65 | p1 | User |

**Note:** Descriptions déjà conformes au template, juste ajuster titres.

---

### 🐛 Fixes Production (3 tâches - P1)

| Ancien Titre | Nouveau Titre | task_order | priority | assignee | Notes |
|--------------|---------------|------------|----------|----------|-------|
| Fix ArgoCD Server Error | `fix(critical): argocd server crashloop in prod` | 92 | p0 | User | Pod name obsolète? Vérifier si toujours pertinent |
| Fix Linkwarden Errors | `fix: linkwarden database connection errors` | 75 | p1 | User | Vérifier si toujours d'actualité |
| Fix postgresql-shared ArgoCD "Unknown" status | `fix: postgresql-shared kustomize patch error` | 90 | p1 | Coding Agent | Description détaillée OK |

**Action requise:** Vérifier état actuel des pods avant migration.

---

### 🏗️ Infrastructure (5 tâches - P1-P2)

| Ancien Titre | Nouveau Titre | task_order | priority | assignee | feature |
|--------------|---------------|------------|----------|----------|---------|
| Stabiliser Cilium Operator - Ajout Resource Limits | `infra: add cilium operator resource limits` | 85 | p1 | User | infrastructure |
| Align Terraform prod with manual ArgoCD changes | `infra: align terraform state with manual argocd patch` | 85 | p1 | User | infrastructure-drift |
| Implement Backup and Restore Strategy with Velero | `feat: implement velero backup strategy` | 60 | p2 | User | backup-restore |
| Upgrade Terraform Talos provider to v0.10.x | `chore: upgrade terraform talos provider to v0.10.x` | 30 | p2 | User | infrastructure |
| Feature: PostgreSQL backup strategy - S3/MinIO | `feat: implement postgresql backup to s3-minio` | 34 | p2 | User | databases |

---

## ✅ PHASE 2: Déploiements Applications (8 tâches - P2)

| Ancien Titre | Nouveau Titre | namespace | task_order | priority | assignee | feature |
|--------------|---------------|-----------|------------|----------|----------|---------|
| Deploy Firefly III in `finance` namespace | `feat: deploy firefly-iii in finance namespace` | finance | 45 | p2 | Coding Agent | finance-management |
| Deploy qBittorrent in `downloads` namespace | `feat: deploy qbittorrent with gluetun` | downloads | 45 | p2 | Coding Agent | downloads |
| Deploy Calibre-web-automation in `media` | `feat: deploy calibre-web-automation` | media | 45 | p2 | User | media-stack |
| Deploy Freegameclaim in `media` namespace | `feat: deploy freegameclaim` | media | 45 | p2 | Coding Agent | media-stack |
| Deploy PRTG in `tools` namespace | `feat: deploy prtg monitoring tool` | tools | 60 | p1 | User | monitoring |
| Deploy Amule in `downloads` namespace | `feat: deploy amule with gluetun` | downloads | 45 | p2 | Coding Agent | downloads |
| Deploy Pyload in `downloads` namespace | `feat: deploy pyload with gluetun` | downloads | 45 | p2 | Coding Agent | downloads |
| [Monitoring] Deploy Headlamp | `feat: deploy headlamp cluster dashboard` | tools | 55 | p2 | User | monitoring |

**Note:** Toutes ont description minimaliste. À enrichir avec template complet.

---

## ✅ PHASE 3: Monitoring & Observability (7 tâches - P2)

| Ancien Titre | Nouveau Titre | task_order | priority | assignee | feature |
|--------------|---------------|------------|----------|----------|---------|
| Intégrer composants dans Home Assistant | `feat: integrate services with home-assistant` | 50 | p2 | User | monitoring |
| Intégrer composants dans Grafana/Prometheus | `feat: integrate apps with prometheus-grafana` | 50 | p2 | Coding Agent | monitoring |
| Configurer les APIs de Homepage | `chore: configure homepage api integrations` | 55 | p2 | User | monitoring |
| Intégrer applications à gethomepage | `feat: integrate apps with gethomepage annotations` | 55 | p2 | Coding Agent | monitoring |
| Configure Alertmanager webhook URL | `fix: configure alertmanager webhook in infisical` | 90 | p1 | User | monitoring |
| Verify Goldilocks data propagation | `chore: verify goldilocks dashboard data` | 45 | p2 | Coding Agent | monitoring |
| [Monitoring] Create Ingresses for Tools | `feat: create ingresses for headlamp and hubble-ui` | 55 | p2 | User | monitoring |

---

## ✅ PHASE 4: Security (3 tâches - P2)

| Ancien Titre | Nouveau Titre | task_order | priority | assignee | feature |
|--------------|---------------|------------|----------|----------|---------|
| Installer et Configurer CrowdSec | `feat: deploy crowdsec with traefik integration` | 50 | p2 | User | security-hardening |
| Installer Trivy Operator | `feat: deploy trivy-operator for vulnerability scanning` | 50 | p2 | Coding Agent | security-hardening |
| Installer Kyverno (Policy as Code) | `feat: deploy kyverno policy engine` | 50 | p2 | User | security-hardening |

---

## ✅ PHASE 5: Research & Evaluation (3 tâches - P3)

| Ancien Titre | Nouveau Titre | task_order | priority | assignee | feature |
|--------------|---------------|------------|----------|----------|---------|
| Etudier https://github.com/firecrawl/open-scouts | `research: evaluate open-scouts deployment feasibility` | 20 | p3 | Coding Agent | research |
| Evaluation: Vaultwarden PostgreSQL vs SQLite | `research: evaluate vaultwarden postgresql migration` | 25 | p3 | Coding Agent | databases |
| Vérifier Renovate après migration tags | `chore: troubleshoot renovate tag version tracking` | 45 | p2 | Coding Agent | automation |

---

## ✅ PHASE 6: Documentation & Chore (4 tâches - P2-P3)

| Ancien Titre | Nouveau Titre | task_order | priority | assignee | feature |
|--------------|---------------|------------|----------|----------|---------|
| Standardize PVC Naming Convention | `docs: document pvc naming convention` | 25 | p3 | User | documentation |
| Amelioration: Automate PostgreSQL user creation | `feat: automate postgresql user provisioning from infisical` | 40 | p2 | User | databases |
| Configurer Kube Janitor ou Curator | `feat: deploy kube-janitor for resource cleanup` | 45 | p2 | Coding Agent | automation |
| Migrer Archon vers le cluster | `feat: migrate archon to kubernetes cluster` | 20 | p3 | Coding Agent | infrastructure |

---

## ⚠️ PHASE 7: À CLARIFIER (9 tâches)

**Action requise:** Review + Préciser avant migration

### 7.1 - Problèmes Trop Vagues

| ID | Ancien Titre | Problème | Action Recommandée |
|----|--------------|----------|-------------------|
| 5c5fffdb | http://adguard.truxonline.com/ ne marche pas | Pas de détails, description vide | ❓ **Vérifier état actuel** → Migrer ou Archiver |
| f3bb0790 | http://docspell.truxonline.com/ ne marche pas | Pas de détails, description vide | ❓ **Vérifier état actuel** → Migrer ou Archiver |
| f61231ab | reprendre le code terraform pour ne pas détruire cluster | Vague, pas de solution proposée | ❓ **Créer ADR** + Tâche précise |
| c17d9a77 | porter les secrets déployés terraform dans minio | Contexte manquant, objectif peu clair | ❓ **Clarifier objectif** → Reformuler |

**Proposition titres (si pertinents):**
```
fix: adguard ingress accessibility issue
fix: docspell ingress accessibility issue
refactor: prevent terraform cluster recreation on changes
chore: migrate terraform-deployed secrets to minio
```

### 7.2 - Nécessitent Précisions

| ID | Ancien Titre | Problème | Action Recommandée |
|----|--------------|----------|-------------------|
| c8b2ae27 | vérifier cohérence/utilité des infisical | Trop vague, quel scope? | ✏️ **Préciser scope** (quelles apps?) |
| fd77d091 | Restructuration en app/sous apps | Étude de faisabilité?, Impact? | ✏️ **Créer ADR** d'abord |
| fba33ab1 | docspell : configurer file structure | Détails techniques manquants | ✏️ **Ajouter acceptance criteria** |
| 284f642d | importer l'historique birdnet | Prérequis prod, comment? | ✏️ **Définir procédure** |
| fdf3499d | Archiver branches test et staging | Toujours pertinent? workflow changé | ❓ **Review workflow** → Keep ou Delete |

**Propositions reformulées:**
```
chore: audit infisical secrets usage across apps
research: evaluate argocd app-of-apps hierarchical structure
feat: configure docspell nfs integration for document processing
chore: migrate birdnet historical data to production pvc
chore: archive obsolete test-staging branches
```

---

## 🗑️ CANDIDATS SUPPRESSION (À Confirmer)

**Raisons possibles:** Obsolètes, doublons, ou déjà résolus

| ID | Titre | Raison | Décision User |
|----|-------|--------|---------------|
| 05796d0d | Fix ArgoCD Server Error | Pod name `argocd-server-77f7969c77-xhrh4` spécifique, peut être obsolète | ⚠️ Vérifier si toujours en erreur |
| 940dabd5 | Fix Linkwarden Errors | Créé 2025-12-18, toujours d'actualité? | ⚠️ Vérifier état actuel |
| 5c5fffdb | http://adguard.truxonline.com/ ne marche pas | Pas de détails, description vide | ⚠️ Tester URL puis décider |
| f3bb0790 | http://docspell.truxonline.com/ ne marche pas | Pas de détails, description vide | ⚠️ Tester URL puis décider |
| fdf3499d | Archiver branches test et staging | Workflow changé vers trunk-based | ⚠️ Toujours pertinent? |

---

## 📋 Plan d'Exécution

### Étape 1: Review Rapide (15-20 min)

**User valide pour chaque catégorie:**

- [ ] **Architecture Cleanup (8):** Migrer titres + assignee
- [ ] **Fixes Production (3):** Vérifier état pods → Migrer ou Archiver
- [ ] **Infrastructure (5):** Migrer avec nouveau format
- [ ] **Déploiements (8):** Enrichir descriptions + Migrer
- [ ] **Monitoring (7):** Migrer
- [ ] **Security (3):** Migrer
- [ ] **Research (3):** Migrer
- [ ] **Docs/Chore (4):** Migrer
- [ ] **À Clarifier (9):** Décider au cas par cas

### Étape 2: Migration Batch (1-2h)

**Script de migration (pseudo-code):**
```python
for task in tasks_to_migrate:
    new_title = apply_naming_convention(task.title)
    new_description = enrich_with_template(task.description)
    new_metadata = {
        "task_order": map_priority_to_order(task.priority),
        "priority": standardize_priority(task.priority),
        "assignee": fix_assignee_format(task.assignee),
        "feature": determine_feature(task)
    }
    update_task(task.id, new_title, new_description, new_metadata)
```

### Étape 3: Validation (30 min)

**Checklist post-migration:**
- [ ] Toutes tâches todo suivent nouveau format
- [ ] task_order cohérent avec priority
- [ ] Assignee uniquement "User", "Coding Agent", "Archon"
- [ ] feature défini pour epics
- [ ] Descriptions utilisent template markdown
- [ ] Aucun doublon

---

## 🎯 Questions pour User

**Avant de lancer la migration, confirmer:**

1. **Fixes Production:** Vérifier état actuel de:
   - ArgoCD Server (task 05796d0d) → Toujours en erreur?
   - Linkwarden (task 940dabd5) → Toujours en erreur?

2. **URLs cassées:**
   - http://adguard.truxonline.com/ (5c5fffdb) → Tester et décider
   - http://docspell.truxonline.com/ (f3bb0790) → Tester et décider

3. **Clarifications nécessaires:**
   - "reprendre code terraform" (f61231ab) → Quel problème spécifique?
   - "porter secrets dans minio" (c17d9a77) → Objectif?
   - "vérifier infisical" (c8b2ae27) → Scope (toutes apps ou spécifiques)?
   - "Archiver branches test/staging" (fdf3499d) → Toujours pertinent avec trunk-based?

4. **Priorités:**
   - Ordre de migration: P0 d'abord puis P1-P2-P3?
   - Ou migrer tout en une fois?

5. **Assignee:**
   - Confirmer: "User" pour toi, "Coding Agent" pour moi (Claude)?
   - Ou préférer d'autres noms?

---

## 📝 Template Exemple (Migration)

**AVANT:**
```yaml
title: "Deploy Firefly III in `finance` namespace"
description: "- Create Kustomize overlays for the application.\n- Manage secrets via Infisical."
status: todo
assignee: Coding Agent
task_order: 0
priority: medium
feature: null
```

**APRÈS:**
```yaml
title: "feat: deploy firefly-iii in finance namespace"
description: |
  ## Context
  Need personal finance management tool for budget tracking and expense monitoring.

  ## Current State
  - No finance namespace exists
  - No finance apps deployed

  ## Target State
  - Finance namespace created
  - Firefly III deployed with PostgreSQL backend
  - Accessible via https://firefly.dev.truxonline.com

  ## Acceptance Criteria
  - [ ] Create finance namespace with appropriate labels
  - [ ] Deploy PostgreSQL instance for Firefly III
  - [ ] Create Kustomize base and overlays (dev, prod)
  - [ ] Configure Infisical secrets (DB credentials, app key)
  - [ ] Setup Ingress with TLS certificate
  - [ ] Verify WebUI accessibility
  - [ ] Test basic functionality (create account, add transaction)

  ## Dependencies
  - Blocked by: None
  - Blocks: None

  ## Estimated Effort
  M (2-4h)

  ## Impact
  - New namespace: finance
  - New apps: firefly-iii, postgresql
  - Files: +15 (base + 2 overlays)

status: todo
assignee: Coding Agent
task_order: 45
priority: p2
feature: finance-management
```

---

## 🚀 Prochaines Actions

**Immédiat:**
1. ✅ User review ce document
2. ✅ User répond aux questions de clarification
3. ✅ User valide priorités de migration

**Une fois validé:**
1. ✅ Je migre Phase 1 (Architecture Cleanup) en exemple
2. ✅ User review la migration exemple
3. ✅ Je migre toutes les autres phases
4. ✅ Validation finale

---

**Auteur:** Claude Sonnet 4.5
**Status:** Attendu review User
**Date:** 2025-12-30
