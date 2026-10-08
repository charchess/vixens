# TXO Fabric — Hermes immutable releases and deployment channels

Scope: #3944, #3947, #3939, #3688. WORKFLOW.md governs all merges, GitOps
snapshots and production promotion. These runtime channels are **not** Git
branches or full-platform dev-v/prod-v/prod-stable release tags.

## Contract

HermesRuntimeRelease is immutable: an upstream official Hermes OCI digest plus
a separately pinned Hindsight plugin digest. A new combination means a NEW
release resource, never an edit of an existing release. AgentRuntimeProfile
contains a mutable spec.releaseRef and still owns storage, capabilities,
resource limits and scheduling. The only way to change a running agent's
channel is to select that profile in AgentIdentity.spec.runtime.profileRef.

Permanent channels are hermes-dev, hermes-test and hermes-stable, with **the
same release object and exact digests** moving from one channel to another.
None is an intrinsic state of the operator. Promotion only moves GitOps
pointers. History must associate the release, pointer, Git commit and the
precise platform snapshot tag.

## Phased introduction

1. **Dev, current PR**: register a real immutable release and hermes-dev
   profile. No AgentIdentity is changed. The Hindsight publisher may only
   nominate dev in a new reviewed PR; it cannot change test, stable or legacy
   channels. The release is a software candidate, NOT physical acceptance.
   Dev is only for isolated/disposable identities, never customer PVCs.
   Retain storage is intentional for a controlled old-state and restore
   rehearsal; cleanup of test state must be explicit.

2. **Physical test preparation**: create a dedicated isolated AgentIdentity
   via GitOps; currently fabric-smoke is Parked and has no agent. Use the
   legacy retained-storage profile to create representative old-version
   local state, then record identity, PVC UID/class, image and Hindsight bank.
   Create a CSI VolumeSnapshot using truenas-snapshot-retain, and switch only
   this isolated identity to hermes-dev in a reviewed, reversible PR. Prove
   official s6 gateway, initContainer plugin loading, LiteLLM/CPA inference,
   Hindsight isolated retain/recall, private SOUL.md, sessions, local/shared
   skills, cron, restart, unchanged PVC UID and bank. Restore a checkpoint
   to a DISTINCT disposable PVC; never attach two writable runtimes to the
   same RWO PVC. Prove rollback runtime plus compatible data.

3. **Test**: after real linked dev/canary and recovery evidence, create
   hermes-test with the same exact releaseRef previously selected in the
   committed hermes-dev history. Use the reviewed isolated retained-canary
   resource/toolset policy. Bind only the dedicated physical canary and
   explicitly authorized pilots. Every later change to hermes-test requires
   a new reviewed PR and acceptance evidence.

4. **Stable**: only after an accepted real hermes-test pilot, proven recovery
   and exact release/snapshot owner approval, create or advance hermes-stable
   to the releaseRef previously selected by hermes-test. An existing stable
   pointer change may restart EVERY bound agent; authorize that blast radius
   separately. No bot or workflow automatically advances test/stable.
   Production GitOps promotion still requires separate, explicit operator
   authorization of a precise dev-v snapshot via promote-prod.yaml.

## Merge gate and audit evidence

Validation Summary incorporates the Hermes channel guard in its Production
Configuration family. This blocks bot-authored test/stable pointer PRs,
requires prior committed dev/test history and syntax-checks acceptance links.
For a test/stable change, supply these lines in the PR description with
ACTUAL GitHub issue comment URLs, after performing real tests:

    Hermes-Physical-Evidence: https://github.com/charchess/vixens/issues/3688#issuecomment-ACTUAL_ID
    Hermes-Recovery-Evidence: https://github.com/charchess/vixens/issues/3688#issuecomment-ACTUAL_ID

For stable also include:

    Hermes-Owner-Approval: https://github.com/charchess/vixens/issues/3947#issuecomment-ACTUAL_ID
    Hermes-GitOps-Snapshot: dev-vYYYY.MM.PR

Those are templates, not accepted evidence. Automated validation verifies
reference format and Git history, **not** whether a physical test passed or
an owner truly approved it. Human reviewers must read each linked record,
verify the exact full digests, dates, individual canary/PVC/bank identity,
checkpoint, restore on a separate PVC and rollback. Never paste private
SOUL.md, secrets, sessions or token contents. No physical acceptance has
yet been recorded.

Example read-only audit commands (example snapshot is NOT a PASS claim):

    git log --oneline -- apps/60-services/txo-fabric/operator/config/profiles/hermes-test.yaml
    git show dev-v2026.10.3948:apps/60-services/txo-fabric/operator/config/profiles/kustomization.yaml
    kubectl get agentruntimeprofiles,hermesruntimereleases

## Brownfield compatibility and rollback

The 17 durable beta agents in hAIrem and Indiba currently reference legacy
hermes-default, sometimes by implicit default. They must NOT be switched
or recreated to introduce channels. Retain the existing profile, PVC UID,
StorageClass, Hindsight bank, SOUL.md, secrets and permissions.
hermes-upgrade-canary remains temporarily for old-state test compatibility.

After physical pilot and verified restore, migrate specifically authorized
AgentIdentity resources one by one through GitOps with pre-change snapshots,
PVC comparison and real rollback tests. Do NOT make a fleet-wide change by
editing hermes-default. Only once every consumer has moved to hermes-stable
should the CRD's implicit default and old aliases be retired by a separate
tested source/operator image pin and GitOps migration. Do not leave two
drifting production aliases.

Rollback is an accepted runtime plus compatible recovered data, not simply
reverting releaseRef. #3688 and the retained-state recovery guide are the
authoritative physical acceptance backlog; never infer a PASS from CI,
Kubernetes Ready, a profile resource or an unbound canary.
