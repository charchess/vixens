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

For hAIrem Client 0, Authentik owns the OIDC application
`txo-fabric-hairem`. The registered redirect URI is regex-bounded to the
hAIrem Fabric agent hostname shape and `/auth/callback`.

Application access is bound to the Authentik group
`txo-fabric-hairem-client0`. The group/policy is platform GitOps; user
membership is enterprise IAM data managed in Authentik and is deliberately not
copied into AgentIdentity or TenantBundle.

Human identity remains an Authentik identity. AgentIdentity remains the
machine/agent identity. Fabric does not become the enterprise user directory.

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
