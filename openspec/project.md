# Project Context - Vixens

## Purpose
Créer un **homelab résilient** pour gérer HomeAssistant et autres services personnels, tout en permettant d'apprendre et d'appliquer les bonnes pratiques d'entreprise (infrastructure as code, GitOps, sécurité progressive). Le projet a pour vocation d'être à la fois **production-ready** dans son design et **pédagogique** dans son approche, avec un budget nul et un développeur unique.

## Tech Stack

### Infrastructure
- **OS**: Talos Linux v1.7.6 (immutable, pas de SSH par design)
- **Kubernetes**: v1.31.4 (embarqué dans Talos)
- **Provisioning**: Terraform + Talos provider
- **CNI**: Cilium 1.16.5 (avec LB natif, remplace MetalLB)
- **Ingress**: Traefik 33.0.0
- **GitOps**: ArgoCD v7.6.12 (approche **3 layers** : system → security → apps)
- **Storage**: Synology CSI (iSCSI) + NFS (legacy, déprécié)
- **Secrets**: Infisical Operator
- **Certificates**: cert-manager + Gandi webhook (DNS-01 challenge)
- **Kustomize**: Approche **overlays** pour les différences env-to-env

### Hardware
- **Hypervisor**: Hyper-V sur Windows Server 2022 (3 nœuds dev/test/staging)
- **Bare Metal**: NiPoGi mini PCs, Intel N150, 16GB DDR4, 512GB NVMe (3 nœuds prod)
- **NAS**: Synology DS1821+, DSM 7.2.2-72806 Update 4 (iSCSI + MinIO backend S3)
- **Network**: UniFi Dream Machine SE (VLANs 111, 200, 208-210)
- **UPS**: Infosec E3 (actuellement en panne, en remplacement)

### Development Tools
- **AI Assistants**: 
  - **Claude** (Anthropic) - Primary, haute confiance
  - **Cline** (Claude-based IDE) - Secondary, haute confiance
  - **Kimi** (Moonshot AI) - Backup, confiance moyenne-haute
  - **Gemini** (Google) - Rarement utilisé, faible confiance (surtout Flash)
- **Validation**: yamllint, OpenSpec validator, GitHub branch protection
- **Git**: GitHub avec workflow forcé (dev → test → staging → prod)
- **Registry**: Docker Hub (public images uniquement)

## Project Conventions

### Code Style & Philosophy
- **Approche "Fonctionnelle"**: Privilégier la simplicité et le "it works" sur la complexité over-engineered
- **Évolution Progressive**: Phase 1-2 en mode "insecure pour aller vite", Phase 3+ ajout de sécurité (Authentik) et monitoring
- **Pas de "latest"**: Toujours version pinnée pour tout (Helm charts, images Docker, providers Terraform)
- **No Inline**: Toutes les valeurs dans des fichiers `values-{env}.yaml`, jamais de YAML inline
- **DRY**: Un seul module Terraform base, un seul chart Helm par application

### Architecture Patterns
- **Talos Immutable**: Recreate on upgrade, pas de `terraform apply -replace`
- **GitOps 100%**: Rien de manuel sauf bootstrap initial d'ArgoCD
- **3 Layers ArgoCD**: 
  1. **System** (Cilium, cert-manager, storage)
  2. **Security** (Authentik, NetworkPolicies)
  3. **Apps** (HomeAssistant, mail, *arr, etc.)
- **Kustomize Overlays**: Structure `base/` + `overlays/{env}/` pour différencier les environnements
- **No SSH**: Talos ne supporte pas SSH (limitation by design acceptée)
- **Local Services**: HomeAssistant et mail doivent rester accessibles localement si cluster down

### Testing & Validation Strategy
- **yamllint**: Tous les fichiers YAML validés automatiquement dans chaque PR via GitHub Actions
- **OpenSpec Validator**: Toutes les specs doivent passer `openspec validate --strict`
- **Workflow Validation**: GitHub branch protection force la progression linéaire (dev→test→staging→prod)
- **Tests IA**: Les IA génèrent le code, je valide entre les étapes (review manuel avant PR)
- **Tests Utilisateur**: Validation manuelle sur `dev` avant PR vers `test` (ex: curl whoami.dev.truxonline.com)

### Git Workflow

#### Branch Strategy
- **dev**: Branche active de développement (force-push autorisé)
- **test**: Branche de test (PR depuis dev uniquement)
- **staging**: Branche de pré-prod (PR depuis test uniquement)
- **main (prod)**: Branche de production (PR depuis staging uniquement)

#### GitHub Protection Rules (à configurer)
```yaml
# Exemple de configuration
test:
  required_status_checks:
    - yamllint
    - openspec-validate
  requires_linear_history: true
  allow_force_pushes: false
staging & main:
  same as test + requires: 1 review (même si je m'auto-review)
