-- 000001_create_tenants.down.sql
-- 回滚租户表

DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;

DROP TABLE IF EXISTS tenants;
