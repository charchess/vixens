# Quality Reports & Lint Workflow

**Guide complet pour générer et interpréter les rapports de qualité de l'infrastructure.**

---

## Vue d'ensemble

Le système de rapports Vixens génère plusieurs types de rapports pour assurer la qualité et la conformité de l'infrastructure:

1. **LINT-REPORT.md**: Qualité du code YAML et conformité standards
2. **STATE-ACTUAL.md**: État technique actuel du cluster
3. **CONFORMITY-REPORT.md**: Écarts entre état actuel et désiré
4. **STATUS.md**: Dashboard consolidé multi-environnements

---

## Commande principale: `just lint-report`

### Usage

```bash
# Générer tous les rapports
just lint-report
```

**Ce que ça fait:**

1. ✅ **Lint YAML** (yamllint sur apps/ et argocd/)
2. ✅ **Détection violations DRY** (configurations dupliquées)
3. ✅ **Vérification standards** (ADR-008: resources, labels)
4. ✅ **Génération LINT-REPORT.md** avec score qualité
5. ✅ **Mise à jour STATE-ACTUAL** (dev + prod)
6. ✅ **Mise à jour CONFORMITY** (dev + prod)
7. ✅ **Mise à jour STATUS.md** (dashboard)

**Durée:** ~30-60 secondes (selon taille cluster)

---

## Rapports générés

### 1. LINT-REPORT.md

**Localisation:** `docs/reports/LINT-REPORT.md`

**Contenu:**
- Score de qualité global (0-100)
- Erreurs yamllint (bloquantes)
- Warnings yamllint (non-bloquantes)
- Violations DRY (duplications)
- Violations standards resources (ADR-008)
- Recommandations d'amélioration

**Score de qualité:**
```
100-90: 🟢 Excellent
89-70:  🟡 Good
69-50:  🟠 Fair
< 50:   🔴 Needs Improvement
```

**Calcul du score:**
```
Base: 100 points

Pénalités:
- Erreur yamllint:           -5 points (max -30)
- Warning yamllint:           -2 points (max -20)
- Violation DRY (duplicate):  -10 points (max -30)
- Violation resource:         -5 points (max -20)
```

**Exemple de rapport:**

```markdown
# Lint & Quality Report

**Generated:** 2026-02-08 10:30:00
**Quality Score:** 85/100
**Status:** 🟡 Good

---

## Summary

| Category              | Count | Status |
|-----------------------|-------|--------|
| Total YAML Files      | 247   | ℹ️     |
| Files Passed          | 235   | ✅     |
| Files Failed          | 12    | ❌     |
| Yamllint Errors       | 8     | ❌     |
| Yamllint Warnings     | 15    | ⚠️     |
| DRY Violations        | 3     | ❌     |
| Resource Violations   | 12    | ❌     |

---

## Yamllint Errors

| File                                    | Line | Message                       |
|-----------------------------------------|------|-------------------------------|
| apps/traefik/base/deployment.yaml       | 45   | line too long (125 > 120)     |
| apps/argocd/base/configmap.yaml         | 78   | trailing spaces               |
...

---

## DRY Violations (Duplicated Configs)

### Duplicate Group 1 (3 files)
- `apps/app1/base/ingress.yaml`
- `apps/app2/base/ingress.yaml`
- `apps/app3/base/ingress.yaml`

---

## Resource Standard Violations (ADR-008)

| Resource              | Container | Issue                      | File                        |
|-----------------------|-----------|----------------------------|-----------------------------|
| Deployment/myapp      | main      | Missing resource requests  | apps/myapp/base/deploy.yaml |
...

---

## Recommendations

### 🔴 Critical: Fix Yamllint Errors
- 8 yamllint errors must be fixed
- Run: `just lint` to see all errors

### 🟡 High Priority: Consolidate Duplicates
- 3 duplicate configuration groups found
- Move shared configs to `apps/_shared/`
- Use Kustomize bases/components for reuse

### 🟠 Medium Priority: Add Resource Limits
- 12 containers missing resource specifications
- Follow ADR-008: All containers must have requests + limits
- Use VPA recommendations from Goldilocks
```

---

### 2. STATE-ACTUAL.md

**Localisation:**
- `docs/reports/STATE-ACTUAL-dev.md` (dev cluster)
- `docs/reports/STATE-ACTUAL-prod.md` (prod cluster)
- `docs/reports/STATE-ACTUAL.md` (copie de prod)

**Contenu:**
- État technique complet de toutes les applications
- Resources (CPU/Memory requests/limits)
- VPA recommendations
- Priority classes, sync waves
- Backup profiles (Litestream)
- Issues détectés (OOM risk, CPU throttling)

**Utilisation:**
- Troubleshooting performance
- Capacity planning
- Resource optimization
- VPA analysis

---

### 3. CONFORMITY-REPORT.md

**Localisation:**
- `docs/reports/CONFORMITY-dev.md`
- `docs/reports/CONFORMITY-prod.md`

**Contenu:**
- Comparaison STATE-ACTUAL vs STATE-DESIRED
- Score de conformité par application (0-100)
- Liste des écarts (CPU, Memory, Priority, etc.)

**Statuts:**
- ✅ OK: 100% conforme
- ⚠️ PARTIAL: 70-99% conforme
- ❌ NOK: < 70% conforme
- 🔴 ABSENT: Application manquante

**Exemple:**

```markdown
# Conformity Report

**Total Apps:** 45
- ✅ Compliant: 38
- ⚠️ Partial: 5
- ❌ Non-compliant: 2

## Conformity Details

| App       | Status      | Score  | Issues                              |
|-----------|-------------|--------|-------------------------------------|
| argocd    | ✅ OK       | 100/100| Full compliance                     |
| traefik   | ⚠️ PARTIAL  | 80/100 | CPU Lim mismatch: 500m vs 1000m     |
| myapp     | ❌ NOK      | 50/100 | Missing resource limits             |
```

---

### 4. STATUS.md

**Localisation:** `docs/reports/STATUS.md`

**Contenu:**
- Dashboard consolidé (dev + prod)
- Vue d'ensemble du statut des applications
- Matrice de statut (OK/NOK/Hibernated/Absent)
- Scores de conformité

**Exemple:**

```markdown
# Application Status Dashboard

**Last Updated:** 2026-02-08
**Cluster Environments:** dev, prod

---

## Overview (Prod Cluster)

| Category             | Count | Total |
|----------------------|-------|-------|
| ✅ OK (Functional)   | 38    | 45    |
| ❌ NOK (Broken)      | 2     | 45    |
| 💤 Hibernated        | 5     | 45    |
| ⚪ Absent            | 0     | 45    |
| Total                | 45    | 45    |

---

## Application Status Matrix

| Application | Dev     | Prod    | Conformity                  | Last Change | Note     |
|-------------|---------|---------|----------------------------|-------------|----------|
| argocd      | ✅ OK   | ✅ OK   | [▓▓▓▓▓▓▓▓▓▓] 100%          | 2026-02-08  | -        |
| traefik     | ✅ OK   | ✅ OK   | [▓▓▓▓▓▓▓▓░░] 80%           | 2026-02-07  | CPU lim  |
| myapp       | ✅ OK   | 💤 HIB  | [▓▓▓▓▓░░░░░] 50%           | 2026-02-05  | Missing limits |
```

---

## Workflow de qualité

### Intégration continue

**Dans GitHub Actions:**

```yaml
# .github/workflows/quality-check.yaml
- name: Quality Check
  run: |
    just lint-report || exit 1
```

**Critères de passage:**
- Score qualité >= 50 (configurable)
- Aucune erreur yamllint bloquante
- Conformité >= 70% pour prod

---

### Amélioration continue

**Processus hebdomadaire:**

1. Générer rapport: `just lint-report`
2. Analyser le score et les violations
3. Créer tâches Beads pour corrections
4. Prioriser selon impact:
   - 🔴 Critical: Erreurs yamllint (bloquent CI/CD)
   - 🟡 High: Violations DRY (tech debt)
   - 🟠 Medium: Resources manquantes (stabilité)

**Exemple:**
```bash
# Créer tâche pour fix
bd create --title "fix: corriger violations DRY dans ingress" \
  --type task \
  --priority 2 \
  --description "Consolider 3 ingress dupliqués vers apps/_shared/"
```

---

## Corriger les violations

### Yamllint Errors

**Problème:** `line too long (125 > 120)`

**Solution:**
```yaml
# Avant (trop long)
- name: MY_VERY_LONG_ENVIRONMENT_VARIABLE_NAME
  value: "some very long value that exceeds 120 characters and causes yamllint to complain"

# Après (OK)
- name: MY_VERY_LONG_ENVIRONMENT_VARIABLE_NAME
  value: >-
    some very long value that exceeds 120 characters
    but is now split across multiple lines
```

**Ou désactiver pour ligne spécifique:**
```yaml
some_key: very_long_value  # yamllint disable-line rule:line-length
```

---

### DRY Violations

**Problème:** 3 fichiers identiques d'ingress

**Solution:**

1. Créer base partagée:
```bash
# apps/_shared/components/ingress/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component

resources:
  - ingress.yaml
```

2. Utiliser le composant:
```yaml
# apps/app1/overlays/dev/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

components:
  - ../../../_shared/components/ingress

# Customize avec patches
patches:
  - patch: |-
      - op: replace
        path: /spec/rules/0/host
        value: app1.dev.truxonline.com
    target:
      kind: Ingress
```

---

### Resource Violations

**Problème:** Container sans requests/limits

**Solution:**

1. Consulter VPA (Goldilocks):
```bash
kubectl get vpa -n <namespace>
kubectl describe vpa <app-name>-vpa
```

2. Ajouter resources:
```yaml
# apps/myapp/base/deployment.yaml
spec:
  template:
    spec:
      containers:
        - name: myapp
          resources:
            requests:
              cpu: 100m      # Valeur VPA recommended
              memory: 128Mi
            limits:
              cpu: 500m      # 2-5x requests
              memory: 256Mi  # 1.5-2x requests
```

3. Tester en dev avant prod

---

## Configuration yamllint

**Fichier:** `yamllint-config.yml`

**Règles actuelles:**
```yaml
extends: default

rules:
  line-length:
    max: 120
  indentation:
    spaces: 2
  trailing-spaces: enable
  comments:
    min-spaces-from-content: 1
  document-start: disable  # Pas obligatoire pour Kubernetes
```

**Modifier les règles:**
```yaml
# Désactiver une règle
rules:
  line-length: disable

# Ajuster seuil
rules:
  line-length:
    max: 150
```

---

## Automatisation

### Génération automatique (cron)

**Créer tâche cron:**
```bash
# Générer rapport quotidien
0 2 * * * cd /root/vixens && just lint-report >> /var/log/vixens-lint.log 2>&1
```

### Alerting sur dégradation

**Script de monitoring:**
```bash
#!/bin/bash
# scripts/monitoring/check-quality-score.sh

SCORE=$(grep "Quality Score:" docs/reports/LINT-REPORT.md | awk '{print $3}' | cut -d'/' -f1)

if [ "$SCORE" -lt 70 ]; then
    echo "⚠️ Quality score dropped to $SCORE"
    # Envoyer notification (Slack, email, etc.)
fi
```

---

## Troubleshooting

### Script échoue: "yamllint not found"

**Solution:**
```bash
# Installer yamllint
pip install yamllint

# Ou avec apt
apt-get install yamllint
```

### Script échoue: "kubeconfig not found"

**Solution:**
```bash
# Vérifier kubeconfig
ls -la /root/vixens/.secrets/dev/kubeconfig-dev
ls -la /root/vixens/.secrets/prod/kubeconfig-prod

# Régénérer si manquant (depuis terravixens)
cd terravixens/terraform/environments/dev
terraform output -raw kubeconfig > /root/vixens/.secrets/dev/kubeconfig-dev
```

### Score ne s'améliore pas

**Diagnostic:**
```bash
# Lister tous les fichiers en erreur
just lint 2>&1 | grep "error"

# Compter violations par type
just lint 2>&1 | grep "error" | awk -F'[' '{print $2}' | sort | uniq -c
```

---

## Métriques de qualité

### Objectifs par phase

**Phase 1 - Stabilisation (actuel):**
- Score >= 50 (minimum acceptable)
- Aucune erreur yamllint bloquante
- Resources définies pour apps critiques

**Phase 2 - Amélioration:**
- Score >= 70 (good)
- DRY violations < 5
- Conformité >= 80%

**Phase 3 - Excellence:**
- Score >= 90 (excellent)
- DRY violations = 0
- Conformité >= 95%

---

## Références

- **[ADR-008](../adr/008-resource-profiles.md)**: Resource profiles standards
- **[ADR-020](../adr/020-housekeeping.md)**: Housekeeping policies
- **[docs/reports/README.md](../reports/README.md)**: Reports documentation

---

**Last Updated:** 2026-02-08
