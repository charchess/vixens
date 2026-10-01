# ADR-032: TXO Fabric is tenant-neutral and valid with zero clients

**Date:** 2026-10-02  
**Status:** Accepted  
**Scope:** AIaaS  
**Related Project:** vixens roadmap  
**Related Issues:** #3709, #3279, #3672, #3675  
**Deciders:** Vixens maintainers  
**Tags:** txo-fabric, multitenancy, tenantbundle, product, retrospective

> **Retrospective ADR:** this records the ownership model already established by the Fabric bootstrap and tenant APIs before this ADR was written.

## Context

TXO Fabric is intended to host hAIrem/Client 0 and future customer organizations on the same product architecture. During early bootstrap work it would have been easy to encode the first tenant, its modules or its operational preferences directly into the management-plane baseline.

That would make the first customer a hidden platform dependency and force later customers to inherit hAIrem-specific resources or controller behavior. The implemented direction instead separates the tenant-neutral platform from tenant declarations.

## Decision

TXO Fabric is a tenant-neutral product platform and is a valid deployment with zero customer/client bundles.

The durable ownership model is:

- the Fabric management plane and generic platform contracts exist independently of any particular tenant;
- `TenantBundle` is the generic declaration of a tenant/customer product instance;
- hAIrem/Client 0 consumes the same generic tenant contracts as future customers;
- Indiba, hAIrem or any other customer name must not appear as a behavioral special case in generic controller logic;
- optional tenant capabilities and modules are expressed through generic product APIs/profiles rather than by changing the platform baseline for one tenant;
- deleting or omitting a tenant must not require removing platform components that are part of the tenant-neutral control plane;
- shared infrastructure may be provided by Core, but customer-specific workloads remain owned by their tenant/product declaration.

A feature may first be exercised by Client 0, but its controller/API contract must be stated generically before it becomes part of Fabric.

## Consequences

### Positive

- Client 0 dogfooding does not turn hAIrem into a privileged architecture path;
- new tenants can be added without cloning or forking controller logic;
- optional modules can evolve independently from the minimum Fabric platform;
- the zero-client state remains useful for bootstrap, recovery and platform validation.

### Negative

- first-customer conveniences sometimes require an additional generic contract instead of a one-off manifest;
- product/module boundaries must be made explicit rather than inferred from deployment order.

## Alternatives considered

### Bake Client 0 into the platform baseline

Rejected because customer-specific resources would become accidental prerequisites for every Fabric installation.

### Maintain customer-specific controller branches

Rejected because it would fragment behavior, testing and upgrade paths across tenants.

## References

- #3709 — retrospective Fabric ADR backfill
- #3279 — tenant-neutral Fabric bootstrap contract
- #3672 — optional Paperclip tenant module work
- #3675 — Client 0 readiness tracking history
- `apps/60-services/txo-fabric/` — current Fabric implementation
