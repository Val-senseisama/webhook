package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"webhook/internal/db"
	"webhook/internal/middleware"
	"webhook/internal/signing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type APIKeysHandler struct {
	DB *db.DB
}

func (h *APIKeysHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Delete("/{id}", h.delete)
	}
}

// apiKeyResponse is what we return — never expose key_hash.
type apiKeyResponse struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// createResponse extends the list response with the raw key shown exactly once.
type createResponse struct {
	apiKeyResponse
	Key string `json:"key"`
}

func toResponse(k db.APIKey) apiKeyResponse {
	return apiKeyResponse{
		ID:        k.ID,
		TenantID:  k.TenantID,
		Name:      k.Name,
		CreatedAt: k.CreatedAt,
	}
}

func (h *APIKeysHandler) list(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromContext(r.Context())
	keys, err := h.DB.ListAPIKeys(r.Context(), tenantID)
	if err != nil {
		slog.Error("list api keys", "error", err)
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}
	resp := make([]apiKeyResponse, len(keys))
	for i, k := range keys {
		resp[i] = toResponse(k)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *APIKeysHandler) create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromContext(r.Context())

	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Name == "" {
		writeJSON(w, http.StatusBadRequest, errMsg("name required"))
		return
	}

	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		writeJSON(w, http.StatusInternalServerError, errMsg("failed to generate key"))
		return
	}
	rawKey := "whk_" + hex.EncodeToString(rawBytes)
	keyHash := signing.HashAPIKey(rawKey)

	key, err := h.DB.CreateAPIKey(r.Context(), db.CreateAPIKeyParams{
		TenantID: tenantID,
		Name:     body.Name,
		KeyHash:  keyHash,
	})
	if err != nil {
		slog.Error("create api key", "error", err)
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}

	slog.Info("api key created", "key_id", key.ID, "name", key.Name)
	writeJSON(w, http.StatusCreated, createResponse{
		apiKeyResponse: toResponse(key),
		Key:            rawKey,
	})
}

func (h *APIKeysHandler) delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid id"))
		return
	}

	// Prevent deleting the key being used for this request — would lock caller out.
	callerHash := signing.HashAPIKey(extractBearerToken(r))
	current, err := h.DB.GetAPIKey(r.Context(), id, tenantID)
	if err != nil {
		if db.IsNotFound(err) {
			writeJSON(w, http.StatusNotFound, errMsg("api key not found"))
			return
		}
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}
	if current.KeyHash == callerHash {
		writeJSON(w, http.StatusConflict, errMsg("cannot delete the key you are currently using"))
		return
	}

	if err := h.DB.DeleteAPIKey(r.Context(), id, tenantID); err != nil {
		if db.IsNotFound(err) {
			writeJSON(w, http.StatusNotFound, errMsg("api key not found"))
			return
		}
		slog.Error("delete api key", "error", err)
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}

	slog.Info("api key deleted", "key_id", id)
	w.WriteHeader(http.StatusNoContent)
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}
