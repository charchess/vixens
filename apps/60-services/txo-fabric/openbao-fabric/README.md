# OpenBao Fabric — empty private GitOps POC (#3728)

**Phase 1 (this PR):** a single, empty OpenBao 2.6.2 server using integrated
Raft storage on a dedicated retained TrueNAS iSCSI volume. Both TXO dev/prod
Kustomize compositions select this component. Argo dev follows main; prod
follows prod-stable and requires a separate human-approved promotion.

- One StatefulSet `openbao-fabric` in existing restricted namespace
  `txo-fabric-system`. Exactly one replica, no HA or auto-unseal. The official
  image is version-tagged `quay.io/openbao/openbao:2.6.2`; digest pinning can
  be hardened in a separate change.
- One `data` PVC template of 5Gi RWO with `truenas-iscsi-retain` and
  Retain/Retain StatefulSet claim retention policy. The generated PVC is
  `data-openbao-fabric-0`. Never automatically prune/delete the claim or PV.
  CSI storage/class/PV runtime binding still needs post-merge confirmation.
- Raft stores data only at `/openbao/data` on the claim. Non-secret HCL
  configuration comes from a ConfigMap; `/tmp` is ephemeral scratch.
- `openbao-fabric` is a private ClusterIP Service, TCP/8200 only. No
  Ingress, NodePort, LoadBalancer, external DNS, network client or TLS yet.
- Exact-selector NetworkPolicy has **zero allowed ingress and egress**,
  alongside the existing namespace default-deny. No direct Hermes, browser,
  BFF or tenant client. OpenBao API is HTTP without TLS: **do not add any
  runtime network clients until the separately reviewed TLS phase**. A
  human-only, locally scoped port-forward to the *pod* during a separately
  authorized bootstrap ceremony is the only proposed access path.
- No Kubernetes API token mounts, root credential, Secret, ExternalSecret,
  ESO store, init/unseal Job, admin policy, audit sink or workload auth.
- Non-root, dropped capabilities, seccomp RuntimeDefault, read-only root FS.
  `fsGroup: 1000` is used for retained RWO volume writes, subject to real CSI
  verification. Requests 100m CPU/128Mi RAM; limits 500m CPU/512Mi RAM.
  Ordinary scheduler placement, no hard node pinning or Peach operation.

## Expected state after Argo convergence

An **uninitialized/sealed** OpenBao responds to `/v1/sys/health` with a
non-200 status. The readiness probe therefore intentionally leaves the pod
`Running/NotReady` and the Service without ready endpoints until a later
human-authorized init/unseal. This is **not** a deployment failure. A
CrashLoop, unbound PVC or image pull failure **is** a deployment failure
and must be diagnosed without printing credentials. There is no liveness
probe restarting a legitimately sealed process.

## Four phases

1. **Delivered in source:** empty private GitOps substrate only. Argo
   convergence and actual PVC/NetworkPolicy observation remain unverified.
2. **Separate approved human bootstrap:** manual init/unseal and internal
   health proof; root/unseal/recovery material stays out of Git, chat, CI,
   logs, Kubernetes Secrets and workloads.
3. **Separate reviewed credential activation:** TLS, audit, Fabric-only
   `openbao-fabric-store`, exact short-lived workload/tenant policies and
   first POC Fabric credentials. Browser and private bridge gates stay false
   until their separately accepted POC path.
4. **Parallel #4049 recovery:** native Raft snapshot, independently
   protected encrypted backup/replica, successful isolated restore.
   **No actual Fabric HMAC, Hermes API credential or tenant data is trusted
   to this substrate before restore acceptance.**

[ADR-041](../../../../docs/adr/041-separate-openbao-fabric-secret-plane.md)
defines Infra/Fabric authority. Existing Infra `ClusterSecretStore/openbao`,
NAS backend, token and historic TXO OpenFGA/WebUI consumers remain unchanged.
No new consumer falls back to Infra. Tenant IDs remain immutable and
single-tenant scoped, not authorization from user/group changes.

[Bootstrap and restore runbook](../../../../docs/troubleshooting/txo-fabric-openbao-bootstrap-and-restore.md)
and [tenant provisioning contract](contracts/tenant-provisioning-boundary.md)
are prerequisites for later credential-bearing phases.
The policies in `policies/*.hcl.example` are strictly reference-only,
not automatically installed. #3728 remains open until its full physical,
tenant-security, audit and backup acceptance is demonstrated.
