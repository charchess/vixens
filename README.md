# Vixens 🦊

Vixens est le dépôt GitOps de la plateforme applicative Kubernetes du homelab.

Git décrit l'état désiré, GitHub organise les changements, ArgoCD réconcilie les clusters et les outils d'observabilité permettent de vérifier le résultat.

## Modèle opératoire

```text
GitHub Issue → branche → Pull Request → CI → main → ArgoCD dev
                                              ↓
                                      validation dev
                                              ↓
                                      promote-prod.yaml
                                              ↓
                                        ArgoCD prod
```

La référence canonique est [WORKFLOW.md](WORKFLOW.md).

## Principes

- Git est la source de vérité.
- GitHub Issues est le backlog canonique.
- Toute modification normale passe par Pull Request.
- GitHub Actions porte les validations et promotions.
- ArgoCD déploie depuis Git ; pas de mutation persistante directe du cluster.
- Les outils locaux et assistants IA sont optionnels : ils doivent suivre le même workflow.
- Les secrets ne sont pas stockés en clair dans Git.

## Stack principale

- **Talos / Kubernetes** : infrastructure cluster, gérée séparément dans TerraVixens.
- **ArgoCD** : réconciliation GitOps.
- **Cilium / Hubble** : CNI, sécurité réseau et observabilité réseau.
- **Traefik** : ingress / reverse proxy.
- **cert-manager** : certificats TLS.
- **External Secrets Operator + OpenBao** : projection des secrets vers Kubernetes.
- **VictoriaMetrics / Loki / Grafana** : métriques, logs et visualisation.
- **Renovate** : suivi automatisé des versions.
- **Trivy Operator** : sécurité et vulnérabilités Kubernetes.

## Structure du dépôt

```text
vixens/
├── apps/              # état désiré des applications Kubernetes
├── argocd/            # Applications ArgoCD / app-of-apps
├── docs/              # architecture, guides, ADR, runbooks
├── scripts/           # helpers techniques réutilisables
├── .github/workflows/ # CI, validation et promotion
├── AGENTS.md          # règles complémentaires pour les agents
└── WORKFLOW.md        # workflow canonique
```

L'infrastructure Talos/Terraform vit dans le dépôt **TerraVixens** ; Vixens se concentre sur la couche Kubernetes/GitOps applicative.

## Démarrage rapide

```bash
git clone https://github.com/charchess/vixens.git
cd vixens

# lire les règles avant de modifier
cat WORKFLOW.md

# voir le backlog
gh issue list --state open
```

Avant toute modification, toujours vérifier le `main` courant et les PR ouvertes : plusieurs humains ou agents peuvent intervenir en parallèle.

## Changer une application

```bash
# 1. consulter / créer l'issue
gh issue view <number>

# 2. repartir du main courant
git fetch origin
git switch main
git pull --ff-only
git switch -c fix/<issue>-<slug>

# 3. modifier et valider localement ce qui est pertinent
kustomize build apps/<category>/<app>/overlays/dev >/dev/null

# 4. ouvrir une PR
gh pr create --base main --fill
```

Après merge, ArgoCD synchronise dev depuis `main`.

## Promotion production

Les tags dev sont créés automatiquement après merge. La promotion production se fait uniquement via GitHub Actions :

```bash
gh workflow run promote-prod.yaml -f version=vYYYY.MM.<PR>
```

Le workflow crée un tag immuable `prod-vYYYY.MM.<PR>` et déplace l'alias `prod-stable` suivi par ArgoCD prod.

`prod-working` est un marqueur manuel de dernier état explicitement connu comme fonctionnel ; il n'est jamais déplacé automatiquement.

## Documentation

- [WORKFLOW.md](WORKFLOW.md) — workflow canonique.
- [AGENTS.md](AGENTS.md) — règles pour assistants et agents.
- [docs/README.md](docs/README.md) — index documentaire.
- [docs/guides/task-management.md](docs/guides/task-management.md) — gestion du backlog GitHub Issues.
- [docs/adr/](docs/adr/) — décisions d'architecture.
- [docs/troubleshooting/](docs/troubleshooting/) — runbooks et diagnostics.

Les documents historiques peuvent mentionner des outils retirés. Ils servent d'historique et ne remplacent jamais les documents canoniques ci-dessus.
