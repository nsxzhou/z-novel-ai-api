-- 000010_create_project_creation_sessions.down.sql
-- 回滚“对话创建项目”会话与轮次表

DROP TABLE IF EXISTS project_creation_turns CASCADE;
DROP TABLE IF EXISTS project_creation_sessions CASCADE;

