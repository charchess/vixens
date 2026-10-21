-- Additive compatibility migration for installations that already applied
-- 001 before Fabric-origin tenant proof binding existed. It records only a
-- SHA-256/base64url digest of a server-issued global BFF session binding.
-- It stores no bearer cookie, OIDC token, subject, UUID or authorization grant.

ALTER TABLE txo_fabric_oidc_login_transactions
    ADD COLUMN IF NOT EXISTS global_session_digest varchar(43);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'txo_oidc_global_session_digest_valid'
          AND conrelid = 'txo_fabric_oidc_login_transactions'::regclass
    ) THEN
        ALTER TABLE txo_fabric_oidc_login_transactions
            ADD CONSTRAINT txo_oidc_global_session_digest_valid CHECK (
                global_session_digest IS NULL OR global_session_digest ~ '^[A-Za-z0-9_-]{43}$'
            );
    END IF;
END $$;
