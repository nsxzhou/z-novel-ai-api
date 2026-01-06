-- 000008_create_conversations_artifacts.down.sql
-- 回滚会话与构件版本化表

DROP TABLE IF EXISTS artifact_versions CASCADE;
DROP TABLE IF EXISTS project_artifacts CASCADE;
DROP TABLE IF EXISTS conversation_turns CASCADE;
DROP TABLE IF EXISTS conversation_sessions CASCADE;
