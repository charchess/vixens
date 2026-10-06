# Active documentation drift audit — 2026-10-06

**Issue:** #3546  
**Status:** Completed — active-document cleanup and final post-merge drift classification completed.

## Goal

Make it safe for a human or automated agent to resume Vixens from repository state without depending on private conversation history or following stale operational instructions.

The audit treats:

1. root `WORKFLOW.md` / `AGENTS.md` as canonical process contracts;
2. GitHub Issues as scope/acceptance truth;
3. `vixens roadmap` as planning truth;
4. current workflows/manifests as executable truth;
5. current architecture/guides/references as explanatory material;
6. ADRs/post-mortems/reports/archive as historical records when explicitly marked or inherently dated.

## Reviewed inventory

### Root / governance

Reviewed:

- `README.md`
- `WORKFLOW.md`
- `AGENTS.md`
- `CLAUDE.md`
- `GEMINI.md`
- `docs/README.md`
- `docs/architecture.md`

### Workflow / CI implementation

Reviewed current semantics of:

- `.github/workflows/validate.yaml`
- `.github/workflows/auto-tag-dev.yaml`
- `.github/workflows/promote-prod.yaml`
- `.github/workflows/mark-prod-working.yaml`
- `.github/workflows/merge-queue.yaml`
- TXO Fabric operator CI/build workflows where they affect release semantics
- active GitHub rulesets on `main`

### Guides

Inventory reviewed under `docs/guides/`:

- adding-new-application
- backup-restore-pattern
- bronzification-action-plan
- gitops-workflow
- merge-queue-configuration
- pattern-config-syncer
- prod-hibernation
- promotion-workflow
- quality-reports
- secret-management
- security-observability
- sizing-migration
- task-management
- txo-fabric-hermes-state-recovery
- txo-fabric-human-entry
- workflow-concurrency

### Procedures

Inventory reviewed under `docs/procedures/`:

- deployment-standard
- application-testing
- adding-new-talos-node
- dev-hibernation
- scout-mode-sizing

### References

Inventory reviewed under `docs/reference/`, with direct attention to workflow,
sizing, maturity and source-of-truth documents:

- RESOURCE_STANDARDS
- app-golden-standard
- quality-standards
- maturity-standards-matrix
- guaranteed-qos-sizing
- application-deployment-standard
- argocd-sync-waves
- sync-waves-implementation-plan
- workflow-state-machine
- multi-agent-orchestration
- configuration-management-strategy
- dependency-management
- technical-debt
- remaining reference files were included in repository-wide drift searches for retired workflow/tool terminology.

### Agent/tool guidance

Reviewed:

- `.opencode/skills/vixens-gitops/`
- `.opencode/skills/vixens-cluster/`
- `.opencode/skills/vixens-argocd-safety/`
- `.opencode/skills/vixens-troubleshoot/`
- `.opencode/skills/vixens-secrets/`
- `.opencode/skills/vixens-maturity/`
- `.opencode/skills/vixens-app-patterns/`
- `.opencode/skills/vixens-kubernetes-patterns/`
- workflow-impacting helpers under `scripts/`

### ADR workflow/governance set

Reviewed directly or by status/index:

- ADR-007 Renovate trunk-based workflow
- ADR-008 / ADR-009 superseded branch workflows
- ADR-017 pure trunk-based decision
- ADR-019 file / historical Renovate Discord approval workflow
- ADR-023 maturity system
- ADR-029 maturity/current-platform alignment
- ADR-030 ADR governance
- ADR-031 through ADR-037 TXO Fabric contracts relevant to current agent onboarding

## Fixed in #3873

- explicit source-of-truth map and cold-start checklist;
- Project `Status` / `Priority` / `Target` semantics;
- handoff/reprise contract;
- exact-candidate human authorization for production;
- `dev-v*` clarified as immutable snapshot identity, not dev-validation proof;
- task-management priority moved from retired labels to Project field;
- production concurrency guide aligned with the real serialized workflow;
- TXO Fabric README aligned with current Hindsight gateway embeddings and generated Authentik structural IAM.

## Fixed in #3874

- complete local/pre-commit → PR → CI → dev → production lifecycle;
- adaptive local validation and honest validation reporting;
- `Validation Summary` documented as authoritative merge gate;
- legacy `scripts/utils/gp` retired;
- OpenCode GitOps/cluster skills stopped redefining workflow or teaching force-sync/manual production-tag operations;
- ADR-017 historical operational snippets fronted with a current implementation warning;
- application-testing procedure aligned with PR/GitOps flow;
- obsolete sizing and bronzification runbooks retired to compatibility stubs;
- legacy sizing audit helper made read-only.

## Current slice findings/fixes

This slice addresses:

- replacement of the stale March-2026 goldification snapshot masquerading as global architecture;
- merge queue documentation: current repository rules require PR + strict `Validation Summary`, but do **not** require a merge queue; the `merge_group` workflow is future-ready/dormant;
- retirement of the 2024 sync-wave implementation plan;
- technical-debt tracking language moved from Beads to GitHub Issues/Project;
- hibernation helper protected from committing on `main`;
- sync-wave validation helper corrected to repository root and branch/PR instructions;
- Renovate Discord approval ADR deprecated because current `renovate.json` auto-merges non-major updates after required checks;
- Renovate application secret documentation moved to OpenBao + External Secrets;
- maturity/config-syncer/Guaranteed-QoS references aligned with OpenBao and sizing-v2 per-container labels.

## Verified current merge policy

As of 2026-10-06:

- repository owner is the personal account `charchess`;
- active `main` ruleset requires Pull Requests;
- required check is `Validation Summary`;
- strict status-check policy is enabled;
- merge methods allowed by ruleset are merge/squash/rebase;
- no active ruleset requires merge queue;
- `.github/workflows/merge-queue.yaml` listens to `merge_group` but does not itself enable the GitHub feature.

Therefore documentation must not present merge queue as a current mandatory developer workflow.

## Intentional historical matches

The following categories may legitimately contain retired commands/names:

- superseded ADRs such as ADR-008 / ADR-009;
- ADR-017 historical implementation/rollback snippets, now explicitly fronted by current-runbook guidance;
- dated post-mortems and incident reports;
- archived documents;
- management/report snapshots.

Their existence is acceptable only when they are not linked/presented as current operational instructions.

## Application-document cleanup completed

The follow-up application pass verified each stale current-secret reference against
the executable manifests rather than replacing terminology blindly.

Current application docs were aligned with OpenBao + External Secrets Operator for
Penpot, Vikunja, Docspell, NetBox, Linkwarden, Gluetun, Mosquitto, cert-manager,
Authentik, Vaultwarden, ExternalDNS/Gandi, Firefly III, Netvisor, Redis Shared,
Alertmanager, MariaDB Shared, PostgreSQL Shared, Firefly III Importer, Home
Assistant, cert-manager-webhook-gandi and Grafana.

`argocd-image-updater` has no current desired state under `apps/70-tools/`;
its application page is now a historical compatibility stub rather than an
invented OpenBao migration.

Remaining `Infisical` mentions in active application pages were checked on the
branch and are explicit negative/historical statements such as "do not recreate
the retired Infisical integration".

The same pass also removed the widespread retired dev-hibernation instruction
"uncomment the app from the ArgoCD kustomization". Application pages now tell
operators to inspect the current dev overlay and use the canonical Git/PR
hibernation/reactivation procedure.

Previously deferred dependencies are also resolved:
- #3707 reconciled ADR governance/catalog status, including ADR-011 as Superseded;
- #3709 backfilled the Fabric AgentIdentity/tenant/workspace ADR contracts and
  cleaned active technical Persona terminology.

## Final post-merge drift classification

A final repository-wide search was run against current `main` after the application
cleanup and follow-up drift fixes.

Classified survivors:

- `Infisical` in active application docs: only explicit negative/historical
  statements such as "do not recreate the retired Infisical integration";
- direct `git push origin main/dev` examples: only dated management/report
  snapshots or this audit's search-pattern inventory;
- manual `prod-stable` force-tag examples: only ADR-017 historical rollback text,
  a dated post-mortem, or an explicit "never do this" warning in the active skill;
- merge queue mentions in active guides: explicitly state that merge queue is not
  currently active/enforced;
- direct ArgoCD Application patch guidance in active guides: no surviving current
  runbook matches;
- retired priority labels `priority:p0/p1`: no active matches;
- active maturity secret wording: ADR-023 now surfaces ADR-029 directly and uses the
  implementation-independent externally-managed-secret criterion.

One executable (non-documentation) drift was found during the sweep:
`apps/40-network/adguard-home/base/deployment.yaml` still carries the old global
`vixens.io/sizing: G-small` label. That is outside this documentation-audit scope
and is tracked separately rather than being silently changed in a docs PR.

No remaining active-document match requires another #3546 correction.

## Search classes used

The audit used repository-wide searches around:

- `dev` / `test` / `staging` branch instructions;
- `prod-stable`, `prod-v*`, `prod-working`;
- direct `git push origin main` / old permanent `dev -> main` flows;
- Beads / Archon / Just;
- Infisical / `InfisicalSecret`;
- direct persistent `kubectl apply/edit/patch`;
- ArgoCD force-sync/prune instructions;
- old global sizing labels and current per-container sizing labels;
- merge queue assumptions.

Because GitHub code search indexing can lag immediately after merges, changed files
were also fetched directly from the branch/main when validating a specific fix.

## Closure condition for #3546

Satisfied on 2026-10-06:

1. active operational/documentation inventory reviewed;
2. workflow/tool/application drift corrected or explicitly marked historical;
3. final post-merge repository-wide drift search classified;
4. surviving retired-pattern matches are historical/negative context rather than
   current operational instructions.

#3546 may close once this final audit amendment is merged.
