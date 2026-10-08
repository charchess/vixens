# TXO Fabric — pinned official Hermes runtime channels

Scope: #3944, #3947, #3939, #3688. WORKFLOW.md continues to govern PRs,
GitOps snapshots and *explicitly approved* production promotion.

## Three permanent pointers, not three test infrastructures

`HermesRuntimeRelease` is immutable: an official upstream Hermes image **digest**
and a separately pinned Hindsight plugin-bundle **digest**. Releases are historical
data objects, not deployed Pods. `AgentRuntimeProfile` selects one by
`spec.releaseRef`; only a bound `AgentIdentity` creates an agent runtime.
There is no tenant, namespace, bank, volume or pod per upstream version.

| Profile pointer | Ownership | Advancement |
| --- | --- | --- |
| `hermes-stable` | Selected official upstream baseline, intended for approved client use | Human review; prior canary history, actual physical state/recovery evidence, explicit owner approval and exact GitOps snapshot |
| `hermes-canary` | Newer pinned upstream candidate under qualification | Human-reviewed pointer update; must have appeared in committed edge history; physical validation occurs **after** selection, before stable |
| `hermes-edge` | Most recent **selected** official upstream release | Review-only CI-generated PR; versioned upstream tag resolved to a real OCI digest; never a floating `:latest` or `:main` |

The same immutable release can be referenced by multiple channels and can advance
edge → canary → stable **without** rebuilding it. Canary and edge may legitimately
reference the same release when the newest published upstream version is being tested.
The target is **three channel objects permanently**, not an unbounded succession
of named `hermes-official-v...` profiles.

## Initial selection — 2026-10-08, software-only (unbound)

- `hermes-stable`: official `v2026.9.24` engine digest
  `sha256:fca358f12efd65bfaaca05884166f15c0e2788375ca30d77061ac1ebc96452b7`.
- `hermes-canary` and `hermes-edge`: official `v0.21.6` engine digest
  `sha256:9774f4f39a9bb8c2f68ce728ed5e99ddbad282163be56764afacf88ed952b784`,
  verified in the annotated Git tag receipt
  https://github.com/NousResearch/hermes-agent/releases/tag/v0.21.6.
- Each points at the pinned platform Hindsight plugin digest
  `sha256:33060387a98b661aee0ed39dcc6ad19d231a7dd3036dfe429818d8e74f3e6bb5`.

**These are upstream source selections, not proven Fabric-compatible deployments.**
In particular, Hermes `v0.21.6` changed plugin hosting and requires actual
discovery/s6/Hindsight tests. Merely registering `hermes-stable` does NOT certify
it for binding to production agents. The initial three-channel PR must carry
`Hermes-Channel-Bootstrap: true` and may not change any tenant/agent declarations.

Existing 17 Indiba/hAIrem beta AgentIdentity resources continue to use legacy
`hermes-default` with their existing custom image. They do not change as part
of channel declaration. Retain `hermes-dev` and `hermes-upgrade-canary` as
**temporary compatibility profiles**; do not nominate them further or silently
delete them. Retire only after checking all consumers and an explicit safe
migration. The initially configured stable profile preserves the legacy storage
class; do not switch a retained agent without checking real PVC UID/class and
the desired retention policy.

## Selection and CI behavior

The pinned Hindsight plugin build checks the **selected versioned official
Hermes tag** by resolving the registry manifest digest, smoke-tests that
image with the plugin payload and publishes a review-only **edge** PR. The
`nominate_edge.py` script may create a new immutable release and move **only**
`hermes-edge.releaseRef`. It cannot touch the other pointers or agents.

To select a different official release, manually dispatch
`.github/workflows/build-txo-hermes-hindsight-plugin.yaml` with
`hermes_tag: vX.Y.Z` after confirming the release exists. The workflow's
default is `v0.21.6` at bootstrap; it is a pinned *selection*, not a
background subscription to a floating upstream `latest`.

The required Validation Summary tests the channel guard:

1. First declaration: register all three unbound channels, human-reviewed PR
   with `Hermes-Channel-Bootstrap: true`; no tenant/agent manifest changes.
2. Edge pointer: bot may propose exact digest in a PR; immutable releases
   cannot be edited, only new ones created.
3. Canary pointer: human review, same release must have appeared in an earlier
   committed edge pointer. This selects a candidate *for* qualification.
4. Stable pointer: human-only, must have appeared in earlier committed canary
   history **and** supply all evidence links and operator approval:

       Hermes-Physical-Evidence: https://github.com/charchess/vixens/issues/3688#issuecomment-ACTUAL_ID
       Hermes-Recovery-Evidence: https://github.com/charchess/vixens/issues/3688#issuecomment-ACTUAL_ID
       Hermes-Owner-Approval: https://github.com/charchess/vixens/issues/3947#issuecomment-ACTUAL_ID
       Hermes-GitOps-Snapshot: dev-vYYYY.MM.PR

These are templates, not acceptance records. The guard validates Git history,
OCI digest syntax and **evidence URL format**, not the substance of physical
tests or whether a human actually approved. Reviewers must verify evidence
and enforce authorization through workflow/branch protection.

## Bounded qualification and safe brownfield migration

Reuse a fixed, small set of isolated qualification agents; optionally park
their supporting tenant and clean temporary restored PVCs after proof.
**Never create a tenant or pod for every release**, or attach customer PVCs
to experimental pointers.

Before advancing stable: prove actual official s6 gateway and initContainer
plugin loading, scoped LiteLLM/CPA inference, Hindsight retain/recall,
SOUL.md ownership, skills, cron, sessions, restarts and old-state migration.
Take a retained-state checkpoint and restore it to a **different PVC**.
Rollback means runtime **plus** compatible data state, not just changing
`releaseRef`. Keep `hairem`/`indiba` production beta identities and banks
unchanged until explicit per-agent migration. Changing a profile pointer
will roll every identity *currently bound* to it.

The runtime channels above are independent of GitOps `dev-v*`, `prod-v*`
and `prod-stable`: production must still receive explicit authorization for
the **exact immutable complete-platform snapshot**. No CI/pointer PR itself
promotes production. `prod-working` remains a manual known-good bookmark.
