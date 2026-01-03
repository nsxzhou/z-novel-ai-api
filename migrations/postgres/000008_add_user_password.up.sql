-- 000008_add_user_password.up.sql
-- 为用户表添加密码散列字段

ALTER TABLE users ADD COLUMN password_hash VARCHAR(255);