-- Audit log entries for a specific event type
-- Usage: psql "$DATABASE_URL" -v event_type='login_failure' -f scripts/audit/audit_logs_by_event.sql
SELECT id, user_id, event_type, ip_address, user_agent, metadata, created_at
FROM audit_logs
WHERE event_type = :'event_type'
ORDER BY created_at DESC;
