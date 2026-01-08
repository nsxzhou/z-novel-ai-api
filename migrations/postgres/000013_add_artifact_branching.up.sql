-- 000013_add_artifact_branching.up.sql
-- 为构件版本增加“多分支”能力（A/B 并行）

ALTER TABLE artifact_versions
    ADD COLUMN IF NOT EXISTS branch_key VARCHAR(64) NOT NULL DEFAULT 'main',
    ADD COLUMN IF NOT EXISTS parent_version_id UUID;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_artifact_versions_parent'
            AND conrelid = 'artifact_versions'::regclass
    ) THEN
        ALTER TABLE artifact_versions
            ADD CONSTRAINT fk_artifact_versions_parent
            FOREIGN KEY (parent_version_id) REFERENCES artifact_versions (id) ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_artifact_versions_artifact_branch_version_no ON artifact_versions (artifact_id, branch_key, version_no DESC);
