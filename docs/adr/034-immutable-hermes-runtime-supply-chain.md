# ADR-034: Build Hermes extensions into an immutable TXO runtime image

**Date:** 2026-10-02  
**Status:** Accepted  
**Scope:** AIaaS  
**Related Project:** vixens roadmap  
**Related Issues:** #3710, #3696, #3685  
**Deciders:** Vixens maintainers  
**Tags:** txo-fabric, hermes, supply-chain, immutable-image, runtime, retrospective

> **Retrospective ADR:** this records the runtime supply-chain contract implemented and accepted under #3696 before this ADR was written.

## Context

TXO Fabric needs Hermes runtime extensions such as reviewed memory providers. Installing those extensions when an agent pod starts would make availability and behavior depend on mutable external package/catalog state, Internet access and unreviewed dependency resolution.

Fabric already relies on GitOps and immutable artifacts. The Hermes runtime therefore needs the same reproducibility and review boundary without coupling its release cadence to the Fabric operator image.

The first accepted implementation under #3696 used a reviewed upstream Hermes release and a pinned Hindsight provider/client dependency set. Those exact versions are historical implementation evidence; the durable decision is the immutable, source-pinned runtime supply chain described here.

## Decision

TXO Fabric uses a TXO-owned immutable Hermes runtime image for platform-required Hermes extensions.

The runtime supply-chain contract is:

- start from an explicit reviewed upstream Hermes image/version;
- bake platform-required providers/plugins and their compatible dependencies into the image at build time;
- pin extension source/dependencies to reviewed immutable revisions/versions rather than resolving floating latest state at pod startup;
- verify during image build/CI that Hermes remains executable and required extensions are discoverable/importable;
- publish source-derived immutable runtime image tags to the approved registry;
- pin `AgentRuntimeProfile` to an immutable runtime image through the normal GitOps PR flow;
- do not require `hermes plugins install`, Git clone, PyPI/package installation or equivalent network mutation during normal agent startup;
- keep the Hermes runtime image lifecycle independent from the TXO Fabric operator image lifecycle;
- do not automatically promote the runtime to production merely because an image build succeeded.

Runtime configuration is a separate concern from runtime composition. Baking a provider into the image does not by itself enable it for an agent. Provider selection, credentials, bank identity and tenant service wiring remain owned by their respective Fabric configuration contracts.

## Consequences

### Positive

- agent startup is reproducible and does not depend on GitHub/package-index availability;
- reviewed extension source and dependencies are part of the image provenance;
- runtime upgrades can be reviewed/pinned independently from operator releases;
- GitOps sees the exact runtime artifact selected for an `AgentRuntimeProfile`;
- startup cannot silently drift to a newer plugin/dependency version.

### Negative

- changing a required Hermes extension requires a new runtime image build and pin;
- the project owns image build/provenance maintenance in addition to controller releases;
- compatibility must be tested at build time and again through runtime acceptance.

## Alternatives considered

### Install plugins when each pod starts

Rejected because it introduces mutable network/package dependencies into agent startup and weakens reproducibility.

### Bundle Hermes into the Fabric operator image

Rejected because the controller and agent runtime have different responsibilities and release cadences.

### Track a floating upstream Hermes/plugin tag

Rejected because a previously reviewed Git commit could produce different runtime behavior after an upstream move.

## Historical implementation evidence

The implementation accepted by #3696 was derived from `nousresearch/hermes-agent:v2026.9.24` and baked the reviewed Hindsight provider pinned to commit `176f8c2de1369f569c489b831d143b78128b5535`, with compatible client dependencies pinned at build time. Future version changes belong in living runtime/build configuration; they do not require a new ADR unless the supply-chain architecture itself changes.

## References

- #3696 — immutable Hermes runtime implementation and acceptance
- #3685 — parent Hermes/Hindsight functional objective
- #3710 — accepted v0 platform-contract ADR harvesting
- `.github/workflows/build-txo-fabric-hermes.yaml` — living build/publish workflow
- `apps/60-services/txo-fabric/` — runtime build/profile configuration
