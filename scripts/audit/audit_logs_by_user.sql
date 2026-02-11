-- Audit log entries for a specific user
-- Usage: psql "$DATABASE_URL" -v user_id='UUID' -f scripts/audit/audit_logs_by_user.sql
SELECT id, user_id, event_type, ip_address, user_agent, metadata, created_at
FROM audit_logs
WHERE user_id = :'user_id'
ORDER BY created_at DESC;
