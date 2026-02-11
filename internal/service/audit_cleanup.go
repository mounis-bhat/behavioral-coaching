package service

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mounis-bhat/starter/internal/storage/db"
)

type AuditCleanupService struct {
	queries *db.Queries
}

func NewAuditCleanupService(queries *db.Queries) *AuditCleanupService {
	return &AuditCleanupService{queries: queries}
}

func (s *AuditCleanupService) PurgeBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	if s == nil || s.queries == nil {
		return 0, errors.New("audit cleanup service not initialized")
	}

	cutoffValue := pgtype.Timestamptz{Time: cutoff.UTC(), Valid: true}
	return s.queries.PurgeAuditLogsBefore(ctx, cutoffValue)
}
