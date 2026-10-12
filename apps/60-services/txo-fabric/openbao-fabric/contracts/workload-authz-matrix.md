# OpenBao Fabric — workload authorization contract (NOT INSTALLED)

## Logical secret names, not granted token mounts

OpenBao Fabric owns **only runtime TXO credentials**. These names are
**logical KV v2 paths** in the future dedicated Fabric mount
`fabric/`. An API read under KV v2 uses the corresponding
`fabric/data/...` path. No KV mount exists or is enabled by #4046.

| Workload identity (future exact subject) | Allowed logical read scope | Explicit denial |
|---|---|---|
| Global `txo-fabric-webui-bff` ServiceAccount | **Only** `fabric/platform/txo-fabric/webui-bff` | No operator-reader, tenant, migration, admin or sibling platform records |
| Operator's Authentik technical reader | **Only** `fabric/platform/txo-fabric/authentik-reader` | No OpenFGA private credential unless a separately reviewed dual-reader role is necessary |
| Operator's OpenFGA technical reader | **Only** `fabric/platform/txo-fabric/openfga-reader` | No Authentik credential or tenant values without explicitly justified co-location |
| BFF CNPG migrator/runtime identities | Separate exact platform credential record(s), future named leaf nodes; no wildcard platform grant | Neither principal gets shared superuser, Infra recovery or cross-tenant read |
| Dedicated connector caller | One exactly named platform connector record after review | No generic `connectors/*` grant |
| One tenant workload, bound to **immutable tenant ID** | `fabric/tenants/<its-tenant-id>/*` within that one tenant only | No `fabric/tenants/*`, no sibling tenant key/path or global tenant list |
| OpenBao Infra break-glass operator | Bootstrap/recovery **outside** workload Kubernetes auth | Never becomes a Fabric runtime ServiceAccount or ESO token |

**Always denied to any application workload identity:** policy create,
update/delete; token creation, arbitrary renewal or token lookup;
mount/audit-system administration; identity auth role management;
root/admin escalation; seal/unseal/init, backup/restore and reading
audit logs. A user, tenant admin, platform admin or Fabric tenant
cannot self-approve changes to these security boundaries. Fabric
normal runtime identities are **read-only** for secret data. Granting
migration DDL SQL is a separate SQL database policy concern, not a
permission to mutate OpenBao policies or mounts.

### Kubernetes auth mapping — pending physical configuration

Every future role binding needs all of the following, reviewed as a
single immutable tuple:

- actual validated Kubernetes TokenReview issuer (not guessed);
- dedicated verified TokenReview audience (not guessed);
- exact ServiceAccount namespace and name (no `*` subject);
- bounded token duration, binding and service-account lifecycle;
- separate policy name/version and exactly one intended logical
  leaf/subtree; limited TTL and renewal/revocation;
- an audit identifier that cannot be forged by a tenant or browser;
- positive read of **only** the exact intended path and negative
  read/list of sibling platform/tenant paths after bootstrap.

Missing any component => **no role and no secret read**, not an
`allow all` fallback. Never infer an immutable Fabric tenant ID from a
namespace label, user group, slug, OIDC subject or HTTP header.
`TenantBundle` / operator truth is the source. A new tenant role is
created only by separately approved, tested reconciliation that
validates the canonical tenant ID and service-account binding.

### ESO boundary

The planned store name is `openbao-fabric-store`, and is NOT the
existing Infra `ClusterSecretStore/openbao`. Platform and tenants
must receive separate Kubernetes auth identities. Prefer
namespace-scoped SecretStores for tenants, with no shared ESO role
that can read more than one tenant subtree. If ESO's controller must
use a centralized role, its authority is limited by exact
ServiceAccount and per-namespace binding with provable sibling
denial; otherwise do not activate it.

No `ExternalSecret` is added here; none may point to OpenBao Infra
for **new** Fabric consumers. Existing OpenFGA Infra references are
documented migration debt and unchanged during this source-only
increment.

### Reference policies

`../policies/*.hcl.example` are **read-only examples**, not
runnable roles, policy IDs, secret IDs or templated tenant credentials.
Examples use scoped KV v2 `data` paths only and deliberately grant no
KV `metadata` list, sibling, namespace/global wildcard or system
paths. Tenant templates contain an unresolved marker that must never
be used verbatim. A future policy installer must reject
unsubstituted or malformed placeholders and prove sibling denial in
a physical acceptance test.

Rotation and revocation belong to a future controlled workflow:
short leases, audited expiry, invalidated previous credentials and
explicit observed ESO refresh. During OpenBao Fabric outage, no
fallback to Infra `openbao` is allowed for a new Fabric consumer.
