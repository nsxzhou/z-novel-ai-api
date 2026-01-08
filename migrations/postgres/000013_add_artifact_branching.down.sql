-- 000013_add_artifact_branching.down.sql
-- 回滚构件版本“多分支”字段

DROP INDEX IF EXISTS idx_artifact_versions_artifact_branch_version_no;

ALTER TABLE artifact_versions
    DROP CONSTRAINT IF EXISTS fk_artifact_versions_parent;

ALTER TABLE artifact_versions
    DROP COLUMN IF EXISTS parent_version_id,
    DROP COLUMN IF EXISTS branch_key;
