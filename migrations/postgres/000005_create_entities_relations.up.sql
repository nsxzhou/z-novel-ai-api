-- 000005_create_entities_relations.up.sql
-- 创建实体和关系表

-- 实体表
CREATE TABLE IF NOT EXISTS entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    aliases TEXT[],
    type VARCHAR(32) NOT NULL,
    description TEXT,
    attributes JSONB DEFAULT '{}',
    current_state TEXT,
    first_appear_chapter_id UUID REFERENCES chapters(id),
    last_appear_chapter_id UUID REFERENCES chapters(id),
    appear_count INT DEFAULT 0,
    importance VARCHAR(16) DEFAULT 'secondary',
    vector_id VARCHAR(64),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(project_id, name)
);

CREATE INDEX IF NOT EXISTS idx_entities_project ON entities (project_id);

CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(project_id, type);

CREATE INDEX IF NOT EXISTS idx_entities_name ON entities (project_id, name);

CREATE INDEX IF NOT EXISTS idx_entities_importance ON entities (project_id, importance);

CREATE INDEX IF NOT EXISTS idx_entities_attributes ON entities USING GIN (attributes);

CREATE INDEX IF NOT EXISTS idx_entities_aliases ON entities USING GIN (aliases);

CREATE TRIGGER update_entities_updated_at
    BEFORE UPDATE ON entities
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 实体状态历史表
CREATE TABLE IF NOT EXISTS entity_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    entity_id UUID NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES chapters (id),
    story_time BIGINT,
    state_description TEXT NOT NULL,
    attribute_changes JSONB,
    event_summary TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_entity_states_entity ON entity_states (entity_id);

CREATE INDEX IF NOT EXISTS idx_entity_states_chapter ON entity_states (chapter_id);

CREATE INDEX IF NOT EXISTS idx_entity_states_time ON entity_states (entity_id, story_time);

-- 关系表
CREATE TABLE IF NOT EXISTS relations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    source_entity_id UUID NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    target_entity_id UUID NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    relation_type VARCHAR(64) NOT NULL,
    strength FLOAT DEFAULT 0.5,
    description TEXT,
    attributes JSONB DEFAULT '{}',
    first_chapter_id UUID REFERENCES chapters (id),
    last_chapter_id UUID REFERENCES chapters (id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (
        project_id,
        source_entity_id,
        target_entity_id,
        relation_type
    )
);

CREATE INDEX IF NOT EXISTS idx_relations_project ON relations (project_id);

CREATE INDEX IF NOT EXISTS idx_relations_source ON relations (source_entity_id);

CREATE INDEX IF NOT EXISTS idx_relations_target ON relations (target_entity_id);

CREATE INDEX IF NOT EXISTS idx_relations_type ON relations (project_id, relation_type);

CREATE TRIGGER update_relations_updated_at
    BEFORE UPDATE ON relations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();