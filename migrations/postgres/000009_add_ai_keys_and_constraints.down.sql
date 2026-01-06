-- 000009_add_ai_keys_and_constraints.down.sql
-- 回滚 ai_key 与约束调整

DROP INDEX IF EXISTS idx_chapters_ai_key;
DROP INDEX IF EXISTS uq_chapters_project_ai_key;
ALTER TABLE chapters DROP COLUMN IF EXISTS ai_key;

DROP INDEX IF EXISTS idx_volumes_ai_key;
DROP INDEX IF EXISTS uq_volumes_project_ai_key;
ALTER TABLE volumes DROP COLUMN IF EXISTS ai_key;

DROP INDEX IF EXISTS idx_entities_ai_key;
DROP INDEX IF EXISTS uq_entities_project_ai_key;

ALTER TABLE entities DROP CONSTRAINT IF EXISTS entities_project_id_type_name_key;
ALTER TABLE entities
    ADD CONSTRAINT entities_project_id_name_key UNIQUE (project_id, name);

ALTER TABLE entities DROP COLUMN IF EXISTS ai_key;
