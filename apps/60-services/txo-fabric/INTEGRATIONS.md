# TXO Fabric v0 integration authorization

Issue #3733 implements the first deliberately small slice of the #3687
integration boundary. It proves that an `AgentIdentity` receives integration
authorization from Fabric intent rather than from retained Hermes state.

## Durable customer-facing contract

Two cluster-scoped Fabric resources carry the v0 contract:

- `IntegrationConnection` identifies a tenant-owned logical integration,
  protocol, endpoint, authentication class and logical `credentialRef`;
- `IntegrationBinding` grants one `AgentIdentity` access to one connection
  with explicit operations and scopes.

Neither object contains a credential value, a Kubernetes Secret key, an OpenBao
path, a provider master credential or customer/vendor-specific fields.

A binding can be `Active` or `Revoked`. Removing or revoking it removes the
credential projection and integration egress from the generated Hermes runtime.
The private `/opt/data` PVC is not consulted when authorization is resolved and
is not modified during revocation/restore.

`AgentIdentity.status.runtime.integrations` exposes only non-secret diagnostics:
binding name, connection name, effective/revoked/denied phase, reason, revision,
operations and scopes.

## Temporary v0 credential adapter

The v0 runtime adapter intentionally uses the narrowest existing
platform-managed Secret mechanism instead of importing the v0.2 IAaaS vault.

For an effective binding, `IntegrationConnection.spec.credentialRef.name` is a
logical Fabric credential name. The v0 adapter currently resolves that name to a
Secret with the same name in the tenant namespace. The Secret must:

- contain key `credential`;
- carry `fabric.truxonline.io/integration-credential: "true"`;
- carry the matching `fabric.truxonline.io/tenant-name`;
- carry the matching `fabric.truxonline.io/integration-connection`.

The raw value is never copied into Git, Fabric CRDs/status, annotations, labels,
controller logs or the generated Deployment. Kubernetes projects the one
credential key read-only at:

`/run/txo/integrations/<connection>/credential`

The non-secret `TXO_INTEGRATIONS_JSON` environment variable tells the runtime
which binding/connection is effective, its endpoint/authentication type,
operations/scopes, credential path and revision.

This direct Secret projection is **temporary v0 plumbing**. It is not the durable
credential architecture and must not be generalized into a customer vault.

## v0.2 migration boundary

#3728 and #3729 replace the temporary adapter with the dedicated IAaaS OpenBao
substrate and TXO Credential Broker, including non-revealing `use != read`.
The customer-facing `IntegrationConnection` / `IntegrationBinding` contract
does not expose the v0 Secret backend, so the runtime can later satisfy the same
logical credential reference through the broker without changing tenant
manifests.

The v0 implementation does **not** provide:

- IAaaS OpenBao;
- `CredentialGrant`;
- brokered non-revealing use;
- human credential-management UX;
- browser identities/backends;
- vendor-specific CRM/mail logic.

## POC network boundary

An effective v0 integration binding adds temporary broad TCP egress on ports
80, 443 and 8080 to the Hermes pod. This exists only to exercise generic
HTTP integrations during the POC.

This rule is **not** a destination sandbox and an `IntegrationBinding` must not
be described as one while terminal/browser capabilities and broad POC egress are
available. Production direction remains policy-derived least-privilege egress
aligned with effective integration authorization and, where appropriate, broker
or gateway boundaries from #3687/#3729.

Revoking the last effective integration removes this extra rule through normal
AgentIdentity reconciliation.

## fabric-smoke canary

`tenants/fabric-smoke/integration-canary.yaml` provides a vendor-neutral HTTP
canary. It requires an exact Bearer token and logs only HTTP method/outcome, never
the Authorization header.

The dedicated test credential is delivered by External Secrets from:

`vixens/prod/apps/60-services/txo-fabric/fabric-smoke-integration-canary`
(property `bearer_token`)

That location is a disposable platform test credential in the existing Vixens
secret boundary. It is **not customer/user IAaaS secret storage** and must not be
used as precedent for #3728.

Only `fabric-smoke-probe` receives the canary binding.
`fabric-smoke-sales-probe` remains intentionally unbound.

## Physical acceptance after explicit human promotion

CI can prove controller semantics, generated CRDs/RBAC, Kustomize wiring and
secret-non-disclosure in rendered runtime objects. CI cannot prove the physical
cluster round-trip.

After an explicit human promotion, physical acceptance still needs to verify:

1. the dedicated `bearer_token` exists in the documented platform secret path;
2. External Secrets creates the tenant-local credential and canary becomes ready;
3. `fabric-smoke-probe` sees an Effective integration status and can call the
   canary using its projected credential;
4. `fabric-smoke-sales-probe` cannot use the canary through the Fabric runtime
   authorization path;
5. switching the binding to `Revoked` removes the projection and temporary
   integration egress after reconciliation;
6. the retained/private runtime PVC is neither required nor sufficient to regain
   the revoked authorization;
7. restoring the binding to `Active` re-establishes access through normal
   reconciliation without copying secret state;
8. controller/runtime/canary logs and status output contain no credential value.

Do not print the token during acceptance. Do not mutate or inspect protected
Hermes PVC contents to prove this contract.
