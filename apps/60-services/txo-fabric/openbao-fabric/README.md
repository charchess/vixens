# OpenBao Fabric — SOURCE-ONLY foundation (#4046)

**Status: CONTRACT ONLY / NOT DEPLOYABLE.** This directory has **no**
`kustomization.yaml`, Kubernetes manifest or ArgoCD Application. It is
deliberately **not referenced** by any production/dev overlay. It must
not be rendered, installed, applied or interpreted as an initialized
OpenBao service.

[ADR-041](../../../../docs/adr/041-separate-openbao-fabric-secret-plane.md)
records the approved architectural separation. The
[operator runbook](../../../../docs/troubleshooting/txo-fabric-openbao-bootstrap-and-restore.md)
defines separately approved physical work.

## Authority and canonical naming

- **Infra OpenBao:** `ClusterSecretStore/openbao` under
  `apps/00-infra/openbao`; preserved exactly, including existing
  endpoint and ESO token-secret reference. It owns infrastructure
  provisioning/repair/administration/backup/restore. It retains
  **external** authority over Fabric break-glass bootstrap and recovery.
- **Fabric OpenBao:** future instance **`openbao-fabric`**; future
  dedicated ESO store **`openbao-fabric-store`**; only credentials
  consumed by Fabric runtime workloads (platform/tenant).
- **No existing automatic migration.** Active OpenFGA ESO
  `openfga/runtime/external-secrets.yaml` still uses Infra
  `ClusterSecretStore/openbao`; staged WebUI
  `webui/bootstrap/credentials.yaml` does too. Do NOT flip references
  before a physically verified Fabric replacement and a reviewed
  operational migration. New Fabric consumers must **never** use the
  Infra store.

## Source layout

- `contracts/authority-and-manifest-envelope.md` — proposed
  object/namespace/ServiceAccount/private network/storage/ESO shape,
  each unresolved decision explicitly marked **BLOCKED**.
- `contracts/workload-authz-matrix.md` — read-only workload path
  matrix, policy ownership, denied capabilities and tenant isolation.
- `policies/*.hcl.example` — **reference-only** KV v2 capability
  snippets. They are **not** installable policy artifacts: do not run
  `bao policy write` on these files or script substitution of tenant
  IDs without a separately authorized provisioner and review. No
  root/unseal/recovery/administration capability is expressed.
- `scripts/validation/validate_openbao_fabric_source.py` at repo root
  — negative assertion guarding this source-only boundary.

## Invariants

1. No Fabric OpenBao `StatefulSet`, `Deployment`, `Service`,
   `Secret`, `ExternalSecret`, `SecretStore`,
   `ClusterSecretStore`, namespace/PVC/PV or Argo Application is
   declared or wired by this directory.
2. No stored credential value or encoded/materialized secret is in
   Git, GitHub Issue/PR or CI logs. Kubernetes authenticator, policies,
   issuer and audience remain configuration **decisions**, not
   values guessed by an agent.
3. Every tenant workload receives only its **immutable one-tenant**
   path `fabric/tenants/<tenant-id>/*`; no sibling enumeration or
   wildcard scope `fabric/tenants/*`.
4. Fabric application identities cannot create tokens/policies/mounts,
   read audit logs, administer Fabric OpenBao, or initialize/unseal.
5. Default deny for both ingress and egress; permit only reviewed,
   exact endpoints after actual cluster verification. No Internet
   catch-all or public OpenBao route.
6. Stateful data must use actual storage with **Retain** reclaim
   semantics and verified backup/restore before activation.
7. No TXO operator/BFF/OpenFGA rollout gate or public browser route
   changes from this source foundation.

## Future staged activation gates (not approved here)

A new, dedicated approval increment must choose the actual versioned
OpenBao image/Helm schema, retained storage class, namespace, exact
ServiceAccount identities/issuer/audience and private network targets,
then create a separately named, non-production activation overlay.
It must remain unreferenced from
`apps/60-services/txo-fabric/overlays/prod` and
`argocd/overlays/prod` until a **second** explicit production
approval. Future ESO consumers must use only
`openbao-fabric-store`, with separate namespace-scoped access for
tenants and no all-tenant credential. The current source-only PR
provides **no service, no backup proof, no live injection and no
runtime acceptance**.
