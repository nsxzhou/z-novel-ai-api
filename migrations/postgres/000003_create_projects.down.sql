-- 000003_create_projects.down.sql
-- 回滚项目表

DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;

DROP TABLE IF EXISTS projects;