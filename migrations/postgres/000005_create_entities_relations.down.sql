-- 000005_create_entities_relations.down.sql
-- 回滚实体和关系表

DROP TRIGGER IF EXISTS update_relations_updated_at ON relations;

DROP TABLE IF EXISTS relations;

DROP TABLE IF EXISTS entity_states;

DROP TRIGGER IF EXISTS update_entities_updated_at ON entities;

DROP TABLE IF EXISTS entities;