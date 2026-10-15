# OpenBao Fabric — private GitOps substrate and internal TLS (#3728)

**Current source-only increment:** internal TLS via namespace-scoped cert-manager
issuance, with **no runtime client, ESO store, new Secret manifest or Fabric
credential activation**. The source changes the existing server listener from
HTTP to HTTPS **on the same private ClusterIP:8200**. Nothing is deployed or
promoted by this PR. The physical instance is already initialized and unsealed
according to issue #3728's separately recorded acceptance; that is **not** a
live observation performed by this change.

## Infra Transit seal preflight (unselected, blocker active)

OpenBao Fabric continues on **Shamir**. A separate
[`transit-staged`](transit-staged/README.md) Kustomization contains
only the future dedicated, non-human bootstrap ServiceAccount. No
normal dev/prod overlay references it; no Pod, token, role, policy,
NetworkPolicy egress or seal configuration changes here.
**Source readiness remains blocked**: Infra's currently declared
endpoint is HTTP, not a verified TLS endpoint with known CA/SNI and
private IP. The existing Fabric server must not be restarted into
a speculative Transit seal. The distinct future Shamir→Transit
ceremony needs explicit human approval and a verified Raft restore.

## Namespace-local trust chain and rollout boundaries

- Four cert-manager objects live **only in `txo-fabric-system`**:
  `Issuer/openbao-fabric-selfsigned` bootstraps
  `Certificate/openbao-fabric-ca` (a self-signed CA stored by cert-manager
  as a namespace Secret); `Issuer/openbao-fabric-ca-issuer` signs
  `Certificate/openbao-fabric-tls`. Only cert-manager creates the
  corresponding CA and leaf TLS Secrets. **Neither certificate nor key
  material is committed in Git**. No ACME/public DNS issuer is used.
- The leaf `secretName: openbao-fabric-tls` has precisely four Service DNS
  SANs: `openbao-fabric`, `openbao-fabric.txo-fabric-system`,
  `openbao-fabric.txo-fabric-system.svc`, and
  `openbao-fabric.txo-fabric-system.svc.cluster.local`. The configured API
  identity is `https://openbao-fabric.txo-fabric-system.svc:8200`.
  Future consumers must receive and validate the **internal CA trust
  bundle in their own separately approved workstream**; no bypass of TLS
  verification is authorized here.
- Pod uses `openbao-fabric-tls` exclusively as a **read-only mounted
  Secret volume** at `/openbao/tls`, with rootless UID 100/GID 1000/fsGroup
  1000 and restrictive volume modes. The HCL references the mounted
  `tls.crt`/`tls.key` paths; neither appears as inline content.
  HTTPS readiness remains `/v1/sys/health` on port 8200; no liveness
  restart loop or alternate public listener is introduced.
- **Precondition for future approved GitOps rollout:** cert-manager and its
  CRDs/controllers operational, CA Issuer Ready, leaf Certificate Ready,
  generated `openbao-fabric-tls` present and readable by UID 100/GID 1000.
  Issuance is asynchronous. If the TLS Secret does not exist, the
  StatefulSet **fails closed** at mount/start rather than exposing HTTP.
  Promotion of this source would roll an already unsealed Raft server:
  the operator must plan a reversible single-pod maintenance window and
  physical HTTPS/permissions/health checks. No rollout is requested here.
  CA renewal/rotation, trust distribution and disaster restoration of
  cert-manager-generated CA custody require an explicitly reviewed
  operational procedure before clients are enabled.

## Retained private substrate and non-goals

- One `quay.io/openbao/openbao:2.6.2` StatefulSet and single-node Raft on
  `data-openbao-fabric-0`, 5Gi RWO `truenas-iscsi-retain`,
  claim retention policy Retain/Retain. **Never delete/prune PVC/PV.**
  No init/unseal automation, HA or forced scheduling.
- Private `Service/openbao-fabric` ClusterIP TCP/8200, exact-selector
  NetworkPolicy with `ingress: []` and `egress: []`, plus namespace
  default-deny. This PR grants **zero network clients** including cert-manager
  (which uses the Kubernetes API to issue Secrets, not a connection to
  the OpenBao listener). No Hermes/BFF/browser/bridge connectivity.
- `runAsNonRoot=true`, numeric UID 100, GID 1000, fsGroup 1000,
  seccomp RuntimeDefault, drop ALL, no privilege escalation, read-only
  rootfs and writable `emptyDir` scratch at `/tmp`. No workload token
  automount, new ServiceAccount or additional privilege.
- Existing Infra `ClusterSecretStore/openbao` and four historical TXO
  ExternalSecrets remain **unaltered**. Neither this TLS Secret nor the CA
  is an OpenBao runtime credential or an ESO Fabric store. No Infra fallback.
  `TXO_FABRIC_PRIVATE_BRIDGE_ENABLED=false` and
  `TXO_WEBUI_BFF_BROWSER_ENABLED=false` remain in force.

## Audit acceptance contract — documented, not installed

Before a future credential-bearing activation, a **separately approved
custody ceremony outside Git/CI** must activate a secret-safe OpenBao audit
device with `log_raw=false` (default; enforce explicitly) so sensitive
request/response fields are HMAC-obfuscated. An audit PVC **separate from
Raft data**, backed by `Retain`, must be installed and capacity-tested
before the audit gate can be accepted. Audit access is forbidden to all
ordinary Fabric workloads; only separately authorized custody operators
may manage/review the sink.

Proposed, **not implemented** acceptance thresholds: rotation daily or
100 MiB per file (whichever first); 30-day retention with securely managed
offsite copies and access audit; warning above 80% capacity, critical at
90%, stop new credential issuance/fail closed at 95% or write failure
rather than silently discard audit. Define and exercise a saturation,
rotation, recovery and read-back runbook before storing any real
credentials. No audit device, mount, PVC, sidecar, sink, egress or
credential is created by this source increment.

## Future gates

- Internal CA trust and HTTPS behavior must be physically verified after
  an explicitly authorized GitOps release, with no client activation.
- Credential-bearing Fabric ESO, workload authentication, per-tenant roles,
  audit device and the browser/Hermes/BFF bridge remain separate reviewed
  tasks. See [ADR-041](../../../../docs/adr/041-separate-openbao-fabric-secret-plane.md).
- [#4049](https://github.com/charchess/vixens/issues/4049) requires
  independent native Raft snapshot, encrypted replication and successful
  isolated restore **before any real Fabric runtime/tenant credential**.
- [Bootstrap and restore runbook](../../../../docs/troubleshooting/txo-fabric-openbao-bootstrap-and-restore.md)
  and [tenant provisioning contract](contracts/tenant-provisioning-boundary.md)
  remain authoritative for later phases. Reference-only
  `policies/*.hcl.example` are not installed. #3728 stays open until
  its full runtime/tenant/audit/recovery acceptance.
