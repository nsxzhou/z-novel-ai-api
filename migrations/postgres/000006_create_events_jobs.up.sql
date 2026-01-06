-- 000006_create_events_jobs.up.sql
-- 创建事件和任务表

-- 事件表
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES chapters (id) ON DELETE SET NULL,
    story_time_start BIGINT NOT NULL,
    story_time_end BIGINT,
    event_type VARCHAR(64),
    summary TEXT NOT NULL,
    description TEXT,
    involved_entities UUID [],
    location_id UUID REFERENCES entities (id),
    importance VARCHAR(16) DEFAULT 'normal',
    tags TEXT [],
    vector_id VARCHAR(64),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_project ON events (project_id);

CREATE INDEX IF NOT EXISTS idx_events_chapter ON events (chapter_id);

CREATE INDEX IF NOT EXISTS idx_events_story_time ON events (project_id, story_time_start);

CREATE INDEX IF NOT EXISTS idx_events_type ON events (project_id, event_type);

CREATE INDEX IF NOT EXISTS idx_events_involved ON events USING GIN (involved_entities);

CREATE INDEX IF NOT EXISTS idx_events_tags ON events USING GIN (tags);

-- 生成任务表
CREATE TABLE IF NOT EXISTS generation_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    tenant_id UUID NOT NULL REFERENCES tenants (id),
    project_id UUID NOT NULL REFERENCES projects (id),
    chapter_id UUID REFERENCES chapters (id),
    job_type VARCHAR(32) NOT NULL,
    status VARCHAR(32) DEFAULT 'pending',
    priority INT DEFAULT 5,
    input_params JSONB NOT NULL,
    output_result JSONB,
    error_message TEXT,
    llm_provider VARCHAR(32),
    llm_model VARCHAR(64),
    tokens_prompt INT,
    tokens_completion INT,
    duration_ms INT,
    retry_count INT DEFAULT 0,
    progress INT DEFAULT 0,
    idempotency_key VARCHAR(128) UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE TRIGGER update_generation_jobs_updated_at
    BEFORE UPDATE ON generation_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_jobs_tenant ON generation_jobs (tenant_id);

CREATE INDEX IF NOT EXISTS idx_jobs_project ON generation_jobs (project_id);

CREATE INDEX IF NOT EXISTS idx_jobs_status ON generation_jobs (status);

CREATE INDEX IF NOT EXISTS idx_jobs_created ON generation_jobs (created_at);

CREATE INDEX IF NOT EXISTS idx_jobs_idempotency ON generation_jobs (idempotency_key);

-- 审计日志表（分区表）
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    user_id UUID,
    action VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource_id UUID,
    request_id VARCHAR(64),
    trace_id VARCHAR(64),
    ip_address INET,
    user_agent TEXT,
    changes JSONB,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
)
PARTITION BY
    RANGE (created_at);

-- 创建初始分区（2026年1-6月）
CREATE TABLE IF NOT EXISTS audit_logs_2026_01 PARTITION OF audit_logs FOR
VALUES
FROM ('2026-01-01') TO ('2026-02-01');

CREATE TABLE IF NOT EXISTS audit_logs_2026_02 PARTITION OF audit_logs FOR
VALUES
FROM ('2026-02-01') TO ('2026-03-01');

CREATE TABLE IF NOT EXISTS audit_logs_2026_03 PARTITION OF audit_logs FOR
VALUES
FROM ('2026-03-01') TO ('2026-04-01');

CREATE TABLE IF NOT EXISTS audit_logs_2026_04 PARTITION OF audit_logs FOR
VALUES
FROM ('2026-04-01') TO ('2026-05-01');

CREATE TABLE IF NOT EXISTS audit_logs_2026_05 PARTITION OF audit_logs FOR
VALUES
FROM ('2026-05-01') TO ('2026-06-01');

CREATE TABLE IF NOT EXISTS audit_logs_2026_06 PARTITION OF audit_logs FOR
VALUES
FROM ('2026-06-01') TO ('2026-07-01');

CREATE INDEX IF NOT EXISTS idx_audit_tenant ON audit_logs (tenant_id, created_at);

CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs (user_id, created_at);

CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_logs (resource_type, resource_id);

CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs (action, created_at);