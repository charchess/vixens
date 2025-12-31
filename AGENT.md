# AGENT.md

Guide d'orientation pour agents AI (Gemini, Claude, etc.) travaillant sur le projet Vixens.

---

## 🚨 RÈGLE MAÎTRE

**[WORKFLOW.md](WORKFLOW.md) est la référence ABSOLUE pour le processus de travail.**

Toutes les instructions de ce fichier sont complémentaires et ne doivent JAMAIS contredire WORKFLOW.md.

**En cas de conflit : WORKFLOW.md a toujours raison.**

---

## 📋 Processus de Travail

### Voir WORKFLOW.md pour le processus complet

**Résumé rapide du cycle de travail :**

1. **Initialisation** → Récupérer les tâches (Archon : `find_tasks`)
2. **Sélection** → Priorité : review > doing > todo (toutes assignées à "Coding Agent")
3. **Analyse** → Définir "Definition of Done", consulter docs/applications/<app>.md
4. **Exécution** → Passer en "doing", travailler de manière incrémentale
5. **Validation** → Tester en dev (kubectl + playwright)
6. **Commit/Push** → Git commit + push vers dev UNIQUEMENT
7. **Review** → Passer en "review" + assignee="User"
8. **Boucle** → Retour à WORKFLOW.md

**IMPORTANT :** Lire WORKFLOW.md en entier avant de commencer toute tâche.

---

## 🛠️ Outils Essentiels

### 1. Archon MCP Server (Task & Knowledge Management)
**Système PRIMARY pour la gestion des tâches.**

```bash
# Récupérer les tâches
find_tasks(filter_by="status", filter_value="todo", per_page=20)
find_tasks(filter_by="assignee", filter_value="Coding Agent")

# Démarrer une tâche
manage_task("update", task_id="...", status="doing")

# Rechercher dans la base de connaissances
rag_search_knowledge_base(query="2-5 mots clés", match_count=5)
rag_search_code_examples(query="tech keywords", match_count=3)

# Terminer une tâche
manage_task("update", task_id="...", status="review", assignee="User")
```

**Règles :**
- Toujours rechercher dans RAG AVANT de coder
- Garder les queries courtes (2-5 mots-clés)
- Status flow : `todo` → `doing` → `review` → `done`

### 2. Serena MCP Server (Code Analysis)
**Analyse sémantique et édition de code.**

```bash
# Instructions initiales
mcp__serena__initial_instructions()

# Recherche de code
mcp__serena__find_symbol(name_path_pattern="ClassName/methodName")
mcp__serena__search_for_pattern(substring_pattern="regex")
mcp__serena__get_symbols_overview(relative_path="path/to/file.py")

# Édition de code
mcp__serena__replace_symbol_body(...)
mcp__serena__replace_content(mode="regex", ...)
```

### 3. Playwright (Validation Web)
**Validation des interfaces web après déploiement.**

```bash
# Naviguer et valider
mcp__playwright__browser_navigate(url="https://app.dev.truxonline.com")
mcp__playwright__browser_snapshot()  # Capture d'état
mcp__playwright__browser_click(...)
```

**Fallback :** Si Playwright ne fonctionne pas, utiliser `curl` et informer l'utilisateur.

---

## 📄 Documentation Centralisée dans Archon

**IMPORTANT :** Toute la documentation critique du projet est centralisée dans Archon MCP Server.

### Accès à la Documentation

**15 documents critiques disponibles via Archon :**
- **Processus** : WORKFLOW.md, AGENT.md
- **Guides** : adding-new-application.md, gitops-workflow.md, task-management.md
- **ADRs** : 007-renovate-dev-first, 008-trunk-based-gitops, 009-simplified-two-branch
- **Références** : argocd-sync-waves.md, task-formalism.md, sync-waves-implementation-plan.md
- **Hub** : docs/README.md, GEMINI.md, RESTRUCTURING-COMPLETE.md, adr/README.md

```bash
# Lister les documents du projet
find_documents(project_id="<vixens-project-id>")

# Rechercher un document spécifique
find_documents(project_id="<id>", query="gitops")
find_documents(project_id="<id>", document_type="guide")

# Lire un document complet
find_documents(project_id="<id>", document_id="<doc-id>")
```

### Architecture de Documentation

```
Git Repository (docs/*.md)      Archon MCP (documents DB)
         ↓                                ↑
   Édition humaine              Accès programmatique agents
         ↓                                ↑
    Source de vérité    ←──sync──→  Centralisation MCP
```

**Principe :**
- **Git** = Source de vérité pour édition humaine et versioning
- **Archon** = Centralisation pour accès programmatique par agents (MCP)
- **Synchronisation** = Les deux systèmes restent alignés

**Types de documents dans Archon :**
- `spec` - Spécifications (WORKFLOW.md, ADRs)
- `guide` - Guides pratiques (adding-new-application.md, etc.)
- `note` - Notes techniques et références

**IMPORTANT :** Les documents dans Archon sont distincts du système RAG (39 sources externes). Utilisez `find_documents()` pour la doc projet, `rag_search_knowledge_base()` pour les docs externes (Kubernetes, ArgoCD, etc.).

---

## 📚 Documentation Clé

### Point d'Entrée Principal
**[docs/README.md](docs/README.md)** - Hub central de documentation

### Guides Critiques (TOUJOURS consulter avant de travailler)
1. **[docs/guides/adding-new-application.md](docs/guides/adding-new-application.md)** ⭐
   Guide complet pour déployer une nouvelle application (Kustomize, secrets, ArgoCD)

2. **[docs/guides/gitops-workflow.md](docs/guides/gitops-workflow.md)**
   Workflow trunk-based (dev → main), commits conventionnels, promotion

3. **[docs/guides/task-management.md](docs/guides/task-management.md)**
   Formalism des tâches Archon, système de priorités

### Documentation par Application
**[docs/applications/](docs/applications/)** - Organisée par catégorie

Chaque app a sa doc : `docs/applications/<category>/<app-name>.md`
- Architecture actuelle
- Configuration
- Secrets
- Validation (commandes automatiques + manuelles)
- Troubleshooting

**IMPORTANT :** Mettre à jour la doc de l'app si la config change.

### Références Techniques
- **[docs/reference/argocd-sync-waves.md](docs/reference/argocd-sync-waves.md)** - Sync waves
- **[docs/reference/task-formalism.md](docs/reference/task-formalism.md)** - Formalism des tâches

### ADRs (Architecture Decision Records)
**[docs/adr/](docs/adr/)** - Décisions architecturales importantes

### Procédures Opérationnelles
**[docs/procedures/](docs/procedures/)** - Procédures de déploiement, backup, DR

---

## 🏗️ Structure du Projet

```
vixens/
├── WORKFLOW.md                 # ⭐ PROCESSUS MAÎTRE
├── CLAUDE.md                   # Guidance pour Claude Code (détails étendus)
├── AGENT.md                    # Ce fichier (orientation rapide)
│
├── terraform/                  # Infrastructure as Code
│   └── environments/
│       ├── dev/               # Cluster dev (actif)
│       └── prod/              # Cluster prod
│
├── argocd/                    # ArgoCD self-management
│   ├── base/
│   └── overlays/              # dev, prod
│
├── apps/                      # Applications Kubernetes
│   ├── 00-infra/             # Infrastructure (ArgoCD, Traefik, etc.)
│   ├── 02-monitoring/        # Monitoring (Prometheus, Grafana, etc.)
│   ├── 10-databases/         # Databases (PostgreSQL, Redis)
│   ├── 20-media/             # Media apps (*arr, Jellyfin, etc.)
│   ├── 40-network/           # Network (AdGuard, External-DNS, etc.)
│   ├── 50-services/          # Services (Home Assistant, Vaultwarden, etc.)
│   └── 70-tools/             # Tools (Homepage, Linkwarden, etc.)
│
├── docs/                      # Documentation
│   ├── README.md             # 📚 HUB CENTRAL
│   ├── guides/               # How-to guides ⭐
│   ├── reference/            # Références techniques
│   ├── applications/         # Docs par app (organisées par catégorie)
│   ├── procedures/           # Procédures opérationnelles
│   ├── adr/                  # Architecture Decision Records
│   ├── reports/              # Rapports d'analyse
│   ├── templates/            # Templates de docs
│   └── troubleshooting/      # Incident logs
│
└── scripts/                   # Scripts d'automatisation
```

---

## 🔧 Stack Technique

**OS :** Talos Linux v1.11.0 (immutable, API-driven)
**Kubernetes :** v1.34.0
**GitOps :** ArgoCD v7.7.7 (App-of-Apps pattern)
**CNI :** Cilium v1.18.3 (eBPF, kube-proxy replacement)
**Ingress :** Traefik v3.x
**Storage :** Synology CSI (iSCSI)
**Secrets :** Infisical (self-hosted à http://192.168.111.69:8085)
**LoadBalancer :** Cilium L2 Announcements + LB IPAM

**Environnements :**
- **Dev :** 3 control planes (obsy, onyx, opale) - ✅ ACTIF
- **Prod :** À déployer (Phase 3)

---

## 📝 Notes Techniques Critiques

### Tolérations Control Plane
```yaml
tolerations:
  - key: node-role.kubernetes.io/control-plane
    effect: NoSchedule
```

### Stratégie de Déploiement
- **PVC RWO (ReadWriteOnce)** → `strategy: Recreate`
- **Autres** → `strategy: RollingUpdate`

### Ingress & Certificats
- **Redirection HTTP → HTTPS** : Toujours configurer
- **TLS Dev** : `letsencrypt-staging`
- **TLS Prod** : `letsencrypt-prod`
- **URLs Dev** : `<app>.dev.truxonline.com`
- **URLs Prod** : `<app>.truxonline.com`

### Workflow GitOps (Trunk-Based)
- **2 branches :** `dev` (development) et `main` (production)
- **Branches test/staging :** Archivées (inutiles pour apps, utiles uniquement pour Terraform)
- **Feature branches** → PR vers `dev` → merge
- **Auto-tag** : GitHub Action tag `dev-vX.Y.Z` après merge dans dev
- **Promotion** : `gh workflow run promote-prod.yaml -f version=v1.2.3`
- **Voir :** [ADR-008](docs/adr/008-trunk-based-gitops-workflow.md)

### Principes de Développement
- **DRY (Don't Repeat Yourself)**
- **State-of-the-art best practices**
- **Maintenabilité prioritaire**
- **Non-régression obligatoire**

---

## 🎯 Commandes Essentielles

### Environnement Dev
```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev
export TALOSCONFIG=/root/vixens/terraform/environments/dev/talosconfig-dev

kubectl get nodes
kubectl get pods -A
kubectl -n argocd get applications
```

### Git Workflow
```bash
# Toujours travailler sur dev
git checkout dev
git add .
git commit -m "feat(app): description"
git push origin dev

# Promotion vers prod (via GitHub Actions)
gh workflow run promote-prod.yaml -f version=v1.2.3
```

### Validation
```bash
# Voir docs/applications/<app>.md pour commandes spécifiques

# Validation infrastructure
terraform -chdir=terraform/environments/dev plan  # Doit afficher "No changes"
kubectl get nodes                                  # Tous "Ready"
kubectl -n argocd get applications                 # Tous "Synced + Healthy"
```

---

## ⚠️ Règles Impératives

1. **WORKFLOW.md est MAÎTRE** - Toujours suivre le processus défini
2. **Archon FIRST** - Jamais de TodoWrite, toujours Archon MCP
3. **RAG avant code** - Rechercher dans la base de connaissances avant d'implémenter
4. **Git : dev ONLY** - Jamais de push direct vers main (utiliser GitHub Actions)
5. **Docs à jour** - Mettre à jour `docs/applications/<app>.md` si l'app change
6. **Validation obligatoire** - Tester en dev avant de passer en review
7. **Non-régression** - Exécuter toutes les validations de `docs/applications/<app>.md`
8. **Questions** - Si info manquante ou config externe nécessaire : **SUSPENDRE et demander à l'utilisateur**

---

## 🆘 En Cas de Problème

1. **Outils MCP** :
   - Archon ne répond pas → Rapport à l'utilisateur et arrêt
   - Playwright ne fonctionne pas → Utiliser `curl` et prévenir l'utilisateur
   - Serena : Demander les instructions initiales

2. **Erreurs de déploiement** :
   - Consulter `docs/troubleshooting/`
   - Vérifier `docs/applications/<app>.md` pour troubleshooting spécifique

3. **Doutes** :
   - Relire WORKFLOW.md
   - Chercher dans RAG Archon
   - Consulter ADRs dans `docs/adr/`
   - **Demander à l'utilisateur** si incertitude persiste

---

## 📖 Pour Aller Plus Loin

- **Détails étendus pour Claude Code** → [CLAUDE.md](CLAUDE.md)
- **Documentation complète** → [docs/README.md](docs/README.md)
- **Processus détaillé** → [WORKFLOW.md](WORKFLOW.md)
- **ADRs** → [docs/adr/](docs/adr/)

---

**Dernière mise à jour :** 2025-12-30
**Maintenu pour :** Agents AI (Gemini, Claude, et autres)
