-- 000008_add_user_password.down.sql
-- 移除密码散列字段

ALTER TABLE users DROP COLUMN IF EXISTS password_hash;