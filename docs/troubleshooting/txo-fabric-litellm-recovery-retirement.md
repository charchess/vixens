# TXO Fabric LiteLLM — declarative retirement of incident-only recovery requests

**Issue:** #4387. **Scope:** generic tenant AI gateway schema lifecycle; not CSI/PVC, OpenBao, or physical cluster repair.

## Contract

New tenant declarations must not contain historical `migrationRetry`,
`migrationRecovery`, `migrationRecoveryFailedGeneration`, or
`migrationRecoveryTransitions`. A new TenantBundle starts with schema
migration attempt zero and retains the UID-bound Job evidence in status.

The old fields are accepted temporarily for already-stored, authorized
recovery requests. Once the last successor has **succeeded** and the
controller has retained the terminal UID-bound evidence, GitOps may retire
**all** request fields together and set `migrationRetry: 0` (or omit it).

The schema's CEL permits only that whole-request removal or the original
immutable/append-only updates. CEL cannot inspect runtime status; the
controller is responsible for safely refusing early/incomplete retirement.
No controller creates a new Job merely because the request fields vanish.

For a prior `EvidenceMissing` attempt, a successor's terminal success
alone is insufficient. The operator looks up the consumed historical
recovery ConfigMap marker by the exact tenant-scoped identity, prior
attempt and Job UID, verifies its deterministic name and controller
owner UID, and then accepts the archived missing attempt. A missing,
foreign, contradictory or ambiguous marker fails closed, as does a
missing/incomplete successor, replay, wrong UID or status ordinal gap.
After TTL removes a previously completed Job, its UID-bound terminal
status remains evidence; a Pending Job with missing executor never does.

This is a transition of **declarative requests**, not a data or database
migration. Do not reset status, delete historical Jobs/markers or change
database/PVC identity to make the transition pass.

## Ordering and rollout

1. Merge the source/controller/CRD compatibility contract and allow its
   generated image pin to converge in a separate reviewed delivery.
2. Only then remove incident-only fields from existing TenantBundle
   manifests in a follow-up GitOps PR, with terminal evidence and exact
   consumed recovery markers required by the controller.
3. On a clean cluster the same generic desired state starts at attempt 0.
   On an existing cluster a terminal migration history remains immutable
   in status. Any incomplete or unverifiable history blocks reconciliation
   instead of synthesizing success or replaying a Job.
4. Keep the external production promotion and physical validation gates
   separate; source CI cannot prove the state of historical Job/ConfigMap
   objects in a specific cluster.

**Rollback:** Revert the compatibility controller/CRD and its subsequent
manifest cleanup as one staged release only after checking whether a
stored TenantBundle has already retired a historical retry declaration.
An older controller that strictly requires `migrationRetry == latest.Attempt`
may reject those already-retired CRs. Git rollback alone does not
reset the Kubernetes status or recreate TTL-removed Jobs.

## Tests and limits

Go regressions test two different tenants, two reconciles, successful
terminal retirement after consumed exact marker, old history with attempt
one, absent/foreign/altered marker, Pending/Failed successor, ordinary
succeeded retry retirement, and the absence of any new migration Jobs.
They do not prove Kubernetes CEL admission until CRD generation and
validation CI, nor live schema migrations. The old static hAIrem/Indiba
GitOps declarations are intentionally unchanged in this source-only PR
to avoid an out-of-order CRD/controller rollout.
