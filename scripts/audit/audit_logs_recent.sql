-- Recent audit log entries (last 7 days)
SELECT id, user_id, event_type, ip_address, user_agent, metadata, created_at
FROM audit_logs
WHERE created_at >= NOW() - INTERVAL '7 days'
ORDER BY created_at DESC;
