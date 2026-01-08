-- Active: 1767701467389@@127.0.0.1@5432@z_novel_ai
-- 000012_add_tenant_balance.up.sql
-- 为租户增加余额字段，并优化使用量记录表

-- 1. 为 tenants 增加 token_balance 字段
-- 默认给予 100万 Token 作为初始余额（根据之前 NewTenant 的逻辑）
ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS token_balance BIGINT NOT NULL DEFAULT 1000000;

-- 2. 为 llm_usage_events 增加 workflow 字段以支持分类统计
ALTER TABLE llm_usage_events
ADD COLUMN IF NOT EXISTS workflow VARCHAR(64);

-- 3. 添加索引
CREATE INDEX IF NOT EXISTS idx_llm_usage_events_workflow ON llm_usage_events (workflow);