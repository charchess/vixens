# Vixens 🦊

**Multi-cluster Kubernetes homelab infrastructure following GitOps best practices**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-v1.34.0-blue.svg)](https://kubernetes.io/)
[![Talos](https://img.shields.io/badge/Talos-v1.11.0-orange.svg)](https://www.talos.dev/)
[![ArgoCD](https://img.shields.io/badge/ArgoCD-v7.7.7-green.svg)](https://argo-cd.readthedocs.io/)

---

## 🎯 Quick Start

```bash
# 1. Clone repository
git clone https://github.com/charchess/vixens.git
cd vixens

# 2. Install tools (see Installation section below)
brew install steveyegge/tap/bd just kubectl yamllint

# 3. Read the workflow
cat WORKFLOW.md

# 4. Start working
just resume
```

---

## 📚 Documentation

**START HERE:**
- **[WORKFLOW.md](WORKFLOW.md)** - 🌟 MASTER workflow (Beads + Just) - READ THIS FIRST
- **[AGENTS.md](AGENTS.md)** - Multi-agent guide (Claude, Gemini, etc.)
- **[docs/README.md](docs/README.md)** - Complete documentation hub

**Quick Links:**
- [Adding a new application](docs/guides/adding-new-application.md)
- [Task management](docs/guides/task-management.md)
- [GitOps workflow](docs/guides/gitops-workflow.md)
- [Architecture decisions (ADR)](docs/adr/)

---

## 🏗️ Architecture Overview

### Core Stack

| Component | Version | Purpose |
|-----------|---------|---------|
| **Talos Linux** | v1.11.0 | Immutable, API-driven OS |
| **Kubernetes** | v1.34.0 | Container orchestration |
| **ArgoCD** | v7.7.7 | GitOps continuous delivery |
| **Cilium** | v1.18.3 | CNI with eBPF (kube-proxy replacement) |
| **Traefik** | v3.x | Ingress controller |
| **Synology CSI** | Latest | Persistent storage (iSCSI) |
| **Infisical** | Latest | Secrets management (self-hosted) |

### Multi-Cluster Setup

| Environment | Nodes | VIP | Status |
|-------------|-------|-----|--------|
| **Dev** | daphne, diva, dulce (3 CP HA) | 192.168.111.160 | ✅ Active |
| **Prod** | Physical nodes (3) | 192.168.111.200 | 📅 Phase 3 |📅 Phase 3 |

### Repository Structure

```
vixens/
├── apps/                # Kubernetes applications (Kustomize)
│   ├── 00-infra/       # Infrastructure (ArgoCD, Traefik, Cilium)
│   ├── 10-home/        # Home automation (Home Assistant, Mosquitto)
│   ├── 20-media/       # Media stack (Jellyfin, *arr apps)
│   ├── 40-network/     # Network services (AdGuard, DNS)
│   └── ...
├── argocd/             # ArgoCD App-of-Apps (self-management)
# (Moved to /root/terravixens)
# ├── terraform/          # Infrastructure as Code (Talos clusters)
├── docs/               # Documentation
├── scripts/            # Automation scripts
├── .beads/             # Task management (git-tracked)
└── WORKFLOW.md         # Master workflow (START HERE)
```

---

## 🛠️ Installation

### Prerequisites

- **OS:** Linux (recommended), macOS, or WSL2
- **Architecture:** amd64 or arm64
- **Shell:** bash or zsh

---

## 📦 Required Tools

### 1. Core Workflow Tools

#### Beads (bd) - Task Management

```bash
# macOS (Homebrew)
brew install steveyegge/tap/bd

# Linux (manual install)
curl -sSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash

# Verify
bd --version
```

**What is Beads?**
Task management system with git-native storage (`.beads/` directory). Replaces traditional issue trackers for this project.

#### Just - Workflow Orchestrator

```bash
# macOS (Homebrew)
brew install just

# Linux (debian)
apt install just

# Linux (manual)
curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin

# Verify
just --version
```

**What is Just?**
Command runner (like Make, but better). Orchestrates the workflow defined in `justfile`.

---

### 2. Kubernetes Tools

#### kubectl - Kubernetes CLI

```bash
# macOS (Homebrew)
brew install kubectl

# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# Linux (debian)
apt-get install kubectl

# Verify
kubectl version --client
```

#### Helm - Kubernetes Package Manager

```bash
# macOS (Homebrew)
brew install helm

# Linux
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Verify
helm version
```

#### Kustomize - Kubernetes Configuration Management

```bash
# macOS (Homebrew)
brew install kustomize

# Linux
curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash
sudo mv kustomize /usr/local/bin/

# Verify
kustomize version
```

---

### 3. Talos & Terraform Tools

#### talosctl - Talos CLI

```bash
# macOS (Homebrew)
brew install siderolabs/tap/talosctl

# Linux
curl -sL https://talos.dev/install | sh

# Verify
talosctl version
```

#### Terraform - Infrastructure as Code

```bash
# macOS (Homebrew)
brew tap hashicorp/tap
brew install hashicorp/tap/terraform

# Linux
wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install terraform

# Verify
terraform version
```

---

### 4. YAML & JSON Tools

#### yamllint - YAML Validation

```bash
# macOS (Homebrew)
brew install yamllint

# Linux (pip)
pip3 install yamllint

# Ubuntu/Debian
sudo apt install yamllint

# Verify
yamllint --version
```

#### yq - YAML Processor

```bash
# macOS (Homebrew)
brew install yq

# Linux
wget https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 -O yq
chmod +x yq
sudo mv yq /usr/local/bin/

# Verify
yq --version
```

#### jq - JSON Processor

```bash
# macOS (Homebrew)
brew install jq

# Linux (apt)
sudo apt install jq

# Verify
jq --version
```

---

### 5. Git & GitHub Tools

#### Git - Version Control

```bash
# macOS (Homebrew)
brew install git

# Linux (apt)
sudo apt install git

# Verify
git --version
```

#### GitHub CLI (gh) - PR Management

```bash
# macOS (Homebrew)
brew install gh

# Linux
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
sudo apt update && sudo apt install gh

# Login
gh auth login

# Verify
gh --version
```

---

### 6. Optional Tools (Development)

#### Python Tools (for scripts)

```bash
# Install uv (Python package manager - optional)
curl -LsSf https://astral.sh/uv/install.sh | sh

# Install project dependencies
uv pip install -r scripts/requirements-tests.txt
```

#### Playwright (WebUI Validation - optional)

```bash
# Install Playwright
pip3 install playwright
playwright install chromium

# Verify
playwright --version
```

---

## 🎓 For AI Agents

### Claude Code

**Required MCP Servers:**
- **Serena** - Symbol-aware code editing
- **Archon** - RAG knowledge base
- **Playwright** - WebUI validation

**Configuration:** See [CLAUDE.md](CLAUDE.md) for setup

### Gemini

**Standard tools only** (no MCP servers required)

**Configuration:** See [GEMINI.md](GEMINI.md) for patterns

### Universal Agent Setup

All agents must have access to:
```bash
bd --version      # Beads CLI
just --version    # Just
git --version     # Git
kubectl version   # Kubernetes CLI
yamllint --version # YAML validation
```

**Reference:** [AGENTS.md](AGENTS.md) for multi-agent guide

---

## 🚀 Getting Started

### 1. First Time Setup

```bash
# Clone repository
git clone https://github.com/charchess/vixens.git
cd vixens

# Install tools (see Installation section above)
brew install steveyegge/tap/bd just kubectl yamllint

# Configure git
git config user.name "Your Name"
git config user.email "your.email@example.com"

# Initialize Beads (if needed)
bd init
```

### 2. Access Kubernetes Cluster

```bash
# Set environment variables (example for dev)
export KUBECONFIG=.secrets/dev/kubeconfig-dev
export TALOSCONFIG=.secrets/dev/talosconfig-dev

# Verify access
kubectl get nodes
talosctl version
```

### 3. Start Working

```bash
# Resume work (entry point)
just resume

# Work on a specific task
just work <task_id>

# Manual task operations
bd list --status open
bd show <task_id>
bd update <task_id> --status in_progress
```

---

## 📋 Common Tasks

### Deploy a New Application

```bash
# 1. Create task
bd create "feat: deploy <app-name>" --assignee coding-agent

# 2. Work on task
just work <task_id>

# 3. Create application structure
mkdir -p apps/<category>/<app-name>/{base,overlays/{dev,prod}}

# 4. Follow guide
# See: docs/guides/adding-new-application.md
```

### Update Existing Application

```bash
# 1. Find task or create one
bd list --status open | grep <app-name>

# 2. Edit application
vim apps/<category>/<app-name>/overlays/dev/kustomization.yaml

# 3. Validate
just lint
kustomize build apps/<category>/<app-name>/overlays/dev

# 4. Push
git add . && git commit -m "feat(<app>): description"
git push origin main
```

### Promote to Production

```bash
# 1. Ensure dev is validated
# 2. Use GitHub workflow
gh workflow run promote-prod.yaml -f version=v1.2.3

# 3. Monitor deployment
kubectl -n argocd get applications -w
```

---

## 🔍 Useful Commands

### Task Management

```bash
just resume                       # Entry point (shows current work)
just work <task_id>              # Full workflow orchestration
bd list --status open            # List open tasks
bd list --status in_progress     # Your active work
bd show <task_id>                # View task details
bd close <task_id>               # Mark complete
```

### Kubernetes Operations

```bash
# Cluster status
kubectl get nodes
kubectl get pods -A
kubectl -n argocd get applications

# Talos operations
talosctl version
talosctl health
talosctl dashboard
```

### Validation

```bash
just lint                        # Run yamllint
kustomize build apps/<app>/overlays/dev  # Test build
kubectl apply --dry-run=client -k apps/<app>/overlays/dev  # Dry run
```

### Terraform

```bash
cd /root/terravixens/terraform/environments/dev
terraform plan                   # Show changes
terraform apply                  # Apply changes
```

---

## 🐛 Troubleshooting

### Beads Not Found

```bash
# Check PATH
echo $PATH | grep /usr/local/bin

# Reinstall
brew reinstall steveyegge/tap/bd

# Or manual install
curl -L https://github.com/beadslabs/beads/releases/latest/download/bd-linux-amd64 -o bd
chmod +x bd
sudo mv bd /usr/local/bin/
```

### Kubectl Context Issues

```bash
# List contexts
kubectl config get-contexts

# Switch context
kubectl config use-context <context-name>

# Or set KUBECONFIG
export KUBECONFIG=/path/to/kubeconfig-dev
```

### ArgoCD Sync Issues

```bash
# Check application status
kubectl -n argocd get applications

# Force sync
kubectl -n argocd patch application <app-name> -p '{"operation":{"initiatedBy":{"automated":true},"sync":{"revision":"HEAD"}}}' --type=merge

# View logs
kubectl -n argocd logs deployment/argocd-application-controller
```

### yamllint Errors

```bash
# Check configuration
cat yamllint-config.yml

# Run manually
yamllint -c yamllint-config.yml apps/<app>/**/*.yaml

# Auto-fix (if applicable)
# Most YAML issues require manual fixes
```

---

## 📊 Project Status

- **Phase 1:** ✅ Terraform infrastructure (Completed)
- **Phase 2:** ✅ GitOps with ArgoCD (Active production)
- **Phase 3:** 📅 Production cluster deployment (Planned)

**Current Environment:** Dev cluster fully operational with 50+ applications

**Status Dashboard:** [docs/STATUS.md](docs/STATUS.md)

---

## 🤝 Contributing

### Workflow

1. Read [WORKFLOW.md](WORKFLOW.md) (MASTER workflow)
2. Create task in Beads: `bd create "feat: description"`
3. Work on feature branch
4. Create PR to `dev` branch
5. After merge, promote to `main` via GitHub workflow

### Branch Strategy (Trunk-Based)

```
feature-branch → dev (development) → main (production)
```

**Important:** Never commit directly to `main`. Use promotion workflow.

### Pull Request Guidelines

- Follow conventional commit format: `type(scope): description`
- Include tests and validation
- Update documentation (application docs + STATUS.md)
- Ensure yamllint passes: `just lint`

---

## 📖 Documentation

**Full documentation available in [docs/](docs/):**

- **Guides** - How-to guides (adding apps, gitops, tasks)
- **Reference** - Technical references (sync-waves, kustomize)
- **Applications** - Per-app documentation
- **ADR** - Architecture Decision Records
- **Procedures** - Operational procedures
- **Troubleshooting** - Common issues and solutions

**Key Files:**
- [WORKFLOW.md](WORKFLOW.md) - Master workflow (READ FIRST)
- [AGENTS.md](AGENTS.md) - Multi-agent guide
- [CLAUDE.md](CLAUDE.md) - Claude Code specific
- [GEMINI.md](GEMINI.md) - Gemini specific
- [docs/STATUS.md](docs/STATUS.md) - Quick status dashboard

---

## 🔒 Security

- Secrets managed via Infisical (self-hosted)
- TLS certificates via cert-manager + Let's Encrypt
- Network segmentation via VLANs
- Role-based access control (RBAC)

**Security scanning:** Automated via GitHub Actions (checkov, YAML validation)

---

## 📜 License

This project is licensed under the MIT License - see LICENSE file for details.

---

## 🙏 Acknowledgments

- **Talos Linux** - Immutable Kubernetes OS
- **ArgoCD** - GitOps continuous delivery
- **Cilium** - eBPF-based networking
- **Beads** - Git-native task management
- **Just** - Command runner

---

## 📞 Support

- **Documentation:** [docs/README.md](docs/README.md)
- **Issues:** Use Beads tasks (`bd create`)
- **Workflow Questions:** Read [WORKFLOW.md](WORKFLOW.md)
- **Architecture Decisions:** Check [docs/adr/](docs/adr/)

---

**Last Updated:** 2026-01-08
**Maintained by:** charchess

---

🦊 **Vixens** - GitOps Kubernetes Homelab Done Right
