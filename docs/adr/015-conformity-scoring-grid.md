# ADR-008: Conformity Scoring Grid Rationale

**Date:** 2025-11-26
**Status:** Deprecated
**Deciders:** User, Coding Agent
**Tags:** quality, conformity

---

> **Note Historique :** Ce document a été restauré pour archive. 
> Son statut a été mis à jour pour refléter l'architecture actuelle.

---

## Contexte

Le projet Vixens utilise **Conformity**, un agent d'audit de code qui évalue la qualité et la cohérence du code sur une échelle de 100 points. Cette notation est cruciale pour :
- Bloquer les déploiements de code de mauvaise qualité (<70 points)
- Forcer la cohérence multi-environnement (dev/test/staging/prod)
- Détecter les secrets exposés avant commit
- Garantir l'adhérence aux standards Kubernetes

La grille de notation initiale utilisait des pénalités "au doigt mouillé", ce qui posait plusieurs problèmes :
- Manque de transparence dans les décisions
- Pénalités non graduées (même pénalité pour erreur mineure vs critique)
- Pas de différenciation dev vs prod
- Subjectivité dans l'évaluation

## Décision

**Adopter une grille de notation graduée et justifiée avec pénalités différenciées par criticité et environnement.**

### Grille Complète (100 Points)

#### 1. Hygiène & Sécurité (30 pts) - GATEKEEPERS

**Syntaxe & Validation (15 pts)** :
- **-2** par warning kubeconform (API deprecated, non-breaking)
  - *Justification* : Erreur mineure, pas de blocage immédiat mais nécessite correction future
- **-5** par erreur validation (invalid field, type mismatch)
  - *Justification* : Erreur moyenne, peut causer problèmes à runtime
- **-10** par erreur critique (missing required field, invalid resource)
  - *Justification* : Erreur majeure, déploiement impossible ou instable

**Secret Safety (15 pts)** :
- **-3** par secret test/dev (dummy_api_key, test_password)
  - *Justification* : Secret non sensible mais mauvaise pratique
- **-5** par secret API/token non critique
  - *Justification* : Risque moyen, exposition limitée
- **-10** par secret critique (AWS/GCP/SSH keys, database passwords)
  - *Justification* : Risque majeur, compromission totale possible
- **-15** par secret prod hardcodé (blocage automatique)
  - *Justification* : Inacceptable, même avec --force

**Rationnel du seuil 30 pts** : Hygiène et sécurité sont les gatekeepers fondamentaux. Un score <15/30 indique des problèmes critiques de sécurité ou syntaxe.

#### 2. Architecture & Cohérence (30 pts) - CŒUR GITOPS

**Iso-Environnement (20 pts)** :
- **+20** : 4/4 environnements homogènes (tous Kustomize OU tous Helm)
  - *Justification* : Cohérence parfaite, maintenabilité maximale
- **+10** : 3/4 environnements homogènes + DT créée + plan correction <30j
  - *Justification* : Cohérence acceptable avec dette technique trackée
- **+0** : 2/4 ou moins homogènes (blocage sauf --force)
  - *Justification* : Chaos architectural, dérive des environnements

**Rationnel** : L'iso-environnement est le principe #1 du GitOps dans Vixens. Permettre dev en Helm et prod en Kustomize crée :
- Incohérence des configs
- Bugs non reproductibles entre env
- Overhead de maintenance (2 systèmes à maîtriser)

**Philosophie & Patterns (10 pts)** :
- **+10** : Code ressemble à l'existant (labels, structure, conventions)
- **-5** : Nouveau pattern sans ADR ou ticket DT créé
  - *Justification* : Innovation non documentée = dette technique future

#### 3. Maintenabilité (20 pts)

**Atomicité (10 pts)** :
- **< 100 lignes** : +10 (idéal, lisible en un écran)
- **101-150 lignes** : +5 (acceptable mais limite)
- **> 150 lignes** : 0 (fichier trop complexe, doit être splitté)

**Rationnel** : Objectif Vixens = manifests < 100 lignes. Au-delà, la complexité augmente exponentiellement, le debugging devient difficile.

**Intention (10 pts)** :
- **+2** par commentaire pertinent (explique le "pourquoi", pas le "quoi")
- **Max 5 commentaires** = 10 pts

**Rationnel** : Code self-documenting > comments, mais les décisions business/techniques doivent être explicitées.

#### 4. Best Practices (10 pts)

**Pénalités graduées par environnement** :

**Dev** :
- `-1` par manquement critique (runAsNonRoot, memory limits)
- `-2` par manquement important (probes, requests)
- `-1` par manquement mineur (labels recommandés)

**Prod** :
- `-3` par manquement critique
- `-2` par manquement important
- `-1` par manquement mineur

**Rationnel** : Dev = apprentissage, erreurs tolérées. Prod = zéro compromis sur sécurité/stabilité.

#### 5. Veille & Documentation (10 pts)

- **+10** : Mention explicite de la documentation consultée (Context7, Archon RAG, ADR)
- **+5** : Documentation partielle
- **+0** : Pas de documentation

**Rationnel** : Traçabilité des décisions techniques, preuve que le code s'appuie sur docs officielles.

### Seuils de Décision

- **PASS (≥85)** : Code production-ready, aucune restriction
  - *Justification* : 15 points de marge = tolère quelques imperfections mineures
- **WARN (70-84)** : Code acceptable, flaggé pour amélioration future
  - *Justification* : Déploiement autorisé mais dette technique créée
- **FAIL (<70)** : Code bloqué sauf --force avec justification
  - *Justification* : Qualité insuffisante, risque trop élevé

## Alternatives Évaluées

### 1. Notation Binaire (PASS/FAIL)
- ✅ Simple
- ❌ Pas de granularité, pas de feedback constructif
- ❌ Décision brutale, pas de zone grise

### 2. Notation sur 10 (A-F style)
- ✅ Familier (notation scolaire)
- ❌ Pas assez granulaire pour 5 catégories
- ❌ Pénalités difficiles à calibrer

### 3. Notation sur 100 Graduée (CHOISI)
- ✅ Granularité fine (pénalités -1 à -15)
- ✅ Feedback détaillé par catégorie
- ✅ Seuils ajustables (85/70)
- ✅ Différenciation dev vs prod

## Conséquences

### Positives
- ✅ **Transparence** : Chaque pénalité est justifiée et expliquée
- ✅ **Prévisibilité** : Développeurs savent à l'avance comment sera évalué leur code
- ✅ **Amélioration continue** : Feedback actionnable (score par catégorie)
- ✅ **Différenciation contexte** : Dev plus permissif que prod (apprentissage)
- ✅ **Force la discipline** : Seuil <70 bloquant encourage la qualité

### Négatives
- ⚠️ **Calibration initiale** : Peut nécessiter ajustements après quelques sprints
  - **Mitigation** : Revue de la grille tous les 3 sprints, ajustements basés sur feedback
- ⚠️ **Subjectivité résiduelle** : "Commentaire pertinent" reste subjectif
  - **Mitigation** : Agent Conformity documente la raison de chaque point attribué
- ⚠️ **Bypass facile avec --force** : Risque de contournement habituel
  - **Mitigation** : --reason obligatoire, dette Archon créée automatiquement

## Métriques de Succès

**À mesurer sur 3 sprints (Sprint 7-9)** :
1. **Score moyen** : Doit être ≥80 après période d'apprentissage
2. **Taux de FAIL** : Doit descendre sous 20% après Sprint 8
3. **Utilisation --force** : Doit être <10% des commits
4. **Dettes techniques créées** : Doivent être résolues dans les 30j

**Revue grille** : Si les métriques dévient, ajuster les pénalités par vote (exemple : secret test -3 → -2 si trop punitif).

## Références

- [Conformity Agent](.claude/agents/conformity.md) - Implémentation complète
- [PROJECT.md](.claude/PROJECT.md) - Workflow Vixens
- [ADR-002: ArgoCD GitOps](002-argocd-gitops.md) - Contexte GitOps
- [ADR-007: Infisical Secrets](007-infisical-secrets-management.md) - Gestion secrets automatisée

## Révisions

- **2025-11-26** : Version initiale (Sprint 6+)
  - Graduation des pénalités (kubeconform -2/-5/-10 au lieu de -3 flat)
  - Différenciation dev/prod pour Best Practices
  - Ajout catégorie secrets graduée (test -3, prod -15)
  - Iso-environnement clarification 4/4 (+20), 3/4 (+10), 2/4 (+0)
