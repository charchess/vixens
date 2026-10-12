# ADR-041: Separate OpenBao Fabric runtime authority from OpenBao Infra

**Date:** 2026-10-12
**Status:** Accepted — architectural boundary; **implementation and physical acceptance pending**
**Scope:** AIaaS
**Related Project:** vixens roadmap
**Related Issues:** #4046, #4043, #3807, #3996, #4037, #3981
**Deciders:** TXO Fabric product owner
**Tags:** secrets, OpenBao, ESO, tenant isolation, disaster recovery

## Context

The historical [ADR-018](018-openbao-external-secrets-and-nas-fqdn.md)
established an External Secrets Operator integration against an OpenBao
instance hosted outside the TXO Fabric runtime. The existing
`ClusterSecretStore/openbao` belongs to this **infrastructure**
authority. It uses the existing URL and token-secret binding in
`apps/00-infra/openbao/base/cluster-secret-store.yaml` and must remain
unchanged by a source-only Fabric increment.

TXO Fabric now needs a separate secret authority for credentials consumed
by its platform workloads and, eventually, isolated customer workloads.
Sharing one infrastructure administrator token/store/mount across
application runtime credentials and infrastructure recovery unnecessarily
couples normal runtime operations to infrastructure break-glass,
increases credential blast radius and confuses audit ownership.

## Decision

**OpenBao Infra:** credentials needed to **provision, administer, repair,
back up or restore infrastructure**, including the bootstrap/unseal,
backup and recovery authority for OpenBao Fabric.

**OpenBao Fabric:** credentials **consumed at runtime by TXO Fabric
components**, including a future strictly tenant-scoped secret space.
Platform workloads and tenant workloads have distinct, explicitly
authenticated identities and non-overlapping least-privilege policies.

The two authorities are different operational trust domains with different
audit streams, secrets, authentication roles, blast radii and rotation
procedures. This separation is **not an absolute privilege boundary
against an OpenBao Infra administrator**: trusted infra break-glass
operators may be able to bootstrap, recover or regain administration of
OpenBao Fabric. Their actions must be separately authorized, recorded
and audited; a Fabric workload must never acquire that authority.

### Classification and ownership

| Class | Examples | Authority |
| --- | --- | --- |
| Infrastructure | Talos/cluster break-glass, Terraform and provisioning, CSI/TrueNAS, backups and S3, DNS/ACME, Argo and GitHub deployment credentials, OpenBao Fabric bootstrap/unseal/backup and restore | **Infra** |
| Fabric platform runtime | BFF→bridge HMAC, CNPG migrator/runtime principals, Authentik technical account reader, OpenFGA technical credentials, internal connector credentials | **Fabric platform** |
| Fabric tenant runtime | `fabric/tenants/<tenant-id>/*` — only workloads authorized for that **one immutable Fabric tenant identifier** | **Fabric tenant** |

Tenant IDs are operator-issued immutable identifiers, never a browser
tenant slug, OIDC `sub`, customer input or mutable group name.
No credentials belonging to a tenant move into OpenBao Infra for
convenience. No tenant KV prefix is visible through another tenant's
role or through a shared catch-all runtime ESO identity. The bare
`fabric/tenants/*` is **not** an allowed tenant workload policy.

### Security and access boundaries

- **Named objects:** use `openbao-fabric` for the future instance and
  `openbao-fabric-store` for a future *distinct* ESO store. Never
  overload `ClusterSecretStore/openbao` or reuse its token.
- **Authentication:** bind exact Kubernetes issuer/audience (determined
  and verified during approved bootstrap), token ServiceAccount name,
  namespace and bound short token lifetime to a **specific** Fabric
  OpenBao Kubernetes auth role. No legacy default SA tokens or
  permissive administrative roles. The Fabric plane does not directly
  inherit infrastructure administrator rights.
- **Fabric platform:** the BFF may read only
  `fabric/platform/txo-fabric/webui-bff`. The designated operator
  Authentik/OpenFGA reader may read only its two named platform records;
  use narrower individual roles if their consumers are distinct.
  Connector credentials require their own separately approved bindings.
- **Fabric tenant:** a role is materialized from an authoritative,
  immutable `tenant-id` **one tenant at a time**. Tenant workload
  subjects cannot list/read siblings, enumerate all tenant paths or
  administer policy. Absent/unverified tenant binding => deny.
- **Never grant application workloads:** policy/mount management, token
  issuance, admin/self-escalation, audit-log read, seal/unseal/init or
  root/superuser authority. ESO has only the read role(s) needed for
  specific secret delivery; a shared all-tenant ESO token is forbidden.
- **KV v2 semantics:** `fabric/data/...` is the data API path;
  `fabric/metadata/...` is **not** granted for cross-tenant list.
  Paths in documentation describe *logical* secrets and never imply a
  running KV mount or working login.

### Storage, availability and recovery

OpenBao Fabric's state is durable and **Retain** under the actual
approved storage class and PV reclaim policy; no `Delete` fallback and
no automatic state destruction on uninstall. Before any stateful
deployment, validate the actual StorageClass/CSI reclaim semantics,
backup target ownership/isolation/encryption, snapshot consistency,
off-site copy and a restore drill. Infra is the authorized custodian of
bootstrap/recovery material; Fabric application credentials never need
to be printed into an issue, CI log or handoff.

The future runtime is private by construction: ingress default-deny,
no Ingress/Gateway/HTTPRoute/NodePort/LoadBalancer, no broad Internet
egress. Bootstrap and runtime network flows must be independently
specified with exact namespaces/workloads and endpoints before
enablement; an unresolved endpoint is a **blocker**, not an invitation
to permit `0.0.0.0/0`.

### Lifecycle gates

1. **Provisioning/bootstrapping:** human-approved isolation, retained
   state, private service, backups and separate recovery authority.
2. **Init/unseal:** only out of Git, CI and chat/logging; no root token,
   recovery key or unseal key material in this repository or issue.
3. **Normal runtime:** workload-bound Kubernetes auth and explicit
   read-only Fabric platform/tenant policies, then a *new* Fabric-specific
   ESO store. Existing infrastructure ESO consumers remain untouched
   until migration has a physically verified equivalent.
4. **Rotation/revocation:** rotate secret values and auth role bindings
   independently, prove old identities/paths deny, verify ESO refresh
   and consumer restart/reload semantics before declaring success.
5. **Backup/restore:** recover retained state from verified encrypted
   snapshots, validate grants, auditing and revocation after restore;
   never assume the mere presence of a snapshot is a successful drill.
6. **Break-glass:** separate human authorization and auditable incident
   procedure; revoke temporary administrative capability afterward.

### Existing legacy dependencies and no automatic migration

At this source-only baseline, TXO OpenFGA
`openfga/runtime/external-secrets.yaml` actively references the old
`ClusterSecretStore/openbao`, and the *unreferenced* WebUI bootstrap
`webui/bootstrap/credentials.yaml` is also pointed there. These are
**known migration debts**. They do not redefine the desired authority.
A separate review must perform availability-preserving re-enrollment
into `openbao-fabric-store` with no privilege widening, only after
physical Fabric store readiness. **This ADR does not change, copy or
migrate any existing credential, store, backend, secret or gate.**

## Consequences

**Benefits:** isolate workload authority from infrastructure recovery,
clearer platform vs tenant credential ownership, safer tenant lifecycle
and independently inspectable audit/rotation.

**Costs:** a second stateful critical security dependency, bootstrap and
disaster-recovery burden, extra role/store governance, staged migrations
and explicit fail-closed behavior during outages.

**Status/limits:** the decision is accepted; **no Fabric OpenBao runtime,
secrets, ESO binding, Kubernetes auth, tenant policy or physical recovery
has been deployed or validated** in #4046. See the
[source-only contract](../../apps/60-services/txo-fabric/openbao-fabric/README.md)
and [operator runbook](../troubleshooting/txo-fabric-openbao-bootstrap-and-restore.md).

## Alternatives considered

- **Continue reusing Infra OpenBao for Fabric runtime:** rejected due
  shared operational/admin boundary, broad failure modes and audit drift.
- **One OpenBao instance per tenant immediately:** deferred; avoid
  multiplying operational stateful instances before the fixed per-tenant
  identity/policy lifecycle is proven. One Fabric instance with verified
  per-tenant auth separation is the initial contract.
- **Migrate every existing TXO ESO reference immediately:** rejected.
  Would risk outage/credential disclosure before new Fabric plane is
  available and physically accepted.

## References

- Issue [#4046](https://github.com/charchess/vixens/issues/4046)
- [ADR-018](018-openbao-external-secrets-and-nas-fqdn.md)
- [ADR-030](030-project-scoped-adr-governance.md)
- [ADR-039](039-fabric-openfga-authorization-boundary.md)
- [ADR-040](040-global-fabric-webui-entrypoint.md)

## Addendum (2026-10-12): Talos placement and controlled reconciliation intent

The approved **hosting target** for `openbao-fabric` is now the
**global TXO Fabric control plane on the Talos Kubernetes cluster**,
managed through GitOps. It is not a K3s development service and not an
independent externally hosted replacement. This **clarifies placement**;
it does not activate a server, create an Argo application, authorize
init/unseal, or supersede the separation of Infra recovery authority
from Fabric runtime credential authority.

The read-only preflight recorded in issue #4046 first mislabelled the
observed control plane as K3s, then corrected it to **Talos v1.14.1 /
Kubernetes v1.36.0**. Those values are *reported observations*, not
live validation performed by this source-only PR. The preflight also
reported high node memory-request saturation and an **Unavailable**
Velero backup location. These are explicit blockers to verify/resolve:
placement and capacity reservations **on Talos**, a physically
available backup destination, and an isolated successful restore drill.
An observed `Retain` CSI class/snapshot API alone does not prove data
recoverability. No stateful GitOps target becomes activation-eligible
until these gates and the prior storage/auth/TLS/network gates pass.
No public route and no broad egress are permitted.

**Day-to-day privilege separation remains mandatory:** routine OpenBao
Infra and its ESO token have **no authority to read Fabric runtime
records**. Infra holds Fabric bootstrap/unseal/recovery/backup
authority only through an independently approved, audited
break-glass ceremony; this is not a normal workload read identity.
Each Fabric runtime consumer reads exactly its named platform leaf
or its one tenant subtree, never an all-tenant token.

The TXO Fabric operator may translate current **canonical** cluster
TenantBundle state into an **idempotent provisioning intent**, not
a direct authorization to administer OpenBao. In the implemented
TenantBundle API `spec.tenantId` is immutable
(`TEN[0-9]{5,}`), and `metadata.uid` identifies the exact
object incarnation. The separate approved provisioning boundary
must re-read and validate both, the reconciliation generation, the
workload ownership and an exact ServiceAccount namespace/audience
binding. It generates only immutable, reviewed policy templates
from a closed catalogue; the caller cannot send ACL/HCL, free-form
paths, policy names, administrator roles or tenant-wildcard grants.
Group and user changes remain application-authorization state under
ADR-039/OpenFGA, never secret-path authority.

A policy-name prefix is **not** a safe delegation boundary:
permission to upload arbitrary policy bodies would let the operator
install a named policy containing tenant-sibling or administrator ACLs.
Even if the OpenBao management API later permits naming restrictions,
this ADR does not assume it verifies ACL *content*. The normal
operator must never hold direct `sys/policies` write, root,
unseal/recovery material, arbitrary mount/token/audit authority or
all-tenant read. The future installer operates with a separately
bootstrapped, narrowly scoped, audited identity and **must** fail
closed if its actual constraints cannot be proven. A reviewed
offline/manual policy-install path is preferable to unsafe automatic
delegation.

Tenant lifecycle may reserve immutable tenant metadata/paths and
approved workload bindings; deletion or revocation **denies access
first**, retains audit and persistent data, and requires a distinct
human-approved destruction procedure. A user, group, namespace label,
browser parameter or OIDC subject cannot trigger role creation.
The complete non-executable transition/request/negative-test contract
is [tenant-provisioning-boundary.md](../../apps/60-services/txo-fabric/openbao-fabric/contracts/tenant-provisioning-boundary.md).
