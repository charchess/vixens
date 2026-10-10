-- Fabric BFF-owned PostgreSQL schema. Apply via reviewed GitOps migration
-- before any BFF accepts an Authentik browser callback. This migration does
-- NOT create users, a database, credentials, networking, or a deployment.
-- Use a dedicated Fabric platform database + least-privilege runtime role.
-- The table holds only SHORT-LIVED OAuth state, not an identity/grant source.
--
-- Production prerequisite: encrypted PostgreSQL storage/backups, no SQL
-- parameter or row logging, private BFF-only DB egress and query grants.
-- Never grant this table to a tenant workload, Hermes, Authentik or browser.

CREATE TABLE IF NOT EXISTS txo_fabric_oidc_login_transactions (
    tenant_key       varchar(63) NOT NULL,
    state_digest     varchar(43) NOT NULL,
    issuer           varchar(1024) NOT NULL,
    client_id        varchar(1024) NOT NULL,
    redirect_uri     varchar(1024) NOT NULL,
    nonce            varchar(43) NOT NULL,
    pkce_verifier    varchar(43) NOT NULL,
    binding_digest   varchar(43) NOT NULL,
    issued_at_ms     bigint NOT NULL,
    expires_at_ms    bigint NOT NULL,

    CONSTRAINT txo_oidc_pk PRIMARY KEY (tenant_key, state_digest),
    CONSTRAINT txo_oidc_tenant_valid CHECK (
        tenant_key ~ '^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$'
    ),
    CONSTRAINT txo_oidc_state_valid CHECK (
        state_digest ~ '^[A-Za-z0-9_-]{43}$'
        AND nonce ~ '^[A-Za-z0-9_-]{43}$'
        AND pkce_verifier ~ '^[A-Za-z0-9_-]{43}$'
        AND binding_digest ~ '^[A-Za-z0-9_-]{43}$'
    ),
    CONSTRAINT txo_oidc_expiry_valid CHECK (
        issued_at_ms >= 0
        AND expires_at_ms = issued_at_ms + 300000
        AND expires_at_ms <= 8640000000000000
    )
);

-- Required by the platform-owned bounded maintenance query.
CREATE INDEX IF NOT EXISTS txo_oidc_expiry
    ON txo_fabric_oidc_login_transactions (expires_at_ms);

REVOKE ALL ON TABLE txo_fabric_oidc_login_transactions FROM PUBLIC;

-- Future GitOps must create a separately reviewed dedicated BFF runtime role
-- with ONLY SELECT (for DELETE RETURNING), INSERT and DELETE. It must not
-- have UPDATE, TRUNCATE, REFERENCES, CREATE, or schema migration privileges.
-- PostgreSQL's DELETE RETURNING requires SELECT on returned columns.
-- Do not hardcode role passwords or use the OpenFGA database credentials.
