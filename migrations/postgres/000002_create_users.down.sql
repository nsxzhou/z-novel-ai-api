-- 000002_create_users.down.sql
-- 回滚用户表

DROP TRIGGER IF EXISTS update_users_updated_at ON users;

DROP TABLE IF EXISTS users;