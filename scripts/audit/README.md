# Audit Log Scripts

Manual SQL scripts for audit log operations.

## Usage

- Run with `psql` using your database URL or connection settings.

Examples:

```bash
psql "$DATABASE_URL" -f scripts/audit/purge_audit_logs.sql
```

```bash
psql "$DATABASE_URL" -f scripts/audit/audit_logs_recent.sql
```
