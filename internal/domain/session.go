package domain

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mounis-bhat/starter/internal/storage/db"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

type SessionUser struct {
	ID            string
	Email         string
	EmailVerified bool
	Name          string
	Picture       *string
	Provider      string
}

type SessionInfo struct {
	ID           pgtype.UUID
	TokenHash    string
	ExpiresAt    time.Time
	LastActiveAt time.Time
	User         SessionUser
}

type SessionService struct {
	queries       *db.Queries
	sessionMaxAge time.Duration
	idleTimeout   time.Duration
}

func NewSessionService(queries *db.Queries, sessionMaxAge, idleTimeout time.Duration) *SessionService {
	return &SessionService{
		queries:       queries,
		sessionMaxAge: sessionMaxAge,
		idleTimeout:   idleTimeout,
	}
}

func (s *SessionService) CreateSession(ctx context.Context, userID pgtype.UUID, ipAddress *netip.Addr, userAgent string) (string, db.Session, error) {
	if err := s.enforceSessionLimit(ctx, userID, 5); err != nil {
		return "", db.Session{}, err
	}

	token, err := generateToken(32)
	if err != nil {
		return "", db.Session{}, err
	}

	tokenHash := HashToken(token)
	userAgentText := pgtype.Text{String: userAgent, Valid: userAgent != ""}

	session, err := s.queries.CreateSession(ctx, db.CreateSessionParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(s.sessionMaxAge), Valid: true},
		IpAddress: ipAddress,
		UserAgent: userAgentText,
	})
	if err != nil {
		return "", db.Session{}, err
	}

	if err := s.enforceSessionLimit(ctx, userID, 5); err != nil {
		return "", db.Session{}, err
	}

	return token, session, nil
}

func (s *SessionService) RevokeUserSessions(ctx context.Context, userID pgtype.UUID) error {
	return s.queries.DeleteUserSessions(ctx, userID)
}

func (s *SessionService) enforceSessionLimit(ctx context.Context, userID pgtype.UUID, limit int) error {
	if limit <= 0 {
		return nil
	}

	for {
		count, err := s.queries.CountUserSessions(ctx, userID)
		if err != nil {
			return err
		}
		if count < int64(limit) {
			return nil
		}

		oldest, err := s.queries.GetOldestUserSession(ctx, userID)
		if err != nil {
			return err
		}
		if err := s.queries.DeleteSession(ctx, oldest.ID); err != nil {
			return err
		}
	}
}

func (s *SessionService) ValidateToken(ctx context.Context, token string) (*SessionInfo, error) {
	if token == "" {
		return nil, ErrSessionNotFound
	}

	tokenHash := HashToken(token)
	row, err := s.queries.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	lastActiveAt := row.LastActiveAt.Time
	if !row.LastActiveAt.Valid {
		lastActiveAt = row.CreatedAt.Time
	}

	if s.idleTimeout > 0 && lastActiveAt.Add(s.idleTimeout).Before(time.Now()) {
		_ = s.queries.DeleteSessionByTokenHash(ctx, tokenHash)
		return nil, ErrSessionExpired
	}

	if row.ExpiresAt.Valid && row.ExpiresAt.Time.Before(time.Now()) {
		_ = s.queries.DeleteSessionByTokenHash(ctx, tokenHash)
		return nil, ErrSessionExpired
	}

	if err := s.queries.UpdateSessionLastActive(ctx, row.ID); err != nil {
		return nil, err
	}

	return &SessionInfo{
		ID:           row.ID,
		TokenHash:    tokenHash,
		ExpiresAt:    row.ExpiresAt.Time,
		LastActiveAt: lastActiveAt,
		User: SessionUser{
			ID:            uuidToString(row.UserID_2),
			Email:         row.UserEmail,
			EmailVerified: row.UserEmailVerified,
			Name:          row.UserName,
			Picture:       textToPointer(row.UserPicture),
			Provider:      row.UserProvider,
		},
	}, nil
}

func (s *SessionService) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	if tokenHash == "" {
		return nil
	}
	return s.queries.DeleteSessionByTokenHash(ctx, tokenHash)
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func generateToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	value, err := uuid.FromBytes(id.Bytes[:])
	if err != nil {
		return ""
	}
	return value.String()
}

func textToPointer(text pgtype.Text) *string {
	if !text.Valid {
		return nil
	}
	return &text.String
}
