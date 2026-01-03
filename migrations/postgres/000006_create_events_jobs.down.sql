-- 000006_create_events_jobs.down.sql
-- 回滚事件和任务表

DROP TABLE IF EXISTS audit_logs_2026_01;

DROP TABLE IF EXISTS audit_logs_2026_02;

DROP TABLE IF EXISTS audit_logs_2026_03;

DROP TABLE IF EXISTS audit_logs_2026_04;

DROP TABLE IF EXISTS audit_logs_2026_05;

DROP TABLE IF EXISTS audit_logs_2026_06;

DROP TABLE IF EXISTS audit_logs;

DROP TABLE IF EXISTS generation_jobs;

DROP TABLE IF EXISTS events;