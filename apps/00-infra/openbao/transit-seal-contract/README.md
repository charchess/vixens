# Infra Transit seal authority — source contract only

This directory is deliberately outside every Infra OpenBao Kustomization. It
contains no Kubernetes resource, endpoint, CA bundle, JWT, bearer credential,
role installation, key operation, or network rule. It must never be selected
or installed by this source contract.

`txo-fabric-seal.hcl.example` is a non-executable policy example for the
future, separately approved Infra role
`auth/kubernetes/role/txo-fabric-seal-bootstrap`. Its complete allowed surface
is exactly:

- `update transit/encrypt/txo-fabric-seal`
- `update transit/decrypt/txo-fabric-seal`

No other path, capability, mount, policy management, authentication management,
token management, KV access, listing, or administration is authorized. This
is Infra startup seal-unwrapping authority only; it is never a Fabric tenant or
runtime SecretStore/ExternalSecret authority and never a fallback.

The evidence gate is documented in
[`openbao-transit-seal-evidence.md`](../../../../docs/runbooks/00-infra/openbao-transit-seal-evidence.md).
