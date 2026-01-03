-- 000004_create_volumes_chapters.down.sql
-- 回滚卷和章节表

DROP TRIGGER IF EXISTS update_chapters_updated_at ON chapters;

DROP TABLE IF EXISTS chapters;

DROP TRIGGER IF EXISTS update_volumes_updated_at ON volumes;

DROP TABLE IF EXISTS volumes;