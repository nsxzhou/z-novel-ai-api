-- 000007_enable_rls.up.sql
-- 启用 Row-Level Security

-- 租户上下文函数
CREATE OR REPLACE FUNCTION set_tenant_id(tenant UUID) RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', tenant::TEXT, TRUE);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION current_tenant_id() RETURNS UUID AS $$
BEGIN
    RETURN NULLIF(current_setting('app.current_tenant_id', TRUE), '')::UUID;
END;
$$ LANGUAGE plpgsql;

-- 启用 RLS
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;

ALTER TABLE volumes ENABLE ROW LEVEL SECURITY;

ALTER TABLE chapters ENABLE ROW LEVEL SECURITY;

ALTER TABLE entities ENABLE ROW LEVEL SECURITY;

ALTER TABLE entity_states ENABLE ROW LEVEL SECURITY;

ALTER TABLE relations ENABLE ROW LEVEL SECURITY;

ALTER TABLE events ENABLE ROW LEVEL SECURITY;

ALTER TABLE generation_jobs ENABLE ROW LEVEL SECURITY;

-- 项目表策略
CREATE POLICY tenant_isolation_select ON projects FOR
SELECT USING (
        tenant_id = current_tenant_id ()
    );

CREATE POLICY tenant_isolation_insert ON projects FOR
INSERT
WITH
    CHECK (
        tenant_id = current_tenant_id ()
    );

CREATE POLICY tenant_isolation_update ON projects FOR
UPDATE USING (
    tenant_id = current_tenant_id ()
);

CREATE POLICY tenant_isolation_delete ON projects FOR DELETE USING (
    tenant_id = current_tenant_id ()
);

-- 卷表策略（通过 project_id 关联）
CREATE POLICY tenant_isolation_select ON volumes FOR
SELECT USING (
        project_id IN (
            SELECT id
            FROM projects
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_insert ON volumes FOR
INSERT
WITH
    CHECK (
        project_id IN (
            SELECT id
            FROM projects
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_update ON volumes FOR
UPDATE USING (
    project_id IN (
        SELECT id
        FROM projects
        WHERE
            tenant_id = current_tenant_id ()
    )
);

CREATE POLICY tenant_isolation_delete ON volumes FOR DELETE USING (
    project_id IN (
        SELECT id
        FROM projects
        WHERE
            tenant_id = current_tenant_id ()
    )
);

-- 章节表策略
CREATE POLICY tenant_isolation_select ON chapters FOR
SELECT USING (
        project_id IN (
            SELECT id
            FROM projects
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_insert ON chapters FOR
INSERT
WITH
    CHECK (
        project_id IN (
            SELECT id
            FROM projects
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_update ON chapters FOR
UPDATE USING (
    project_id IN (
        SELECT id
        FROM projects
        WHERE
            tenant_id = current_tenant_id ()
    )
);

CREATE POLICY tenant_isolation_delete ON chapters FOR DELETE USING (
    project_id IN (
        SELECT id
        FROM projects
        WHERE
            tenant_id = current_tenant_id ()
    )
);

-- 实体表策略
CREATE POLICY tenant_isolation_select ON entities FOR
SELECT USING (
        project_id IN (
            SELECT id
            FROM projects
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_insert ON entities FOR
INSERT
WITH
    CHECK (
        project_id IN (
            SELECT id
            FROM projects
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_update ON entities FOR
UPDATE USING (
    project_id IN (
        SELECT id
        FROM projects
        WHERE
            tenant_id = current_tenant_id ()
    )
);

CREATE POLICY tenant_isolation_delete ON entities FOR DELETE USING (
    project_id IN (
        SELECT id
        FROM projects
        WHERE
            tenant_id = current_tenant_id ()
    )
);

-- 实体状态表策略
CREATE POLICY tenant_isolation_select ON entity_states FOR
SELECT USING (
        entity_id IN (
            SELECT e.id
            FROM entities e
                JOIN projects p ON e.project_id = p.id
            WHERE
                p.tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_insert ON entity_states FOR
INSERT
WITH
    CHECK (
        entity_id IN (
            SELECT e.id
            FROM entities e
                JOIN projects p ON e.project_id = p.id
            WHERE
                p.tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_delete ON entity_states FOR DELETE USING (
    entity_id IN (
        SELECT e.id
        FROM entities e
            JOIN projects p ON e.project_id = p.id
        WHERE
            p.tenant_id = current_tenant_id ()
    )
);

-- 关系表策略
CREATE POLICY tenant_isolation_select ON relations FOR
SELECT USING (
        project_id IN (
            SELECT id
            FROM projects
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_insert ON relations FOR
INSERT
WITH
    CHECK (
        project_id IN (
            SELECT id
            FROM projects
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_update ON relations FOR
UPDATE USING (
    project_id IN (
        SELECT id
        FROM projects
        WHERE
            tenant_id = current_tenant_id ()
    )
);

CREATE POLICY tenant_isolation_delete ON relations FOR DELETE USING (
    project_id IN (
        SELECT id
        FROM projects
        WHERE
            tenant_id = current_tenant_id ()
    )
);

-- 事件表策略
CREATE POLICY tenant_isolation_select ON events FOR
SELECT USING (
        project_id IN (
            SELECT id
            FROM projects
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_insert ON events FOR
INSERT
WITH
    CHECK (
        project_id IN (
            SELECT id
            FROM projects
            WHERE
                tenant_id = current_tenant_id ()
        )
    );

CREATE POLICY tenant_isolation_update ON events FOR
UPDATE USING (
    project_id IN (
        SELECT id
        FROM projects
        WHERE
            tenant_id = current_tenant_id ()
    )
);

CREATE POLICY tenant_isolation_delete ON events FOR DELETE USING (
    project_id IN (
        SELECT id
        FROM projects
        WHERE
            tenant_id = current_tenant_id ()
    )
);

-- 生成任务表策略
CREATE POLICY tenant_isolation_select ON generation_jobs FOR
SELECT USING (
        tenant_id = current_tenant_id ()
    );

CREATE POLICY tenant_isolation_insert ON generation_jobs FOR
INSERT
WITH
    CHECK (
        tenant_id = current_tenant_id ()
    );

CREATE POLICY tenant_isolation_update ON generation_jobs FOR
UPDATE USING (
    tenant_id = current_tenant_id ()
);

CREATE POLICY tenant_isolation_delete ON generation_jobs FOR DELETE USING (
    tenant_id = current_tenant_id ()
);