# Agent instructions for Vixens

Vixens is a GitOps repository. Agents use the same repository rules as humans; there is no agent-specific workflow.

## Read first

1. `WORKFLOW.md`
2. `docs/architecture.md`
3. `docs/adr/000-index.md` when a decision is relevant
4. the affected manifests under `apps/` or `argocd/`

Those documents are authoritative. Do not invent a parallel workflow.

## Hard rules

- Desired Kubernetes state changes through Git, not direct workload mutation.
- Never push directly to `main`.
- Normal work uses a short-lived branch and a PR to `main`.
- Production promotion uses `promote-prod.yaml`; do not manually move `prod-stable`.
- Never move `prod-working` without explicit owner approval.
- GitHub Issues are the task tracker. Do not use Beads or an external task database.
- Do not add Terraform/Talos provisioning to this repository; that belongs in `terravixens`.
- Never put secret material in Git, including examples and diagnostic output.
- Do not rewrite historical ADR decisions or post-mortems. Create a new ADR for a new decision.

## Validation

Run `just validate` before requesting merge. CI is authoritative and must pass independently.

For diagnosis, read-only cluster commands are fine. If runtime state differs from Git, repair Git or trigger reconciliation; do not encode a permanent fix with `kubectl apply/edit/patch`.

## Documentation

Keep documentation compact and referential:

- workflow → `WORKFLOW.md`
- architecture → `docs/architecture.md`
- decisions → `docs/adr/000-index.md`
- how-to → `docs/guides/`
- per-application operations → `docs/applications/`
- incidents → `docs/post-mortems/`

Avoid copying the same operational rule into multiple files.
