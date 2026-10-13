# TXO Fabric — OpenBao Fabric POC bootstrap and restore acceptance


## Phase 1 POC — explicitly authorized source deployment (#3728)

The product owner has authorized **an empty, sealed, single-replica GitOps
deployment only**, superseding the older source-only/backup-before-empty-server
restriction. Dev Argo may create the objects after review/merge; prod-stable
promotion remains separately gated. The server uses `truenas-iscsi-retain`
5Gi PVC and no clients. Initial state is **Running/NotReady** (sealed and
uninitialized), so a non-ready Service is intentional; a failed container
or unbound claim is not. Read-only acceptance checks must inspect namespace
policy, retained PVC/PV, Service type and health status without values.
A human-approved local pod port-forward is the future bootstrap transport,
not permission to initialize or disclose root/unseal material here.

**Phase 2:** separately authorized init/unseal, state and internal health.
**Phase 3:** TLS, audit, exact Fabric-only ESO/workload roles, then credentials.
**Phase 4 (parallel #4049):** native Raft snapshot, independent encrypted
recovery and isolated successful restore **before real Fabric HMAC, Hermes
API credentials or tenant data are stored**. Velero was decommissioned
(#4050); historical BSL references below are not a backup plan.

All later gates in this runbook apply to **privileged operation**, not to
the already approved empty phase-1 GitOps source objects.

## Talos-only placement decision / Prompt 07

OpenBao Fabric is the private **global TXO Fabric component on Talos
Kubernetes**, to be managed by GitOps. It is **not** an experimental
K3s workload and not an externally hosted substitute. The #4046
read-only preflight was corrected from K3s to Talos v1.14.1 /
Kubernetes v1.36.0; this PR performs **no live verification**.
High memory requests and the reported Velero
`BackupStorageLocation/default=Unavailable` block stateful
activation until non-secret capacity-placement evidence and a
**working backup destination plus actual isolated restore drill**
are verified. A present Retain StorageClass or Ready snapshot is
insufficient. No default-to-Delete storage, no public endpoint, no
Internet catch-all or speculative backup route.

The operator cannot install OpenBao policies. Future tenant
`TenantBundle` reconciliation sends only idempotent immutable-ID
intent to an independently authenticated, audited and template-bound
provisioner; see
[tenant-provisioning-boundary.md](../../apps/60-services/txo-fabric/openbao-fabric/contracts/tenant-provisioning-boundary.md).
Group/user IAM is an OpenFGA/ADR-039 concern and cannot grant an
OpenBao path. Revocation denies *new* access before preserving
retained data/audit and before any separately authorized disposal.
A future physical acceptance must explicitly test replay,
sibling-ID rejection, arbitrarily supplied HCL, wrong workload
subject/audience, existing short-lived lease revocation and
secret-store unavailable/sealed behavior without fallback.

**Status: phase-1 POC GitOps source authorized by #3728, but no manual
bootstrap, credentials or operational acceptance authorized.** No Secret,
auth backend or ESO store is introduced by this increment. Run these steps **only** after a separate,
explicit human approval identifying the exact GitOps candidate,
environment, data retention policy and operator. Never run commands
from this document as an informal manual cluster fix.

**Separation:** existing `ClusterSecretStore/openbao` is **Infra**
(`apps/00-infra/openbao/base/cluster-secret-store.yaml`) and
keeps its exact URL/tokenSecretRef as-is. The proposed new authority
is `openbao-fabric`; its ESO integration is named
`openbao-fabric-store`. Infra retains custody of Fabric bootstrap,
break-glass, backup and restore. Fabric holds only runtime TXO
credentials and future tenant-scoped records. The split is an audit
and blast-radius boundary, **not** immunity to an Infra break-glass
administrator.

## Hard safety rules for ALL stages

- Never copy or emit an actual credential, bearer token, password,
  recovery/unseal material, root/admin authority, private key,
  `Secret.data`, encoded secret, SQL URI or source DB parameter into
  Git, issues, PRs, CI/test logs, terminal transcripts delivered to an
  agent or the handoff. Reports contain only **existence, readiness,
  success/failure, timestamps and non-secret object names**.
- Human approval must explicitly authorize any physical bootstrap,
  stateful storage creation, init/unseal, backup/export, auth policy
  installation, ESO activation, rotation/revocation or restore. A PR
  merge or valid `dev-v*` alone is insufficient.
- No direct persistent cluster mutation by the implementation agent;
  manifests must use normal GitOps PR/review and a **separately
  approved** release. Do not issue `kubectl apply/edit/delete`,
  Argo forced sync, PV/PVC deletion, or production promotion.
- OpenBao Fabric may not silently fall back to the Infra server.
  Fail closed if the new server, service identity, policy, certificate,
  secret or authoritative tenant binding is unavailable.
- No public Ingress/Gateway/HTTPRoute/NodePort/LoadBalancer or
  permissive network policy. No admin/break-glass token mounted in
  any workload ServiceAccount/Pod. No shared tenant-wide wildcard
  credentials.

## Gate 0 — approval + implementation design

**Owner:** Electra + infrastructure authority + charchess for the
specific activation candidate.

Before enabling **credential-bearing operation**, establish and review:
chosen supported OpenBao image/chart and immutable digest, intended
namespace and Kubernetes Pod Security, exact HA/storage topology
(e.g. verified Raft implementation if chosen), approved private TLS
identity/CA, cluster issuer and TokenReview audience, dedicated
ServiceAccounts, namespace/pod selectors, deterministic bootstrap/
migration ordering and exact network destinations. None is assumed by
#4046. Capture missing choices as blockers, **not wildcard rules**.

Revalidate that the selected storage class and its physical reclaim
policy are **Retain** for PVC/PV; refuse any `Delete` reclaim
fallback. Choose RPO/RTO, snapshot consistency procedure, backup
object-store identity/ownership/encryption and disaster-restore
authority. Proof can contain class names/policy and existence status,
not material or private endpoints that identify secrets.

## Gate 1 — stateful provision and backup plan

The #3728 POC has a real private StatefulSet/PVC/ClusterIP/deny-all source
selected by the TXO dev/prod overlays; production still requires human
promotion. It has no credential clients or ESO bindings. The owner explicitly
selected the official version-tagged image 2.6.2 for this POC.

Before **real credential/data use**, prove in a separate physical acceptance
window: actual bound retained storage; restore-compatible version and
schema; protected snapshot destination and independent read-only
backup verification. A snapshot job configured but never restored is
**not** proof of recoverability.

## Gate 2 — initialization, unseal and break-glass custody

A separately approved trusted human infra custodian executes
initialization/unseal **outside Git, CI and agent chat**, using the
authorized interactive/secure custody system. Recovery shares and
unseal/root material remain in that system; enforce distinct custody,
multi-party review, short-lived break-glass access and audited use.
Do not put recovery data in ConfigMaps, git-crypt fixtures,
`ExternalSecret` properties, Terraform outputs/logs or operator
environment variables.

After seal/unseal, record only the non-secret initialized/sealed
readiness flags and relevant audited event identifiers. Disable/revoke
bootstrap authority from normal workload identities. A restart should
not create a fresh empty state or expose bootstrap material.

## Gate 3 — Kubernetes workload authentication and policies

Verify the **real** Kubernetes issuer/audience and exact
ServiceAccount/namespace subject for each consumer; TokenReview must
fail for a different audience or namespace. Install read-only,
versioned, narrowly named policy roles according to
`openbao-fabric/contracts/workload-authz-matrix.md`. The
`*.hcl.example` snippets are **not** installable policies; any
tenant ID marker must be replaced by a separately validated
immutable `fabricTenantId`, never a path wildcard/tenant slug.

Examples of intended **logical** read scopes:
`fabric/platform/txo-fabric/webui-bff`,
`fabric/platform/txo-fabric/authentik-reader`,
`fabric/platform/txo-fabric/openfga-reader`,
`fabric/tenants/<one-immutable-tenant-id>/*`.
A Fabric workload cannot issue auth tokens, manage policies or
mounts, read audit logs or initialize/unseal the service.

Prove a positive read of its own **test fixture only**, and refusal of
wrong identity, wrong audience, sibling tenant, global tenant
metadata list and any policy/token/admin API. Do not emit fixture
contents. Auth role/provisioning credentials themselves are held
outside workload Kubernetes auth.

## Gate 4 — Fabric-only ESO wiring

A separate GitOps PR introduces a new
`openbao-fabric-store` (prefer namespace-scoped `SecretStore`
for tenant namespaces) using short-lived, workload-bound Kubernetes
auth, an approved private URL/CA, audited authentication and exact
path policy. Neither copy nor reuse the existing
`ClusterSecretStore/openbao` token. Ensure ESO's control-plane
identity cannot read sibling tenant data even if it reconciles
several namespaces. Do **not** grant a blanket `fabric/tenants/*`
policy.

Only after the Fabric store is actually Ready and negative controls
pass may the appropriate newly reviewed `ExternalSecret` be wired
to that store. Existing OpenFGA references to Infra and the staged
WebUI bootstrap Infra references are migration debt; changing them
requires a **separate**, reversible cutover with no unknown outage or
silent fallback.

## Gate 5 — existence-only injection and fail-closed negatives

Verify via metadata/status **only**, never printing keys or values:

1. The new Fabric store authenticates only its named workload
   principals and the exact intended secret path exists.
2. A future allowed test `ExternalSecret` reports Ready and its
   target exists with expected type/key **names** only; the workload
   starts after values are available but no value is included in
   any logs. Validate rotation/revocation on a distinct test object.
3. **Negative A: wrong workload identity** (wrong SA, namespace,
   TokenReview audience or issuer) cannot authenticate/read.
4. **Negative B: sibling tenant path** cannot read or list another
   tenant's secret; tenant `C` never inherits `A` grants.
5. **Negative C: OpenBao Fabric unavailable/sealed** causes
   injection/consumer authorization to fail closed, with no
   fallback to Infra `openbao` and no stale sensitive operations.

A `Ready=True` CR alone is not proof if the actual workload
identity or object never existed. Test re-availability explicitly.

## Gate 6 — backup, restore and revocation acceptance

Create an encrypted verified snapshot with integrity/readability
evidence and protected retention. In a deliberately isolated
nonproduction recovery environment, execute a **restore drill**
using authorized human custodians; verify KV content only by
expected record/key **existence and cryptographic comparison held
within the approved secure environment**, not printed values.
Reapply/check policies, issuer/audiences, audit stream and revoked
identities after restore. A restored stale token or sibling path
must not become valid. Record nonsecret timestamps, restore duration,
object counts and negative-test PASS/FAIL.

Rehearse a credential/key rotation with simultaneous ESO refresh,
actual workload restart/reload semantics and a strictly denied
old credential. Distinguish recoverable infrastructure recovery
authority from runtime workload read authority.

## Gate 7 — controlled rollback / acceptance

If any stage fails, **deny** new Fabric secret reads and roll back
**only the separately approved activation overlay/gates through
GitOps**. Keep retained OpenBao data, backups, PVC/PV and historical
tenant identity ledgers. Never erase/replace persistent state or
redirect a denied Fabric caller to OpenBao Infra. Capture causes,
nonsecret status and a follow-up issue; re-approval is required.

The #4046 **source-only** increment can be reviewed/merged
independently but **must remain OPEN** until its own physical
init/unseal, backup/restore, auth policy, ESO and negative tests
are separately executed and accepted. Do not claim a tenant can
read a secret or a browser login is operational merely because
this document or an ADR exists.

## Current prerequisites still unresolved

- Supported pinned OpenBao Fabric Helm/image and private TLS transport.
- Stateful CSI StorageClass with verified **Retain** semantics,
  backup destination and restore drill.
- Actual issuer/audience, exact workload identities and auth backend.
- Exact ingress/egress allowlist endpoints, ESO controller/tenant
  scoped role binding and production activation approval.
- Physical negative tests, failure recovery and reversible migration
  of existing TXO Infra-secret dependencies.

**Record each as issue-linked evidence in a future authorized
increment, without any secret value or privileged transcript.**
