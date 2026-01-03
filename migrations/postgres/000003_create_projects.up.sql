-- 000003_create_projects.up.sql
-- 创建项目表

CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    owner_id UUID REFERENCES users (id),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    genre VARCHAR(64),
    target_word_count INT,
    current_word_count INT DEFAULT 0,
    settings JSONB DEFAULT '{}',
    world_settings JSONB DEFAULT '{
        "time_system": "linear",
        "calendar": "custom",
        "locations": []
    }',
    status VARCHAR(32) DEFAULT 'draft',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_projects_tenant ON projects (tenant_id);

CREATE INDEX IF NOT EXISTS idx_projects_owner ON projects (owner_id);

CREATE INDEX IF NOT EXISTS idx_projects_status ON projects (status);

CREATE INDEX IF NOT EXISTS idx_projects_genre ON projects (genre);

-- 更新时间触发器
CREATE TRIGGER update_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();