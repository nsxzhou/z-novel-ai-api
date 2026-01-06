-- 000010_create_project_creation_sessions.up.sql
-- 创建“对话创建项目”会话与轮次表

-- 创建会话：用于在 Project 未存在时，进行多轮对话收集信息并最终创建 Project
CREATE TABLE IF NOT EXISTS project_creation_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    stage VARCHAR(32) NOT NULL DEFAULT 'discover',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    draft JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_project_id UUID REFERENCES projects (id) ON DELETE SET NULL,
    created_project_session_id UUID REFERENCES conversation_sessions (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT project_creation_sessions_stage_check CHECK (stage IN ('discover', 'narrow', 'draft', 'confirm')),
    CONSTRAINT project_creation_sessions_status_check CHECK (status IN ('active', 'completed', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_project_creation_sessions_tenant ON project_creation_sessions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_project_creation_sessions_user ON project_creation_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_project_creation_sessions_created_at ON project_creation_sessions (created_at);

CREATE TRIGGER update_project_creation_sessions_updated_at
    BEFORE UPDATE ON project_creation_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 会话轮次：存储用户/助手消息（可追溯）
CREATE TABLE IF NOT EXISTS project_creation_turns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    session_id UUID NOT NULL REFERENCES project_creation_sessions (id) ON DELETE CASCADE,
    role VARCHAR(16) NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT project_creation_turns_role_check CHECK (role IN ('system', 'user', 'assistant'))
);

CREATE INDEX IF NOT EXISTS idx_project_creation_turns_session ON project_creation_turns (session_id);
CREATE INDEX IF NOT EXISTS idx_project_creation_turns_session_created ON project_creation_turns (session_id, created_at);

-- 启用 RLS
ALTER TABLE project_creation_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE project_creation_turns ENABLE ROW LEVEL SECURITY;

-- 会话表策略（直接按 tenant_id 隔离）
CREATE POLICY tenant_isolation_select ON project_creation_sessions FOR
SELECT USING (
        tenant_id = current_tenant_id ()
    );

CREATE POLICY tenant_isolation_insert ON project_creation_sessions FOR
INSERT
WITH
    CHECK (
        tenant_id = current_tenant_id ()
    );

CREATE POLICY tenant_isolation_update ON project_creation_sessions FOR
UPDATE USING (
    tenant_id = current_tenant_id ()
);

CREATE POLICY tenant_isolation_delete ON project_creation_sessions FOR DELETE USING (
    tenant_id = current_tenant_id ()
);

-- 会话轮次表策略（通过 session_id 关联 session.tenant_id）
CREATE POLICY tenant_isolation_select ON project_creation_turns FOR
SELECT USING (
        session_id IN (
            SELECT id
            FROM project_creation_sessions
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_insert ON project_creation_turns FOR
INSERT
WITH
    CHECK (
        session_id IN (
            SELECT id
            FROM project_creation_sessions
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_delete ON project_creation_turns FOR DELETE USING (
    session_id IN (
        SELECT id
        FROM project_creation_sessions
        WHERE
            tenant_id = current_tenant_id ()
    )
);

