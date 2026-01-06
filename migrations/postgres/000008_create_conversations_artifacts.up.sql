-- 000008_create_conversations_artifacts.up.sql
-- 创建会话（长期对话）与构件版本化表

-- 会话：同一 Project 的长期对话容器（支持 task 切换）
CREATE TABLE IF NOT EXISTS conversation_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    current_task VARCHAR(32) NOT NULL DEFAULT 'novel_foundation',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conversation_sessions_tenant ON conversation_sessions (tenant_id);

CREATE INDEX IF NOT EXISTS idx_conversation_sessions_project ON conversation_sessions (project_id);

CREATE TRIGGER update_conversation_sessions_updated_at
    BEFORE UPDATE ON conversation_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 会话轮次：存储用户/助手消息（可追溯）
CREATE TABLE IF NOT EXISTS conversation_turns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    session_id UUID NOT NULL REFERENCES conversation_sessions (id) ON DELETE CASCADE,
    role VARCHAR(16) NOT NULL,
    task VARCHAR(32) NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT conversation_turns_role_check CHECK (role IN ('system', 'user', 'assistant'))
);

CREATE INDEX IF NOT EXISTS idx_conversation_turns_session ON conversation_turns (session_id);

CREATE INDEX IF NOT EXISTS idx_conversation_turns_session_created ON conversation_turns (session_id, created_at);

-- 构件：每个 Project 下按类型（worldview/characters/outline/novel_foundation）独立对象
CREATE TABLE IF NOT EXISTS project_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    type VARCHAR(32) NOT NULL,
    active_version_id UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (project_id, type)
);

CREATE INDEX IF NOT EXISTS idx_project_artifacts_tenant ON project_artifacts (tenant_id);

CREATE INDEX IF NOT EXISTS idx_project_artifacts_project ON project_artifacts (project_id);

CREATE TRIGGER update_project_artifacts_updated_at
    BEFORE UPDATE ON project_artifacts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 构件版本：每次生成写入新版本；回滚通过切 active_version_id 完成
CREATE TABLE IF NOT EXISTS artifact_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    artifact_id UUID NOT NULL REFERENCES project_artifacts (id) ON DELETE CASCADE,
    version_no INT NOT NULL,
    content JSONB NOT NULL,
    created_by UUID REFERENCES users (id),
    source_job_id UUID REFERENCES generation_jobs (id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (artifact_id, version_no)
);

CREATE INDEX IF NOT EXISTS idx_artifact_versions_artifact ON artifact_versions (artifact_id);

CREATE INDEX IF NOT EXISTS idx_artifact_versions_artifact_created ON artifact_versions (artifact_id, created_at);

ALTER TABLE project_artifacts
    ADD CONSTRAINT project_artifacts_active_version_fkey FOREIGN KEY (active_version_id) REFERENCES artifact_versions (id) ON DELETE SET NULL;

-- 启用 RLS
ALTER TABLE conversation_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE conversation_turns ENABLE ROW LEVEL SECURITY;
ALTER TABLE project_artifacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE artifact_versions ENABLE ROW LEVEL SECURITY;

-- 会话表策略（直接按 tenant_id 隔离）
CREATE POLICY tenant_isolation_select ON conversation_sessions FOR
SELECT USING (
        tenant_id = current_tenant_id ()
    );

CREATE POLICY tenant_isolation_insert ON conversation_sessions FOR
INSERT
WITH
    CHECK (
        tenant_id = current_tenant_id ()
    );

CREATE POLICY tenant_isolation_update ON conversation_sessions FOR
UPDATE USING (
    tenant_id = current_tenant_id ()
);

CREATE POLICY tenant_isolation_delete ON conversation_sessions FOR DELETE USING (
    tenant_id = current_tenant_id ()
);

-- 会话轮次表策略（通过 session_id 关联 session.tenant_id）
CREATE POLICY tenant_isolation_select ON conversation_turns FOR
SELECT USING (
        session_id IN (
            SELECT id
            FROM conversation_sessions
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_insert ON conversation_turns FOR
INSERT
WITH
    CHECK (
        session_id IN (
            SELECT id
            FROM conversation_sessions
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_delete ON conversation_turns FOR DELETE USING (
    session_id IN (
        SELECT id
        FROM conversation_sessions
        WHERE
            tenant_id = current_tenant_id ()
    )
);

-- 构件表策略（直接按 tenant_id 隔离）
CREATE POLICY tenant_isolation_select ON project_artifacts FOR
SELECT USING (
        tenant_id = current_tenant_id ()
    );

CREATE POLICY tenant_isolation_insert ON project_artifacts FOR
INSERT
WITH
    CHECK (
        tenant_id = current_tenant_id ()
    );

CREATE POLICY tenant_isolation_update ON project_artifacts FOR
UPDATE USING (
    tenant_id = current_tenant_id ()
);

CREATE POLICY tenant_isolation_delete ON project_artifacts FOR DELETE USING (
    tenant_id = current_tenant_id ()
);

-- 构件版本表策略（通过 artifact_id 关联 project_artifacts.tenant_id）
CREATE POLICY tenant_isolation_select ON artifact_versions FOR
SELECT USING (
        artifact_id IN (
            SELECT id
            FROM project_artifacts
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_insert ON artifact_versions FOR
INSERT
WITH
    CHECK (
        artifact_id IN (
            SELECT id
            FROM project_artifacts
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_delete ON artifact_versions FOR DELETE USING (
    artifact_id IN (
        SELECT id
        FROM project_artifacts
        WHERE
            tenant_id = current_tenant_id ()
    )
);
