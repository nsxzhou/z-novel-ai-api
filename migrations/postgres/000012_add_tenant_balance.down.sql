-- Active: 1767701467389@@127.0.0.1@5432@z_novel_ai
-- 000012_add_tenant_balance.down.sql

ALTER TABLE tenants DROP COLUMN IF EXISTS token_balance;

ALTER TABLE llm_usage_events DROP COLUMN IF EXISTS workflow;

DROP INDEX IF EXISTS idx_llm_usage_events_workflow;