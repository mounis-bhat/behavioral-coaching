package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/netip"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mounis-bhat/starter/internal/storage/db"
)

type AuditLogger struct {
	queries *db.Queries
}

func NewAuditLogger(queries *db.Queries) *AuditLogger {
	return &AuditLogger{queries: queries}
}

func (l *AuditLogger) Log(ctx context.Context, event string, userID pgtype.UUID, ip *netip.Addr, userAgent string, metadata map[string]any) {
	if l == nil || l.queries == nil {
		return
	}

	var meta []byte
	if metadata != nil {
		if raw, err := json.Marshal(metadata); err == nil {
			meta = raw
		}
	}

	ua := pgtype.Text{String: userAgent, Valid: userAgent != ""}
	_ = l.queries.CreateAuditLog(ctx, db.CreateAuditLogParams{
		UserID:    userID,
		EventType: event,
		IpAddress: ip,
		UserAgent: ua,
		Metadata:  meta,
	})
}

func hashEmail(email string) string {
	sum := sha256.Sum256([]byte(email))
	return hex.EncodeToString(sum[:])
}

func uuidFromString(value string) pgtype.UUID {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}
}
