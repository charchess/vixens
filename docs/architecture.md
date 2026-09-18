# Vixens current architecture

This document describes the current architecture of the **Vixens GitOps repository**.
Historical rationale belongs in ADRs and post-mortems.

## Scope

Vixens owns desired Kubernetes application state and ArgoCD application definitions.

It does **not** own machine provisioning, Talos machine configuration or Terraform. Those responsibilities live in the separate `terravixens` repository.

## Delivery architecture

```text
short-lived Git branch
        ↓
GitHub Pull Request
        ↓
required repository validation
        ↓
main
        ↓
ArgoCD development environment
        ↓
runtime validation
        ↓
immutable dev-vYYYY.MM.<PR> snapshot
        ↓
manual promote-prod workflow
        ↓
prod-vYYYY.MM.<PR> + prod-stable
        ↓
ArgoCD production environment
```

Development repository sources track `main`. Production repository sources track `prod-stable`.
The promotion workflow resolves the immutable dev tag to its commit before creating production tags.

`prod-working` is a manually controlled recovery reference and is not updated automatically.

## Kubernetes platform components represented here

The GitOps state includes, among others:

- ArgoCD for reconciliation
- Cilium networking
- Traefik ingress
- cert-manager
- External Secrets Operator
- OpenBao-backed secret materialization
- TrueNAS CSI/NFS storage definitions and node-local storage where appropriate
- Kyverno policy enforcement
- monitoring, autoscaling and application workloads

The exact deployed component set is defined by the active ArgoCD overlay, not by a hand-maintained inventory in this document.

## Secrets

Current secret flow:

```text
OpenBao
   ↓
External Secrets Operator
   ↓
ExternalSecret
   ↓
Kubernetes Secret
   ↓
Workload
```

Application manifests reference the `openbao` `ClusterSecretStore`. Secret values never belong in this repository.

Some filenames still contain the historical word `infisical`; filenames are not authoritative. Active resources must use the External Secrets API.

See [secret management](guides/secret-management.md) and ADR-029.

## Storage

Current GitOps storage definitions are centered on TrueNAS-backed CSI/NFS plus local-path storage for workloads where node-local persistence is intentional.

Older ADRs and post-mortems refer to Synology CSI because that was the platform at the time. Those historical records are intentionally preserved.

ADR-028 records the decision to retire automatic DataAngel restore injection from production workloads.

## Repository safety boundary

Permanent runtime changes are expressed in Git. Read-only cluster inspection is operationally useful, while direct workload mutation is not a replacement for desired state.

ArgoCD itself is currently configured for intentionally unauthenticated administrative access. This is an explicitly accepted homelab risk documented by ADR-030; the repository must not silently reinterpret it as an accidental misconfiguration.

## Documentation architecture

Current rules live in a small set of canonical documents:

- workflow: `WORKFLOW.md`
- architecture: this file
- decisions: `docs/adr/000-index.md`
- operational how-to: `docs/guides/`
- app-specific notes: `docs/applications/`

Third-party agent frameworks are not part of the Vixens architecture.
