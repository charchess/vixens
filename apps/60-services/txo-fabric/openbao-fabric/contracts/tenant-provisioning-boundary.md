# Controlled tenant secret provisioning — design-only contract (#4046 / Prompt 07)

**Status: NON-EXECUTABLE.** This describes a future interface and proofs,
not a CRD, webhook, OpenBao policy installer, controller, Kubernetes Job,
request broker, migration or enabled GitOps overlay. No component currently
accepts or executes this request.

## Canonical sources, never labels or browser identity

The existing cluster-scoped `TenantBundle` operator contract has:
`spec.tenantId` — **immutable FabricTenantID**, validated against
`^TEN[0-9]{5,}$`; `metadata.name` — technical Kubernetes tenant slug;
`metadata.uid` — immutable object incarnation; and
`metadata.generation` — reconciliation revision. The final two values
must be read from Kubernetes by the trusted controller, **not supplied by
a tenant/group/user or browser**. Existing `HumanAccess.Web.IAMGroups`
are group *keys*, not grants to Fabric secrets. Tenant ownership binds
both the immutable tenant ID **and** TenantBundle UID: never recycle
an old UID's role for a recreated bundle bearing the same ID.

Only verified cluster-observed canonical TenantBundle state may
request new secret **metadata/path reservations**, not content. User
and group updates belong to ADR-039/AuthentiK/OpenFGA authorization;
they do **not** create OpenBao read roles. An AgentIdentity or other
workload is only eligible when the operator has independently verified
its own owner/tenant binding and a dedicated platform-owned
ServiceAccount. The browser may neither initiate this API nor choose
an identity, path, policy or approver.

## Sequence (later, separate approved implementation)

```text
GitOps TenantBundle (cluster-scoped, UID + immutable spec.tenantId)
    |
    v   validate actual object UID/generation and non-deleting lifecycle
TXO Fabric operator reconciler (intent ONLY; no OpenBao privilege)
    |   authenticated, bounded canonical Ensure/Revoke request
    v
Fabric secret-provisioning boundary (SEPARATE workload/authority)
    |   re-read TenantBundle and exact workload owner from API
    |   compare UID, generation, tenant ID, approved identity binding
    |   admit ONLY fixed, reviewed template catalogue and derived paths
    |   reject arbitrary HCL, paths, policy names and all-tenant roles
    v
Future Fabric OpenBao scoped management adapter
    |   separate narrow, audited bootstrap-installed management identity
    |   fixed templates and readback for exactly ONE tenant, no root
    |   no plaintext credentials or secret-value creation here
    v
Reviewable tenant-bound Kubernetes auth + namespace-local ESO store
    |   exact (namespace, ServiceAccount, audience, issuer), verified
    |   policy read on ONLY fabric/tenants/<canonical-id>/approved-leaf
    v
Tenant workload: read its OWN logical Fabric subtree only
    |   wrong identity / sibling tenant / sealed or unavailable => DENY
```

The boundary may not be co-located inside the operator or share its
ServiceAccount. Its **separately scoped management identity** is not a
runtime consumer identity; it must be bootstrapped by an approved
privileged operator outside the workload auth flow and narrowed to the
specific pre-reviewed policy/auth operations. Whether OpenBao can safely
scope those management operations to fixed policy bodies is **unproven**:
authorization based on a policy *name prefix* does NOT constrain its
content. A policy writer allowed to submit arbitrary HCL can embed
`sys/*`, `auth/*` or `fabric/data/tenants/*` in that content and
escalate to another tenant or administration. Therefore **no direct
operator `sys/policies` rights**, even under a name prefix. A future
implementation must constrain requested inputs in independent code,
generate exact template content server-side, restrict the separate
installer identity and prove denied arbitrary writes and drift by
readback. If privilege scope cannot be demonstrated, **do not ship a
runtime provisioner**; use a reviewed offline GitOps/human approval
workflow instead. Never assert OpenBao enforces policy-content
restrictions natively without tested evidence.

## Proposed request schema (review contract, not an API)

The authenticated operator may submit **only** fields below. The
receiver derives/checks all authority against current canonical
cluster state and its immutable binding registry. Unknown fields
are a hard denial. JSON keys illustrate a shape, not a runnable endpoint.

```json
{
  "contractVersion": "v1",
  "action": "Ensure",
  "source": {
    "kind": "TenantBundle",
    "apiVersion": "fabric.truxonline.io/v1alpha1",
    "name": "canonical-tenant-slug",
    "uid": "canonical-kubernetes-object-uid",
    "generation": 1
  },
  "fabricTenantId": "TEN00001",
  "workload": {
    "namespace": "reviewed-namespace",
    "serviceAccount": "reviewed-service-account",
    "audience": "reviewed-kubernetes-audience"
  },
  "pathTemplate": "tenant-runtime-v1",
  "idempotencyKey": "deterministic-key-bound-to-source-and-action"
}
```

**Not trusted from this JSON:** source identity, namespace, SA,
audience, template eligibility, tenant lifecycle, phase or path
ownership. The actual authenticated caller must be the approved
operator workload; authenticate using a separately reviewed service
channel and Kubernetes issuer/audience. Require current, non-deleting
TenantBundle; a UID/generation/tenantId mismatch, stale condition or
missing approved workload-binding entry denies before writes.

The **boundary**, not the operator/client, expands one reviewed
`pathTemplate` into **approved exact logical leaves** of
`fabric/tenants/<canonical-FabricTenantID>/...`. Any supplied
`paths`, `policy`, `hcl`, `role`, `tenantSlug`, `group`,
`user`, `oidcSub`, `browser`, or policy-name field is rejected,
including hidden nested members. Never substitute a wildcard tenant
prefix or turn a free-form relative leaf into a trusted one.
The future `tenant-runtime-v1` version must be resolved in an
independently reviewed **closed template registry**, with template
digest, exact approved leaves and an immutable name-to-content
mapping. It is **not installed/defined by this document**.

An idempotency key must be **computed and verified at the receiver**
from source `uid`, `tenantId`, `generation`, `action`,
validated workload-binding fingerprint and fixed template version.
A client-supplied key is at most an equality check, not authority.
Duplicates at the same generation converge without new credentials,
roles, grants, token minting or repeated rotation; a stale generation
is Denied; a legitimate new generation requires full reread and
revalidation. Use atomic compare-and-swap/retained operation journal
to prevent races and avoid replay across operator replicas. Never
record secret material in audit/status/journal.

## Provisioning and revocation state table

| Trigger / authority | Action and resulting phase | Safety/evidence invariant |
| --- | --- | --- |
| Canonical active TenantBundle created | `Pending` → `Validated` → `Provisioned` only after exact UID/generation, template, binding and readback | No account or policy materialized merely from a name/label |
| Same tenant/source/generation re-reconciled | `Provisioned` unchanged; idempotent journal read | No duplicate binding, grants, role/token issuance or secret value rotation |
| Tenant spec revision with still-valid identity | New request `Pending`; verify current generation, binding and limited template before convergence | Old request and stale generation rejected |
| Group membership or user creation/reconciliation | **No secret-provisioning request**; ADR-039/OpenFGA authorization projection only | Group/user never controls secret paths or workload SA |
| New workload identity for an existing tenant | Separate verified workload-binding request, no user-derived authority; normal `Pending` to `Provisioned` only if approved | One immutable tenant owner, namespace and SA and verified audience/issuer |
| Unknown tenant, deleted/recreated UID, stale source, foreign identity, arbitrary path/HCL | `Denied` and audit reason (no secret value) | No partial grants or runtime fallback |
| Fabric OpenBao unavailable/sealed or cannot verify readback | `Error`; any new admission/access must fail closed, no Infra fallback | Never treat ESO cache/previous Ready as proof of a current grant |
| Tenant deletion, suspension or ownership revocation | `Revoke`: deny **new access first**, unbind/revoke auth/ESO grants before `Revoked`; retries idempotent | Preserve retained secret data, metadata, audit and tombstones |
| Data destruction request | **Out of band, separately approved**; not a reconciliation side effect | Snapshot/retention/audit legal hold verified before any deletion |

Where an outage prevents the revocation write, status remains
`Error`, the platform deny gate must stop sensitive operations, and
an operator must retry until revoked. A historical cached value or a
preexisting minted lease may survive temporarily; **reconciler status
alone cannot revoke an already-issued capability**. Future physical
acceptance must test lease TTLs, revoked credentials, ESO refresh,
workload restarts and absence of cross-tenant access.

The future status is an *observation*, not a grant:
`{ phase: Pending|Validated|Provisioned|Denied|Revoked|Error,
observedGeneration, sourceUID, bindingFingerprint, templateVersion,
lastTransitionTime, reason }`.
Reasons are from a closed non-secret catalogue:
`UnknownTenant`, `SourceMismatch`, `StaleGeneration`,
`InvalidWorkloadBinding`, `UnapprovedTemplate`,
`ForbiddenPolicyContent`, `SiblingTenantPath`,
`NoncanonicalGroupUserAuthority`, `FabricUnavailable`,
`FabricSealed`, `RevocationPending`, `Verified`.
No raw UUID of a human, secret bytes, bearer, access token or
untrusted exception text may be placed in status. Status is not
machine-readable operational truth until separately implemented/tested.

## Isolation and acceptance matrix

- **Tenant A → A:** after explicit activation, verified exact SA,
  issuer, audience and A-only template allow read of A's approved
  records; not a positive test before the service exists.
- **Tenant A → B:** denied read **and** list, even if group/user has
  a Fabric Admin application relation in OpenFGA.
- **Unrecognized / recreated TenantBundle:** denied, including reuse
  of a prior technical slug or same tenantID under a different UID.
- **Wrong namespace, SA, issuer or audience:** denied on auth even if
  the submitted request claims the correct FabricTenantID.
- **Arbitrary HCL/policy content, wildcard tenant path, admin ACL:**
  denied by independent boundary, not allowed by naming convention.
- **Group/user-derived tenant authority:** denied without ever invoking
  the policy installer. OpenFGA may allow application actions without
  enlarging the secret store ACL.
- **Fabric unavailable/sealed, missing backup or placement gate:** no
  activation/provisioning; no fallback to the Infra store.
- **Delete/revoke:** deny first; retained data and audit remain; only
  a separately approved, destructive lifecycle can erase records.

**Required separate approvals:** Talos resource/capacity placement,
real `Retain` CSI/PV, working external backup destination and
successful isolated restore drill, audited key custody, versioned
chart/CA, exact TokenReview identity binding, fixed installer template
registry and adversarial physical tests. This contract adds no
Kubernetes resource or enabled code.
