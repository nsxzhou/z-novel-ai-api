-- 000004_create_volumes_chapters.up.sql
-- 创建卷和章节表

-- 卷表
CREATE TABLE IF NOT EXISTS volumes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    seq_num INT NOT NULL,
    title VARCHAR(255),
    description TEXT,
    summary TEXT,
    word_count INT DEFAULT 0,
    status VARCHAR(32) DEFAULT 'draft',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (project_id, seq_num)
);

CREATE INDEX IF NOT EXISTS idx_volumes_project_seq ON volumes (project_id, seq_num);

CREATE TRIGGER update_volumes_updated_at
    BEFORE UPDATE ON volumes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 章节表
CREATE TABLE IF NOT EXISTS chapters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    volume_id UUID REFERENCES volumes (id) ON DELETE SET NULL,
    seq_num INT NOT NULL,
    title VARCHAR(255),
    outline TEXT,
    content_text TEXT,
    summary TEXT,
    notes TEXT,
    story_time_start BIGINT,
    story_time_end BIGINT,
    word_count INT DEFAULT 0,
    status VARCHAR(32) DEFAULT 'draft',
    generation_metadata JSONB,
    version INT DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (
        project_id,
        volume_id,
        seq_num
    )
);

CREATE INDEX IF NOT EXISTS idx_chapters_project_seq ON chapters (project_id, seq_num);

CREATE INDEX IF NOT EXISTS idx_chapters_volume ON chapters (volume_id);

CREATE INDEX IF NOT EXISTS idx_chapters_status ON chapters (status);

CREATE INDEX IF NOT EXISTS idx_chapters_story_time ON chapters (
    project_id,
    story_time_start,
    story_time_end
);

CREATE TRIGGER update_chapters_updated_at
    BEFORE UPDATE ON chapters
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();