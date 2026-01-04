-- 000002_create_users.up.sql
-- 创建用户表

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    external_id VARCHAR(128),
    email VARCHAR(255) NOT NULL,
    name VARCHAR(128),
    avatar_url TEXT,
    role VARCHAR(32) DEFAULT 'member',
    password_hash VARCHAR(255),
    settings JSONB DEFAULT '{}',
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, email)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users (tenant_id);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

CREATE INDEX IF NOT EXISTS idx_users_external_id ON users (external_id);

-- 更新时间触发器
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();