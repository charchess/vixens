-- Apply after 001_oidc_login_transactions.sql through a REVIEWED
-- Fabric-platform GitOps migrator, not from a public request.
-- This is BFF-controlled SIGN-IN state, not a customer credential vault.
-- Runtime role needs only SELECT/INSERT/DELETE, not schema owner rights.
CREATE TABLE IF NOT EXISTS txo_fabric_webui_sessions (
    session_digest varchar(43) PRIMARY KEY,
    issuer varchar(1024) NOT NULL,
    oidc_subject varchar(255) NOT NULL,
    authentik_uuid varchar(36) NOT NULL,
    issued_at_ms bigint NOT NULL,
    expires_at_ms bigint NOT NULL,
    CONSTRAINT webui_session_hash_valid
      CHECK (session_digest ~ '^[A-Za-z0-9_-]{43}$'),
    CONSTRAINT webui_session_issuer_valid
      CHECK (issuer ~ '^https://[a-z0-9.-]+/application/o/txo-fabric-webui/$'),
    CONSTRAINT webui_session_uuid_valid
      CHECK (authentik_uuid ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT webui_session_ttl_valid
      CHECK (issued_at_ms >= 0 AND
             expires_at_ms = issued_at_ms + 28800000 AND
             expires_at_ms <= 8640000000000000)
);
CREATE INDEX IF NOT EXISTS webui_session_expiry
  ON txo_fabric_webui_sessions (expires_at_ms);
REVOKE ALL ON TABLE txo_fabric_webui_sessions FROM PUBLIC;
-- No tenant/fabricUserId/group ID, password, IdP access token or raw cookie.
-- A session identity is not proof of a tenant grant; check IAM/FGA LIVE.
