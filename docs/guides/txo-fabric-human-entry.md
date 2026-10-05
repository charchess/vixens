# TXO Fabric human entry boundary (v0)

Issue #3689 defines the first real human-to-AgentIdentity interaction path.

## V0 transport

The v0 transport is the upstream Hermes web dashboard already present in the
pinned Hermes runtime. TXO Fabric does not add a second chat protocol.

An AgentIdentity must explicitly opt in:

```yaml
spec:
  humanAccess:
    enabled: true
```

The owning TenantBundle provides the transport and IAM policy:

```yaml
spec:
  humanAccess:
    web:
      domainSuffix: truxonline.com
      ingressClassName: traefik
      tlsClusterIssuer: letsencrypt-prod
      publicDNS: true
      dnsTarget: truxonline.com
      iamGroups:
        - client0
      oidc:
        issuer: https://authentik.truxonline.com/application/o/txo-fabric-hairem/
        clientId: txo-fabric-hairem
        scopes: openid profile email
```

AgentIdentity does not carry OIDC endpoints, client IDs, ingress classes or
public hostnames. Future channels belong beside `humanAccess.web` in the
tenant-owned transport policy rather than becoming channel-specific fields on
AgentIdentity.

## Stable routing

For the first web transport, Fabric derives:

`https://<agentKey>-<tenantName>.<domainSuffix>`

The Service and Ingress select the stable AgentIdentity labels, never Pod names
or Pod IPs. A pod replacement therefore keeps the same human-facing URL.

The operator records the resolved URL in
`AgentIdentity.status.runtime.humanEndpoint`.

No Service or Ingress is created for an AgentIdentity without
`humanAccess.enabled: true`.

## Authentication

Hermes binds its dashboard on port 9119 only for opted-in agents and uses its
bundled self-hosted OIDC provider.

Fabric injects only non-secret public OIDC configuration:

- issuer;
- public PKCE client ID;
- scopes;
- public URL.

No OIDC client secret is used or stored in the AgentIdentity/TenantBundle CRDs.

TenantBundle declares the structural IAM group keys allowed to enter human
surfaces through `humanAccess.web.iamGroups`. TXO Fabric derives Authentik
groups as:

`txo-fabric-<tenantName>-<iamGroup>`

and publishes one aggregate Authentik blueprint for all active tenants. The
blueprint contains the tenant structural groups, public PKCE OIDC
provider/application and policy bindings. This removes the previous requirement
to hand-author one Authentik blueprint per tenant.

For hAIrem Client 0, `iamGroups: [client0]` therefore resolves to
`txo-fabric-hairem-client0`. For Indiba, `iamGroups: [sales]` resolves to
`txo-fabric-indiba-sales`.

Fabric owns these structural IAM objects, but **not their human membership**.
Users, passwords, MFA, invitations and user/group membership remain live
enterprise IAM data managed in Authentik.

Human identity remains an Authentik identity. AgentIdentity remains the
machine/agent identity. Fabric does not become the enterprise user directory.

## Tenant admin dashboards

The tenant human-access policy also governs administrative WebUI surfaces owned
by tenant capabilities. A Hindsight-enabled tenant explicitly opts in with
`memory.hindsight.humanAccess: true`; when `humanAccess.web` is also present,
Fabric reconciles a stable memory administration endpoint:

`https://hindsight-<tenantName>.<domainSuffix>`

For hAIrem Client 0 this resolves to:

`https://hindsight-hairem.truxonline.com`

The Hindsight Control Plane is a separate server-side component. It receives the
tenant Hindsight API key from the tenant runtime Secret and calls the private
tenant Hindsight API directly. The browser never receives that API key, and
neither the Hindsight API nor PostgreSQL is exposed publicly.

The public Control Plane route is protected by the platform Authentik
ForwardAuth boundary. When Hindsight human access is enabled, the same
Fabric-generated aggregate Authentik blueprint adds the tenant proxy
provider/application and binds it to the TenantBundle `iamGroups`. Fabric also
aggregates every Fabric-owned Hindsight proxy provider into the Embedded Outpost
declaration so one tenant cannot replace another tenant's provider membership.

The Authentik outpost callback path `/outpost.goauthentik.io` is routed to the
embedded Authentik outpost without recursively applying ForwardAuth.

Fabric reports the private memory endpoint in
`TenantBundle.status.memory.hindsight.endpoint` and the stable human-facing
dashboard in `TenantBundle.status.memory.hindsight.humanEndpoint`.

## Network boundary

Tenant namespaces remain default-deny.

For an opted-in agent Fabric adds:

- ingress to dashboard port 9119 only from the `traefik` namespace;
- a Cilium FQDN egress policy allowing DNS plus TCP/443 only to the exact OIDC
  issuer hostname (for hAIrem: `authentik.truxonline.com`);
- split-DNS publication through the existing external-dns controllers. Internal
  UniFi DNS follows the Ingress host; public Gandi publication is an explicit
  tenant policy (`publicDNS`) and may request a CNAME target (`dnsTarget`).

The ordinary Kubernetes egress policy is not widened for human access. This is
deliberate: enabling a UI must not grant arbitrary HTTPS access or bypass the
IntegrationBinding authorization/egress contract proven by #3733.

Disabling human access removes the Service, Ingress and ingress allow-policy.
Invalid/missing tenant human policy also removes those external resources
fail-closed.

## Human file exchange

The dashboard Files surface is not pointed at the agent's private
`/opt/data`.

Human access requires:

- `AgentIdentity.spec.access.userRef`;
- a matching tenant workspace user scope;
- that user scope to enable `collaborative: true`.

Fabric sets `HERMES_DASHBOARD_FILES_ROOT` to:

`/workspace/shared/users/<userRef>/collaborative`

This makes the human-visible Files tab use the same authorized workspace model
as #3662. Private Hermes state, credentials and agent-local skills stay outside
that human file surface.

The current Client 0 POC uses workspace user `client0` for both Tesla and
Tina, proving that one human workspace can be intentionally used with multiple
agents.

## Client 0 acceptance population

- Tesla: human access enabled, `client0` workspace.
- Tina: human access enabled, `client0` workspace.
- Tiffa: human access disabled; no Service/Ingress should exist.

Physical acceptance must prove:

1. unauthenticated access is rejected by Hermes OIDC;
2. the same authenticated Client 0 user can use Tesla then Tina;
3. a Tesla message stays in Tesla's runtime/session and a Tina message stays
   in Tina's;
4. Tiffa has no public human endpoint;
5. a file uploaded through the dashboard user workspace is visible to the
   intended agent, and an agent-produced file is visible through that same
   workspace;
6. deleting/replacing an Hermes pod does not change the public URL;
7. normal usage requires no kubectl, Pod IP or Kubernetes object name.

## Future channels

The v0 dashboard is one transport, not the final UX.

Future mail/chat/social/voice/mobile channels should resolve an authenticated
human + stable AgentIdentity through a Fabric-owned authorization/routing
contract. They must not reintroduce the legacy implicit peer mesh, Pod
addressing or per-channel credentials inside AgentIdentity.
