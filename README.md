# Vixens

GitOps desired state for the Vixens Kubernetes homelab.

Vixens intentionally contains the Kubernetes/ArgoCD state and the minimum tooling/documentation required to review, validate and operate that state. Machine provisioning and Terraform live in the separate **terravixens** repository.

## Start here

- [WORKFLOW.md](WORKFLOW.md) — canonical development and promotion workflow
- [docs/architecture.md](docs/architecture.md) — current architecture
- [docs/adr/000-index.md](docs/adr/000-index.md) — architecture decision registry
- [AGENTS.md](AGENTS.md) — minimal rules for coding agents

## Repository layout

```text
apps/                  Kubernetes application bases and overlays
argocd/                ArgoCD App-of-Apps definitions
docs/
  adr/                  immutable architecture decision records
  applications/         application runbooks/design notes
  guides/               current operational guides
  post-mortems/         historical incident records
  reference/            stable repository standards
scripts/validation/     repository validation helpers
.github/workflows/      CI, dependency updates and production promotion
justfile                small convenience wrapper around the canonical workflow
```

## Deployment model

```text
short-lived branch
        ↓
       PR
        ↓
      main ───────────────→ ArgoCD dev
        │
        └─ dev-v... snapshot
                 ↓
          promote-prod workflow
                 ↓
            prod-stable
                 ↓
             ArgoCD prod
```

`prod-working` is a manually controlled rescue reference and is never moved automatically.

## Local validation

Required tools are Git, GitHub CLI, Just, yamllint, Kustomize and Python.

```bash
just validate
```

CI repeats validation independently before merge.

## Task tracking

Use GitHub Issues. Convenience commands:

```bash
just gh-resume
just gh-start <issue-number>
just gh-tasks
just gh-done <pr-number>
```

## GitOps rule

Git is the desired-state API. Read-only `kubectl` is useful for diagnosis; direct workload mutation is not the normal way to repair the cluster.
