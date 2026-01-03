-- 000007_enable_rls.down.sql
-- 回滚 RLS 策略

-- 删除生成任务表策略
DROP POLICY IF EXISTS tenant_isolation_select ON generation_jobs;

DROP POLICY IF EXISTS tenant_isolation_insert ON generation_jobs;

DROP POLICY IF EXISTS tenant_isolation_update ON generation_jobs;

DROP POLICY IF EXISTS tenant_isolation_delete ON generation_jobs;

-- 删除事件表策略
DROP POLICY IF EXISTS tenant_isolation_select ON events;

DROP POLICY IF EXISTS tenant_isolation_insert ON events;

DROP POLICY IF EXISTS tenant_isolation_update ON events;

DROP POLICY IF EXISTS tenant_isolation_delete ON events;

-- 删除关系表策略
DROP POLICY IF EXISTS tenant_isolation_select ON relations;

DROP POLICY IF EXISTS tenant_isolation_insert ON relations;

DROP POLICY IF EXISTS tenant_isolation_update ON relations;

DROP POLICY IF EXISTS tenant_isolation_delete ON relations;

-- 删除实体状态表策略
DROP POLICY IF EXISTS tenant_isolation_select ON entity_states;

DROP POLICY IF EXISTS tenant_isolation_insert ON entity_states;

DROP POLICY IF EXISTS tenant_isolation_delete ON entity_states;

-- 删除实体表策略
DROP POLICY IF EXISTS tenant_isolation_select ON entities;

DROP POLICY IF EXISTS tenant_isolation_insert ON entities;

DROP POLICY IF EXISTS tenant_isolation_update ON entities;

DROP POLICY IF EXISTS tenant_isolation_delete ON entities;

-- 删除章节表策略
DROP POLICY IF EXISTS tenant_isolation_select ON chapters;

DROP POLICY IF EXISTS tenant_isolation_insert ON chapters;

DROP POLICY IF EXISTS tenant_isolation_update ON chapters;

DROP POLICY IF EXISTS tenant_isolation_delete ON chapters;

-- 删除卷表策略
DROP POLICY IF EXISTS tenant_isolation_select ON volumes;

DROP POLICY IF EXISTS tenant_isolation_insert ON volumes;

DROP POLICY IF EXISTS tenant_isolation_update ON volumes;

DROP POLICY IF EXISTS tenant_isolation_delete ON volumes;

-- 删除项目表策略
DROP POLICY IF EXISTS tenant_isolation_select ON projects;

DROP POLICY IF EXISTS tenant_isolation_insert ON projects;

DROP POLICY IF EXISTS tenant_isolation_update ON projects;

DROP POLICY IF EXISTS tenant_isolation_delete ON projects;

-- 禁用 RLS
ALTER TABLE generation_jobs DISABLE ROW LEVEL SECURITY;

ALTER TABLE events DISABLE ROW LEVEL SECURITY;

ALTER TABLE relations DISABLE ROW LEVEL SECURITY;

ALTER TABLE entity_states DISABLE ROW LEVEL SECURITY;

ALTER TABLE entities DISABLE ROW LEVEL SECURITY;

ALTER TABLE chapters DISABLE ROW LEVEL SECURITY;

ALTER TABLE volumes DISABLE ROW LEVEL SECURITY;

ALTER TABLE projects DISABLE ROW LEVEL SECURITY;

-- 删除函数
DROP FUNCTION IF EXISTS current_tenant_id ();

DROP FUNCTION IF EXISTS set_tenant_id (UUID);