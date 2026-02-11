-- Purge audit logs older than 90 days
DELETE FROM audit_logs
WHERE created_at < NOW() - INTERVAL '90 days';
