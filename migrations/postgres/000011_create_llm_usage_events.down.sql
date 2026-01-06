-- 000011_create_llm_usage_events.down.sql
-- 回滚 LLM 使用量事件表

DROP TABLE IF EXISTS llm_usage_events CASCADE;

