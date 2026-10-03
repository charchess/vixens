# TXO Fabric Hermes retained-state upgrade and recovery

Issue #3688 defines the safety contract for Hermes image changes when an
`AgentIdentity` keeps valuable private `/opt/data`.

## State boundaries

An Hermes runtime upgrade must treat these as separate lifecycle domains:

- **private runtime state**: the AgentIdentity PVC mounted at `/opt/data`;
- **durable external memory**: tenant Hindsight and the AgentIdentity bank;
- **shared business workspace**: tenant/user/group workspace mounts;
- **shared/curated skills**: platform workspace skill libraries;
- **runtime image and platform policy**: `AgentRuntimeProfile`, toolset policy,
  gateway/integration authorization and immutable image pins.

A rollback of the container image alone is not a state rollback.

## Dedicated upgrade canary

`AgentRuntimeProfile/hermes-upgrade-canary` and
`AgentIdentity/fabric-smoke-upgrade-probe` are the retained-state canary.

The profile starts from the accepted `hermes-default` runtime contract but uses
`truenas-iscsi-retain`. Its image may move ahead of `hermes-default` only for
an explicit upgrade acceptance.

The shared `hermes-default` profile must not be changed to a new Hermes image
until the candidate has passed this canary with state created by the previous
accepted image.

## Minimum upgrade gate

Before promoting a new Hermes runtime image for durable Client 0 agents:

1. prove the candidate on ordinary disposable `fabric-smoke` first;
2. on `upgrade-probe`, create representative non-secret retained state with
   the currently accepted runtime: config, a local skill and cron/background
   state that Fabric expects to preserve;
3. record the source PVC UID and current runtime image;
4. create a point-in-time CSI checkpoint using
   `VolumeSnapshotClass/truenas-snapshot-retain`;
5. move only `hermes-upgrade-canary` to the candidate image through Git/PR;
6. after ArgoCD convergence, prove the retained PVC UID is unchanged and the
   representative state remains usable;
7. prove the Hermes gateway, Hindsight, toolset policy and inference path still
   operate through their Fabric-owned contracts;
8. if the candidate migrates state incompatibly, restore into a **separate
   disposable recovery PVC** from the checkpoint and prove the old accepted
   runtime can consume that restored state;
9. only then may the shared runtime profile be considered for a later PR.

Do not test rollback by attaching two writable runtimes to the same RWO PVC.

## Checkpoint primitive

The cluster already provides CSI snapshots for the TrueNAS iSCSI driver:

- `truenas-snapshot-retain`: retained recovery checkpoint;
- `truenas-snapshot-delete`: disposable snapshot lifecycle.

Upgrade checkpoints are operational recovery artifacts, not long-lived
application desired state. Their names must identify the agent and source
release, and acceptance evidence must record:

- source PVC name and UID;
- VolumeSnapshot name and ReadyToUse state;
- bound VolumeSnapshotContent;
- source/runtime release;
- restore PVC name/UID when recovery is tested.

Do not print file contents, tokens, cookies or secret values as checkpoint
evidence.

## Rollback semantics

Supported rollback means **accepted runtime image + compatible data state**.

If a candidate only changed executable code and the existing retained state is
known compatible, reverting the image may be sufficient. If the candidate
performed an irreversible or unknown state migration, use the pre-upgrade
checkpoint instead of assuming the old image can read the mutated live PVC.

The live retained PVC must not be destructively overwritten merely to prove a
restore. Restore into a separate disposable PVC first.

## Relation to runtime-local backups

Hermes' own config backup path is useful defense in depth but is not the whole
recovery contract. #3743 tracks correct runtime ownership for that backup path.
A CSI checkpoint protects the broader private runtime state boundary.

#3742 tracks probe robustness so transient node/runtime stalls are not mistaken
for data/runtime incompatibility during upgrade acceptance.

## Client 0 migration gate

The legacy Client 0 profiles in #3673 remain authoritative until at least one
copied representative profile has passed:

`old accepted state -> checkpoint -> Fabric canary -> candidate runtime -> restore proof`.

The first migration is copy/rehearsal only. Cutover happens only after functional
parity and recovery evidence exist.

For an AgentIdentity using the explicit legacy-profile adoption mode, take the
checkpoint **before** promoting the Git change that enables
`runtime.storage.adoptLegacyProfile`. The adoption itself does not copy or delete
the legacy directory: `profiles/<agentKey>` is mounted as the runtime's
`/opt/data` through `subPath`. Bootstrap may still update runtime-owned config,
permissions and Hindsight provider metadata inside that retained profile, so the
pre-cutover checkpoint remains mandatory.
