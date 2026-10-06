# Active documentation drift audit — 2026-10-06

**Issue:** #3546  
**Status:** In progress — central workflow/architecture/reference surfaces reviewed; application-document cleanup remains.

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

## Remaining active drift discovered

Repository-wide searches still identify application documentation that presents
Infisical as a current dependency/secret backend. These require a separate
application-document pass against each application's current manifests before
#3546 can close.

Examples discovered include:

- `docs/applications/70-tools/penpot.md`
- `docs/applications/70-tools/vikunja.md`
- `docs/applications/70-tools/docspell.md`
- `docs/applications/60-services/netbox.md`
- `docs/applications/70-tools/linkwarden.md`
- `docs/applications/60-services/gluetun.md`
- `docs/applications/60-services/mosquitto.md`
- `docs/applications/00-infra/cert-manager.md`
- `docs/applications/03-security/authentik.md`
- `docs/applications/40-network/external-dns.md`
- `docs/applications/60-services/firefly-iii.md`
- `docs/applications/40-network/netvisor.md`
- `docs/applications/04-databases/redis-shared.md`
- `docs/applications/60-services/vaultwarden.md`
- `docs/applications/60-services/firefly-iii-importer.md`
- `docs/applications/10-home/homeassistant.md`
- `docs/applications/00-infra/cert-manager-webhook-gandi.md`
- `docs/applications/02-monitoring/grafana.md`
- `docs/applications/70-tools/argocd-image-updater.md`
- `docs/applications/04-databases/postgresql-shared.md`
- `docs/applications/04-databases/mariadb-shared.md`
- `docs/applications/02-monitoring/alertmanager.md`

Some application docs already explicitly state that Infisical is retired; those
matches are intentional and need no rewrite.

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

Do not close #3546 until:

1. the remaining active application docs have been checked against current manifests;
2. any surviving retired-tool references are either corrected or explicitly historical;
3. a final post-merge drift search on current `main` is classified;
4. active documentation navigation points only to current guidance or clearly marked historical compatibility stubs.
