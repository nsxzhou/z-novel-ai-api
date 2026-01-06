-- 000009_add_ai_keys_and_constraints.up.sql
-- 为 AI 生成的可迭代对象增加稳定标识（key），并调整冲突约束

-- entities：支持同名不同类型；使用 ai_key 做稳定映射
ALTER TABLE entities ADD COLUMN IF NOT EXISTS ai_key VARCHAR(128);

ALTER TABLE entities DROP CONSTRAINT IF EXISTS entities_project_id_name_key;

ALTER TABLE entities
    ADD CONSTRAINT entities_project_id_type_name_key UNIQUE (project_id, type, name);

CREATE UNIQUE INDEX IF NOT EXISTS uq_entities_project_ai_key ON entities (project_id, ai_key)
WHERE
    ai_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_entities_ai_key ON entities (project_id, ai_key);

-- volumes：使用 ai_key 做稳定映射
ALTER TABLE volumes ADD COLUMN IF NOT EXISTS ai_key VARCHAR(128);

CREATE UNIQUE INDEX IF NOT EXISTS uq_volumes_project_ai_key ON volumes (project_id, ai_key)
WHERE
    ai_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_volumes_ai_key ON volumes (project_id, ai_key);

-- chapters：使用 ai_key 做稳定映射
ALTER TABLE chapters ADD COLUMN IF NOT EXISTS ai_key VARCHAR(128);

CREATE UNIQUE INDEX IF NOT EXISTS uq_chapters_project_ai_key ON chapters (project_id, ai_key)
WHERE
    ai_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_chapters_ai_key ON chapters (project_id, ai_key);
