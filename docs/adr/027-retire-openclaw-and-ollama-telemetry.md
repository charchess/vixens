# ADR-027: Retire OpenClaw and Ollama Telemetry

**Date:** 2026-09-12
**Status:** Accepted
**Deciders:** Vixens maintainers
**Tags:** gitops, argocd, observability, openrouter

---

## Context

OpenClaw is deprecated. Its Argo CD Applications still reconcile a deployment,
service, ingress, network policies, secret materialization, and production
backup configuration in the `services` namespace. The applications use Argo
CD resource finalizers and automated pruning, so removing their parent desired
state causes Argo CD to delete the child workload only after the Git change is
merged into the watched revision.

The hAIrem dashboard also made unauthenticated, direct runtime probes to two
Ollama endpoints. Those probes are telemetry only: the repository has no
remaining active Ollama provider dependency outside OpenClaw. Hindsight is now
configured to use OpenRouter for LLM and embeddings requests, and exposes its
own request/error telemetry.

## Decision

- Remove both OpenClaw Argo CD Application manifests, their entries in the dev
  and prod app-of-apps Kustomizations, and the complete OpenClaw GitOps tree.
- Do not delete or mutate an OpenBao secret directly. Removing the desired
  workload stops its references; secret lifecycle is a later, separately
  approved phase after retention/backup review.
- Replace hAIrem's direct Ollama endpoint probes with a read-only
  OpenRouter/Hindsight component derived from Hindsight health, operation, and
  LLM request telemetry. The dashboard does not contact OpenRouter directly
  and does not contain provider credentials.
- Retire the OpenClaw application runbook and record the retirement in the
  status dashboard. Historical incident reports remain as audit evidence.

## Consequences

### Positives

- Merging to `main` lets the dev app-of-apps remove the OpenClaw Application;
  its finalizer then prunes the retired dev workload. Promoting the same
  revision to `prod-stable` does the equivalent in production.
- The dashboard no longer creates network traffic to Ollama or reports model
  residency that is unrelated to the current Hindsight provider path.
- The active telemetry aligns with the supported OpenRouter/Hindsight route and
  exposes request failures and queued/stuck operations.

### Constraints and migration gates

- This change does not shut down an Ollama server, delete OpenClaw data,
  delete S3 backups, or remove an OpenBao secret. Those actions need an
  explicit later approval and a retention/restore decision.
- The production removal is not live until the reviewed Git change is merged,
  `prod-stable` is deliberately promoted, and Argo CD reports the OpenClaw
  Application deleted with no remaining managed resources.
- Before any later OpenBao-secret or backup cleanup, classify each remaining
  reference as an execution dependency, runtime-secret dependency, or passive
  telemetry dependency, and verify a restorable data-retention path.

## Validation

Before merge, build both app-of-apps overlays and verify that no active
Kustomization or Argo CD Application references `openclaw`. Build and typecheck
the hAIrem dashboard. After the approved merge and Argo reconciliation, use
read-only Argo CD/Kubernetes inspection to verify that the Application and its
managed resources are gone; do not use imperative deletion.

## References

- [Hindsight runtime configuration](../applications/60-services/hindsight.md)
- [Application status dashboard](../STATUS.md)
- [ADR-017: Pure Trunk-Based Development](017-pure-trunk-based-single-branch.md)
