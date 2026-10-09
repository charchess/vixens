# Central private OpenFGA runtime — staged for promotion

**Status: activation reviewed in PR #3994; not live until the exact version is manually promoted.**
The prod overlay includes this component in the activation candidate. Parent: #3990 / ADR-039.

## Topology

- Single `txo-fabric-system/txo-fabric-openfga` ClusterIP service (HTTP 8080 only), **no Ingress or playground**, no per-tenant server.
- Shared CNPG cluster `databases/postgresql-shared`, dedicated `txo_fabric_openfga` database and `DatabaseRole`, retain-on-removal.
- Migration Sync hook first, OpenFGA `v1.22.0` serving workload second. Explicit datastore PostgreSQL, fail closed if DB missing; no in-memory fallback.
- NetworkPolicy permits only the trusted Fabric operator pod to reach OpenFGA HTTP; all other ingress denied. Egress limited to CNPG Postgres and cluster DNS. A future Fabric BFF is **not** added to the allowlist until its authenticated identity and grants have been reviewed.
- Only Fabric backends may use the OpenFGA credential. Standard preshared key auth is NOT a store-level tenant security mechanism.

## Required one-time operator bootstrap BEFORE enabling the overlay

Store in **OpenBao**, using the existing ClusterSecretStore `openbao` and secrets flow (not Git):

`vixens/prod/apps/60-services/txo-fabric/openfga`

with fields:
- `username`: exactly `txo_fabric_openfga`
- `password`: random high-entropy PostgreSQL password, **URI-safe handling tested** (stored as opaque Secret, never printed)
- `presharedKeys`: random high-entropy OpenFGA service API key (never supplied to browsers, agents, or tenants)

The two ExternalSecrets create:
- `databases/txo-openfga-postgresql`: `username`, `password` for CNPG `DatabaseRole.passwordSecret`;
- `txo-fabric-system/txo-openfga-runtime`: `username`, `password`, `presharedKeys` for migration + server.

### Bootstrap without an OpenBao CLI

The OpenBao `bao` executable is **not installed by default** on `grenat`. Use the authenticated OpenBao **web UI** instead (the configured cluster-side OpenBao endpoint is `http://nas.truxonline.com:8200`; use a trusted network and an HTTPS endpoint if one is available):

1. Open the OpenBao UI, authenticate as an operator allowed to **create** a KV-v2 secret.
2. Select the `kv` secrets engine and create the path `vixens/prod/apps/60-services/txo-fabric/openfga` (the `kv` mount is separate from the path).
3. On the trusted administrator machine, generate two independent keys (do not paste their values to chat or GitHub):

   ```bash
   python3 - <<'PY'
   import secrets
   print("password =", secrets.token_hex(32))
   print("presharedKeys =", secrets.token_hex(32))
   PY
   ```

4. Enter all **three** OpenBao fields, exactly: `username = txo_fabric_openfga`, `password = <first random value>`, `presharedKeys = <second random value>`. Save once.
5. Verify only that the **secret path and field names** exist; do not copy their values into diagnostics.

The ExternalSecrets still require an authorized OpenBao/ESO token and the `ClusterSecretStore/openbao` to be ready. If the OpenBao UI cannot be reached, install the official `bao` CLI following https://openbao.org/docs/install/ and log in with an appropriately scoped identity. **Do not use the ESO Kubernetes token as an administrator credential.**

**Do not paste secret values in an issue, PR, shell history, logs or this document.**
Make sure the CNPG database and role converge before expecting the migration Job to succeed. Database backup/recovery is provided by the shared CNPG infrastructure, but a restore must prove actual OpenFGA relation data recovery and rehydrate missing store/model mapping.

## Mandatory physical acceptance after activation

1. Check both ExternalSecrets `Ready` and Kubernetes Secrets **present** (never print values).
2. Check CNPG `DatabaseRole` and `Database` are reconciled and PostgreSQL login works using scoped credentials (avoid exposing the password).
3. Check migration Job completed with the **exact** image version and no fallback to `memory`; wait for OpenFGA Deployment `Available`.
4. Check ClusterIP and NetworkPolicy: approved Fabric caller can reach authenticated `/stores`/Check; tenant agent/BFF without grant and public ingress **cannot**.
5. Provision stores/model versions and synthetic tuples via the **trusted Fabric authorization administration backend**, not an unaudited ad-hoc human `fga store import` used as ongoing state.
6. Verify tenant A/B isolation, inherited/group grants, revocation, restart, backup restore and fail-closed OpenFGA/database outage behavior.
7. Only then attach Fabric WebUI #3981 / provider UI #3979; do not remove existing protections while adapters are absent.

## Activation and manual promotion

Activation PR [#3994](https://github.com/charchess/vixens/pull/3994) appends `- ../../openfga/runtime` to the **production Fabric overlay**. A merged source/staging PR alone does **not** deploy OpenFGA; production ArgoCD follows `prod-stable`. The OpenBao record and dependency readiness must be verified *before* requesting promotion.

The activation PR's merge creates its immutable `dev-vYYYY.MM.<PR>` via the existing auto-tag workflow. After CI and checks, tell the owner the exact identifier and recommend the copy/paste command:

```sh
gh workflow run promote-prod.yaml -f version=vYYYY.MM.<PR>
```

Do not actually promote without explicit authorization for that candidate. Do not change `prod-working` until the owner has physically confirmed the release as known-good.

## Versioning

Runtime image: `docker.io/openfga/openfga@sha256:9cf9a20af32a40434cd3a788c429879439ee9ea9abf65c9c200ee17cd43f769e`, resolved from official upstream `v1.22.0` via GitHub CI. The exact same digest is used by the migration job and Deployment and verified by the activation workflow. Registry digest validation is **not** a substitute for physical CNPG migration/login acceptance and OpenBao bootstrapping.
