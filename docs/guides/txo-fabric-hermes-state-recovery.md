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

## Dedicated upgrade canary: what exists and what does not

As of 2026-10-08, `AgentRuntimeProfile/hermes-upgrade-canary` exists in GitOps
with `truenas-iscsi-retain`, but **there is no declared AgentIdentity bound
to it**. The previously documented `fabric-smoke-upgrade-probe` does **not**
exist in current Git. `TenantBundle/fabric-smoke` is `Parked` and declares
no agents; `hairem-sandbox` is not a validated disposable test cell. Neither
`hairem` nor `indiba` may be casually treated as disposable.

Consequently, merging a profile-image pin PR is **not** an executed runtime
canary and supplies **zero** physical acceptance evidence. Provisioning a
dedicated isolated throwaway canary agent with a configured Hindsight tenant,
independent memory bank and retained test PVC requires its own authorized
GitOps change. Do not reuse customer agent keys, banks or private PVCs.

The canary must run both a fresh instance and state originally written by the
previous accepted runtime, using the same real s6 gateway, model path, toolset
policy, plugin discovery and Hindsight mode as the candidate production path.

The shared `hermes-default` profile must not move ahead before this validation.


## Minimum upgrade gate

Before promoting a new Hermes runtime image for durable Client 0 agents:

1. **first provision and prove** an isolated GitOps-managed disposable
   test agent (not currently declared), with Hindsight enabled and a unique bank;
2. on that canary, create representative non-secret retained state with
   the currently accepted runtime: config, a local skill and cron/background
   state that Fabric expects to preserve;
3. record the source PVC UID and current runtime image;
4. create a point-in-time CSI checkpoint using
   `VolumeSnapshotClass/truenas-snapshot-retain`;
5. move only `hermes-upgrade-canary` to the candidate image through Git/PR,
   then ensure an actual AgentIdentity selects the profile;
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

## Official Hermes cutover and GitOps release gates (2026-10-08)

The official-runtime bootstrap source has merged as #3940. It keeps the
current `hermes-default` pinned to the older TXO-derived Hermes image;
`nousresearch/hermes-agent:v2026.9.24` is the **unmodified upstream**
candidate and Hindsight dependencies come from a separately SHA256-pinned
OCI plugin bundle, prepared by initContainer on each Pod creation. The
agent's `SOUL.md` and private state remain owned by Hermes/the agent.

**Generated PRs have deliberately different roles:**

1. **#3942** pins the *operator* image built from #3940. Complete its CI
   and merge first. The intermediate source-only dev snapshot
   `dev-v2026.10.3940` is **not** an acceptable production promotion:
   the repository promotion workflow requires the built operator image
   to match the snapshot source. Verify the resulting operator-pin
   `dev-v*` candidate, then explicitly promote it only with human
   authorization and validate existing tenants/agents after ArgoCD sync.
2. **#3941** changes only `hermes-upgrade-canary` to the official
   runtime and pins the Hindsight plugin OCI by digest. Review its
   CI after #3942; if merged and explicitly promoted, it only creates
   a **candidate profile**. It does **not** run or validate Hermes
   automatically without a bound canary AgentIdentity.
3. A separately reviewed/approved GitOps test-agent declaration
   is needed for genuine physical acceptance. Its first run must verify
   s6 health, Hindsight `local_external` retain/recall, scoped bank,
   credential isolation and routing via tenant LiteLLM, personal
   `SOUL.md`, sessions, private/local skills, cron, group skills,
   and expected human endpoint as applicable.
4. Complete the retained-state checkpoint + separately restored PVC
   test described above (#3688). An upstream **version-tag match**
   is not sufficient to prove the migration of previously persisted
   state, plugin imports or rollback compatibility.
5. **Do not flip `hermes-default` for the whole fleet.** Instead,
   after acceptance, introduce a distinct production candidate profile
   (e.g. `hermes-official-v2026-09-24`) mirroring the accepted
   `hermes-default` resources, security ceilings and storage policy,
   differing only in runtime image and versioned Hindsight bootstrap.
   Opt in **one** non-critical retained agent by changing its
   `AgentIdentity.spec.runtime.profileRef` through GitOps after
   checkpointing that specific PVC. Validate the agent and reverse
   the binding on failure, restoring its data snapshot when needed.
6. Migrate hAIrem/Indiba agents **individually** with proof of
   per-agent PVC UID, `SOUL.md` integrity, session/skills, Hindsight
   bank ID, credentials and access boundaries. Keep untouched agents
   on legacy `hermes-default`. Only when the fleet is individually
   accepted may a later explicit PR repoint `hermes-default`
   for newly provisioned agents and retire the derived-image pipeline.

**Storage caveat:** `hermes-upgrade-canary` uses
`truenas-iscsi-retain`; existing agent PVCs may use different storage
classes. Never attempt to rebind/recreate/migrate their PVC implicitly
as a side effect of a runtime profile reference change. Record the
existing PVC/storage class and preserve the bound object; a
production opt-in profile must be designed to be compatible with the
actual existing storage contract.

**Release discipline:** promotions operate on complete immutable Git
snapshots, not individual PR files. A green GitHub CI and the existence
of `dev-v*` do not authorize a production rollout. Record the
snapshot, post-sync observations and restore evidence for each stage.
Update `prod-working` only after explicit operator approval.

The functional-role implementation in draft #3938 is separate. Rebase
it against the official-Hermes implementation and validate native
reference skills without editing `SOUL.md` before enabling functional
bindings for real agents.

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
