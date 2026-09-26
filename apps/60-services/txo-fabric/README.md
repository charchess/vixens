# TXO Fabric — tenant-neutral management plane

This directory defines the tenant-neutral TXO Fabric management-plane bootstrap
and the first disposable provisioning POC.

A fresh deployment remains healthy with **zero tenants**: no customer namespace,
Hermes runtime, Hindsight bank, Paperclip or Valkey workload is created until a
`TenantBundle` or `AgentIdentity` is declared.

## API contracts

### TenantBundle

`TenantBundle` represents the logical bundle owned by one customer entity. It
separates logical ownership from physical placement: a module may later use a
shared PostgreSQL cluster or another shared platform service while remaining
owned and authorized as part of one tenant bundle.

Optional modules are declared as data instead of being hard-wired into a
customer overlay. This is the extension point for Hindsight, object storage,
Paperclip, Valkey and other capabilities.

### AgentIdentity

`AgentIdentity` is deliberately separate from `TenantBundle`: creating a tenant
does not implicitly create agents. Each identity names its tenant and its logical
Hindsight bank, while the runtime implementation remains controlled by TXO
Fabric.

## Provisioning POC

For the first end-to-end test, Kyverno acts as a small declarative reconciler:

- `TenantBundle` generates a `tenant-<slug>` namespace;
- every generated tenant namespace receives a default-deny NetworkPolicy;
- `AgentIdentity` generates an isolated Hermes PVC, Deployment and restricted
  egress policy inside its tenant namespace;
- generated resources are synchronized with their trigger, so deleting an
  `AgentIdentity` removes that agent runtime and deleting a `TenantBundle`
  removes its generated namespace;
- generated resources are orphaned if the provisioning policy itself is removed,
  preventing a policy refactor from accidentally deleting tenant cells.

This is intentionally **not** the final TXO Fabric controller. Kyverno does not
make `status` authoritative and does not yet orchestrate database/module
finalizers. A dedicated controller can replace this POC behind the same CRDs once
those lifecycle semantics are required.

## Security and credential boundary

New Hermes profiles are blank disposable profiles. They do **not** clone the
legacy hAIrem profile, provider API keys, OAuth state, messaging channels, skills
or persona data.

The generated runtime carries only TXO metadata and a Hindsight endpoint/bank
hook. `TXO_LLM_AUTH_MODE=unconfigured` is explicit: model access is not considered
ready until the planned scoped LLM credential broker exists. Likewise, the
Hindsight environment hook reserves the intended bank identity but is not a
replacement for the future Memory Gateway contract.

Tenant namespaces default-deny ingress and egress. The POC Hermes policy permits
only cluster DNS and the shared Hindsight API on TCP/8888; it does not grant
arbitrary Internet egress.

## Zero-tenant acceptance

After Argo CD has reconciled the base without any tenant declarations:

```console
kubectl get tenantbundles
No resources found

kubectl get agentidentities
No resources found
```

The following must also be true:

- no `tenant-*` namespace exists because of TXO Fabric;
- no Hermes runtime exists because of TXO Fabric;
- no tenant database, memory bank, repository, Paperclip or Valkey instance is
  created implicitly.

`hAIrem` receives no special treatment. Test and production tenant cells must use
the same API contracts as future external customers.
