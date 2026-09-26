# TXO Fabric — tenant-neutral bootstrap

This directory defines the minimum TXO Fabric management-plane bootstrap.

The bootstrap is intentionally **tenant neutral**: after a fresh deployment the
platform is healthy but owns no customer cell, no customer namespace, no Hermes
runtime and no tenant Hindsight/Paperclip/Valkey workload.

## Boundary

The management plane exists to operate tenant cells. A tenant cell exists only
when a `TenantBundle` is declared later through the platform API/GitOps flow.

This first slice installs only:

- the `txo-fabric-system` management namespace;
- a default-deny network policy for future management workloads;
- the cluster-scoped `TenantBundle` API contract;
- the Argo CD application that keeps this bootstrap reconciled.

There is deliberately no `TenantBundle` object in `base` or in the `dev`
overlay.

## TenantBundle contract

`TenantBundle` represents the logical bundle owned by one customer entity. It
separates logical ownership from physical placement: a module may later use a
shared PostgreSQL cluster or another shared platform service while remaining
owned and authorized as part of one tenant bundle.

Optional modules are declared as data instead of being hard-wired into a
customer overlay. This is the extension point for components such as Hindsight,
file/object repositories, Paperclip, Valkey or execution runtimes.

`hAIrem` is not special-cased here. It will be created in a later change as the
first consumer of the same tenant-onboarding path used for any external client.

## Zero-tenant acceptance

After Argo CD has reconciled `apps/60-services/txo-fabric/overlays/dev`:

```console
kubectl get tenantbundles
No resources found
```

The following must also be true:

- no `tenant-*` namespace is created by this bootstrap;
- no Hermes runtime exists because of this bootstrap;
- no tenant database, memory bank, repository, Paperclip or Valkey instance is
  created implicitly;
- adding the first tenant is a separate declarative change.

## Next slice

The next POC step is to create the `hAIrem` tenant bundle, observe the tenant
provisioning path, then add agents and optional modules independently.
