# TXO Fabric — Hermes private-state upgrade and recovery

Scope: #3688, #3944, #3947, #3964. The canonical release-channel model is
[txo-fabric-hermes-release-channels.md](txo-fabric-hermes-release-channels.md).
`WORKFLOW.md` controls explicit GitOps production promotion.

## No infrastructure per Hermes version

There are **three permanent runtime pointers**:
`hermes-stable`, `hermes-canary` and `hermes-edge`. Each selects a pinned
immutable official `HermesRuntimeRelease` (Hermes image digest and Hindsight
plugin digest). A pointer alone does not create a pod, tenant or PVC.

Spark (hAIrem `ten00001-usr000001-agt00011`, #3964) is the nominated
**existing** edge pilot. Its desired binding is merged into Git, but it must
not be described as physically tested or deployed until the exact snapshot
has been explicitly promoted and the cluster inspected. Other beta agents
remain on the backward-compatible `hermes-default`.

Legacy `hermes-dev` and `hermes-upgrade-canary` profiles are compatibility
artifacts, **not** the destination for new version qualification.
`fabric-smoke` is parked and currently declares no canary AgentIdentity;
`fabric-smoke-upgrade-probe` must not be referenced as an existing resource.

## Separate state and runtime concerns

- Private Hermes state: per-AgentIdentity PVC mounted at `/opt/data`,
  including agent-owned SOUL.md, config, sessions, local skills and cron.
- External Hindsight bank: tenant-backed service and bank identity, not a
  replacement for a snapshot of local `/opt/data`.
- Shared tenant/user/group workspace: not a private-agent migration target.
- Credentials/model access: tenant LiteLLM/CPA, scoped broker, secrets and
  authorization, not disposable just because an agent workspace is empty.
- Runtime: official pinned OCI image, separately pinned Hindsight plugin,
  platform toolset policy and release pointer.

Changing a Hermes version or channel **must not implicitly recreate a PVC**.
A runtime rollback alone does not undo an on-disk schema migration.
Persistent storage policy is moving to tenant-owned `spec.agentStorage`
(#3967), independent of runtime release pointers.

## Initial beta rollout while agent workspaces are empty

On 2026-10-08 the owner confirmed current hAIrem/Indiba agent workspaces
contain no valuable business data. **Only** the private agent workspaces
explicitly inventoried as empty may be intentionally re-provisioned under
a separately approved rollout; this is not permission to destroy tenant
PostgreSQL, Hindsight bank data, shared files, CPA OAuth state or other PVCs.

Before authorizing a production snapshot:
1. Verify `main`, all operator source/image pin PRs, and the immutable
   `dev-v*` snapshot; compare it with current `prod-stable`.
2. Observe the exact live AgentIdentity, namespace, PVC name, UID,
   StorageClass, bound PV, PV reclaim policy, consumers and ownerReferences.
3. Confirm **per PVC** that it is disposable/empty and that no other service
   or user-owned state is stored there; record a reversible procedure or
   explicitly acknowledge re-provisioning cannot recover the old contents.
4. Plan controlled detachment, exclusive RWO access, and **separately authorized**
   replacement of only those private PVCs that need a new Retain StorageClass.
   Kubernetes PVC `storageClassName` is immutable. Merely updating the tenant
   storage policy must leave existing PVCs untouched.
5. Promote the exact candidate only after the owner's explicit authorization,
   then verify ArgoCD and the live operator image.
6. Check Spark's official s6 startup, plugin discovery, Hindsight scoped
   retain/recall, LiteLLM→CPA inference, personal SOUL.md/skills, sessions,
   cron and authenticated human entry. Confirm other agents stayed unchanged.

This early empty-state exception is **not** the normal upgrade procedure
once real customer data accumulates.

## Populated-agent upgrade and recovery gate

Before promoting an incompatible or uncertain Hermes version for an agent
with valuable retained state:

1. Identify exact agent, prior runtime release, PVC UID, linked PV and
   Hindsight bank; create representative non-secret local state.
2. Capture a checkpoint before the runtime changes, preferably using the
   cluster's CSI `VolumeSnapshotClass/truenas-snapshot-retain`.
3. Record snapshot ReadyToUse, source PVC UID, VolumeSnapshotContent and
   recovery references without exposing secrets.
4. Advance only the reviewed pinned edge/canary release pointer appropriate
   for that cohort; test actual s6 gateway, plugin imports, Hindsight,
   LiteLLM/CPA, toolsets, skills, cron, sessions and restarts.
5. Prove representative state survives the image change and continued
   operation of the same retained PVC, or document incompatibility.
6. **Restore the snapshot to a different PVC** and test the old runtime with
   that restored state. Never attach two writers to one RWO volume or
   overwrite the live retained PVC merely to prove a restore.
7. Record physical evidence under #3688/#3947. Only then consider human-reviewed
   canary → stable promotion, followed by individually planned client rollout.

Runtime-owned config backups are useful defense in depth but cannot replace
a private-state checkpoint; #3743 tracks that ownership. #3742 covers runtime
probe robustness.

## Special case: legacy-profile adoption

`runtime.storage.adoptLegacyProfile` changes the runtime's effective home
to an existing `profiles/<agentKey>` directory via `subPath`. Even though
the operator does not copy/flatten that state, startup can change config or
Hindsight metadata. **If that old profile holds valuable data**, checkpoint
it *before* promoting adoption. The first acceptance needs old state,
upgrade, separately restored PVC and verified rollback per #3688.

A retained PV's `Retain` reclaim policy means the backend should survive
PVC deletion; it is **not itself a backup** and does not guarantee that the
PVC object survives a controller garbage-collection race. #3824 covers the
separate ownerReference fix.
