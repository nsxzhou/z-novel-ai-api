-- 000011_create_llm_usage_events.up.sql
-- 创建 LLM 使用量事件表（按租户统计 Token，用于非 project 作用域的 LLM 调用）

CREATE TABLE IF NOT EXISTS llm_usage_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL,
    model VARCHAR(64) NOT NULL,
    tokens_prompt INT NOT NULL DEFAULT 0,
    tokens_completion INT NOT NULL DEFAULT 0,
    duration_ms INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_llm_usage_events_tenant ON llm_usage_events (tenant_id);
CREATE INDEX IF NOT EXISTS idx_llm_usage_events_tenant_created ON llm_usage_events (tenant_id, created_at);

-- 启用 RLS
ALTER TABLE llm_usage_events ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_select ON llm_usage_events FOR
SELECT USING (
        tenant_id = current_tenant_id ()
    );

CREATE POLICY tenant_isolation_insert ON llm_usage_events FOR
INSERT
WITH
    CHECK (
        tenant_id = current_tenant_id ()
    );

CREATE POLICY tenant_isolation_delete ON llm_usage_events FOR DELETE USING (
    tenant_id = current_tenant_id ()
);

