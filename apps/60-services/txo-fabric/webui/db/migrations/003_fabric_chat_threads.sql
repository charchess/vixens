-- 003: Future-only Fabric BFF chat state, NOT activated by current prod bootstrap.
-- Apply AFTER 001/002 using the dedicated platform migrator in a separately
-- approved GitOps migration; never change an already applied migration.
-- This stores opaque IDs and immutable ownership, NO credentials, cookies,
-- headers, tool/SSE payload, upstream errors or arbitrary browser metadata.

CREATE TABLE IF NOT EXISTS txo_fabric_chat_threads (
  thread_id varchar(40) PRIMARY KEY,
  tenant_key varchar(63) NOT NULL,
  fabric_tenant_id varchar(64) NOT NULL,
  tenant_bundle_uid uuid NOT NULL,
  fabric_user_id varchar(35) NOT NULL,
  agent_key varchar(56) NOT NULL,
  agent_identity_uid uuid NOT NULL,
  agent_observed_generation bigint NOT NULL,
  remote_session_id varchar(128),
  created_at_ms bigint NOT NULL,
  updated_at_ms bigint NOT NULL,
  -- Token fenced inside the platform repository, NEVER sent to browser.
  lease_token varchar(32),
  lease_expires_at_ms bigint,
  lease_uncertain boolean NOT NULL DEFAULT false,
  CONSTRAINT chat_thread_id_valid CHECK (thread_id ~ '^[a-f0-9]{40}$'),
  CONSTRAINT chat_tenant_key_valid CHECK (tenant_key ~ '^[a-z0-9]([a-z0-9-]{0,54}[a-z0-9])?$'),
  CONSTRAINT chat_tenant_id_valid CHECK (fabric_tenant_id ~ '^TEN[0-9]{5,}$'),
  CONSTRAINT chat_user_id_valid CHECK (fabric_user_id ~ '^usr[a-f0-9]{32}$'),
  CONSTRAINT chat_agent_key_valid CHECK (agent_key ~ '^[a-z0-9]([a-z0-9-]{0,54}[a-z0-9])?$'),
  CONSTRAINT chat_agent_generation_valid CHECK (agent_observed_generation > 0),
  CONSTRAINT chat_session_valid CHECK (remote_session_id IS NULL OR
    remote_session_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
  CONSTRAINT chat_times_valid CHECK (created_at_ms >= 0
    AND updated_at_ms >= created_at_ms
    AND updated_at_ms <= 8640000000000000),
  CONSTRAINT chat_lease_valid CHECK (
    (lease_token IS NULL AND lease_expires_at_ms IS NULL AND lease_uncertain = false)
    OR (lease_token ~ '^[a-f0-9]{32}$' AND lease_expires_at_ms IS NOT NULL
      AND lease_expires_at_ms > 0 AND lease_expires_at_ms <= 8640000000000000)
  )
);

-- A server-created Hermes session cannot be adopted by a second thread of
-- the same immutable AgentIdentity, including in another BFF replica.
CREATE UNIQUE INDEX IF NOT EXISTS txo_chat_remote_once
  ON txo_fabric_chat_threads (agent_identity_uid, remote_session_id)
  WHERE remote_session_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS txo_chat_owner_list
  ON txo_fabric_chat_threads (fabric_tenant_id, tenant_bundle_uid,
    fabric_user_id, created_at_ms, thread_id);

CREATE TABLE IF NOT EXISTS txo_fabric_chat_turns (
  thread_id varchar(40) NOT NULL REFERENCES txo_fabric_chat_threads(thread_id)
    ON DELETE RESTRICT,
  turn_id varchar(32) NOT NULL,
  state varchar(12) NOT NULL,
  lease_token varchar(32) NOT NULL,
  started_at_ms bigint NOT NULL,
  finished_at_ms bigint,
  result_code varchar(16),
  PRIMARY KEY (thread_id, turn_id),
  CONSTRAINT chat_turn_id_valid CHECK (turn_id ~ '^[a-f0-9]{32}$'),
  CONSTRAINT chat_turn_lease_valid CHECK (lease_token ~ '^[a-f0-9]{32}$'),
  CONSTRAINT chat_turn_state_valid CHECK
    (state IN ('claimed', 'completed', 'failed', 'cancelled')),
  CONSTRAINT chat_turn_result_valid CHECK (
    (state = 'claimed' AND finished_at_ms IS NULL AND result_code IS NULL)
    OR (state = 'completed' AND result_code = 'completed'
      AND finished_at_ms IS NOT NULL AND finished_at_ms >= started_at_ms)
    OR (state = 'failed' AND result_code = 'failed'
      AND finished_at_ms IS NOT NULL AND finished_at_ms >= started_at_ms)
    OR (state = 'cancelled' AND result_code = 'cancelled'
      AND finished_at_ms IS NOT NULL AND finished_at_ms >= started_at_ms)
  )
);

-- DDL owner and dedicated runtime role are separate. The eventual approved
-- bootstrap must grant only SELECT/INSERT/UPDATE on THESE two tables.
-- Never grant DELETE/TRUNCATE/DDL, broad schema permissions or tenant access.
REVOKE ALL ON TABLE txo_fabric_chat_threads FROM PUBLIC;
REVOKE ALL ON TABLE txo_fabric_chat_turns FROM PUBLIC;
