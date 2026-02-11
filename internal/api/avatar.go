package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mounis-bhat/starter/internal/config"
	"github.com/mounis-bhat/starter/internal/storage"
	"github.com/mounis-bhat/starter/internal/storage/blob"
	"github.com/mounis-bhat/starter/internal/storage/db"
)

const (
	avatarMaxBytesDefault = 5 * 1024 * 1024
)

type AvatarHandler struct {
	queries   *db.Queries
	blob      *blob.Client
	maxBytes  int64
	allowList map[string]string
}

type AvatarUploadURLRequest struct {
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type AvatarUploadURLResponse struct {
	Key       string              `json:"key"`
	URL       string              `json:"url"`
	Method    string              `json:"method"`
	Headers   map[string][]string `json:"headers"`
	ExpiresAt time.Time           `json:"expires_at"`
}

type AvatarConfirmRequest struct {
	Key string `json:"key"`
}

type AvatarURLResponse struct {
	URL       *string    `json:"url"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func NewAvatarHandler(store *storage.Store, blobClient *blob.Client, cfg config.StorageConfig) *AvatarHandler {
	maxBytes := cfg.AvatarMaxBytes
	if maxBytes <= 0 {
		maxBytes = avatarMaxBytesDefault
	}

	return &AvatarHandler{
		queries:  store.Queries,
		blob:     blobClient,
		maxBytes: maxBytes,
		allowList: map[string]string{
			"image/jpeg": "jpg",
			"image/png":  "png",
			"image/webp": "webp",
		},
	}
}

// HandleAvatarUploadURL creates a presigned PUT URL for avatar uploads
// @Summary      Get avatar upload URL
// @Description  Creates a presigned PUT URL for uploading a profile image
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body AvatarUploadURLRequest true "Upload URL request"
// @Success      200  {object}  AvatarUploadURLResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      503  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/avatar/upload-url [post]
func (h *AvatarHandler) HandleAvatarUploadURL(w http.ResponseWriter, r *http.Request) {
	if h.blob == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "storage unavailable"})
		return
	}

	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req AvatarUploadURLRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	contentType := strings.ToLower(strings.TrimSpace(strings.Split(req.ContentType, ";")[0]))
	ext, ok := h.allowList[contentType]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported content type"})
		return
	}

	if req.Size <= 0 || req.Size > h.maxBytes {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid file size"})
		return
	}

	if user.ID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	key := "users/" + user.ID + "/avatar." + ext

	presigned, err := h.blob.PresignPutObject(r.Context(), key, contentType)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create upload url"})
		return
	}

	writeJSON(w, http.StatusOK, AvatarUploadURLResponse{
		Key:       key,
		URL:       presigned.URL,
		Method:    presigned.Method,
		Headers:   presigned.Headers,
		ExpiresAt: presigned.Expires,
	})
}

// HandleAvatarConfirm confirms the uploaded avatar and saves it
// @Summary      Confirm avatar upload
// @Description  Validates the uploaded object and stores it on the user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body AvatarConfirmRequest true "Confirm upload request"
// @Success      200  {object}  AvatarURLResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      503  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/avatar/confirm [post]
func (h *AvatarHandler) HandleAvatarConfirm(w http.ResponseWriter, r *http.Request) {
	if h.blob == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "storage unavailable"})
		return
	}

	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req AvatarConfirmRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	key := strings.TrimSpace(req.Key)
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid key"})
		return
	}

	prefix := "users/" + user.ID + "/"
	if !h.isAllowedAvatarKey(key, prefix) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid key"})
		return
	}

	if err := h.blob.HeadObject(r.Context(), key); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "upload not found"})
		return
	}

	userID := uuidFromString(user.ID)
	if !userID.Valid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	stored, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	_, err = h.queries.UpdateUser(r.Context(), db.UpdateUserParams{
		ID:      userID,
		Picture: pgtype.Text{String: key, Valid: true},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	if stored.Picture.Valid {
		oldKey := strings.TrimSpace(stored.Picture.String)
		if oldKey != "" && oldKey != key && shouldDeleteAvatarKey(oldKey, prefix) {
			_ = h.blob.DeleteObject(r.Context(), oldKey)
		}
	}

	presigned, err := h.blob.PresignGetObject(r.Context(), key)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create download url"})
		return
	}

	url := presigned.URL
	writeJSON(w, http.StatusOK, AvatarURLResponse{
		URL:       &url,
		ExpiresAt: &presigned.Expires,
	})
}

// HandleAvatarURL returns a presigned URL for the user's avatar
// @Summary      Get avatar URL
// @Description  Returns a presigned GET URL for the current avatar
// @Tags         auth
// @Produce      json
// @Success      200  {object}  AvatarURLResponse
// @Failure      401  {object}  map[string]string
// @Failure      503  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/avatar-url [get]
func (h *AvatarHandler) HandleAvatarURL(w http.ResponseWriter, r *http.Request) {
	if h.blob == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "storage unavailable"})
		return
	}

	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	userID := uuidFromString(user.ID)
	if !userID.Valid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	stored, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	if !stored.Picture.Valid || strings.TrimSpace(stored.Picture.String) == "" {
		writeJSON(w, http.StatusOK, AvatarURLResponse{})
		return
	}

	value := strings.TrimSpace(stored.Picture.String)
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		writeJSON(w, http.StatusOK, AvatarURLResponse{URL: &value})
		return
	}

	presigned, err := h.blob.PresignGetObject(r.Context(), value)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create download url"})
		return
	}

	url := presigned.URL
	writeJSON(w, http.StatusOK, AvatarURLResponse{
		URL:       &url,
		ExpiresAt: &presigned.Expires,
	})
}

func shouldDeleteAvatarKey(value, prefix string) bool {
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return false
	}
	if !strings.HasPrefix(value, prefix+"avatar.") {
		return false
	}
	return true
}

func (h *AvatarHandler) isAllowedAvatarKey(key, prefix string) bool {
	if !strings.HasPrefix(key, prefix+"avatar.") {
		return false
	}

	ext := strings.TrimPrefix(key, prefix+"avatar.")
	if ext == "" {
		return false
	}
	for _, allowed := range h.allowList {
		if ext == allowed {
			return true
		}
	}
	return false
}
