package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"webhook/internal/db"
	"webhook/internal/signing"

	"github.com/google/uuid"
)

type contextKey string

const TenantIDKey contextKey = "tenant_id"

// APIKeyAuth extracts the Bearer token, hashes it, looks up the matching tenant,
// and injects tenant_id into the request context.
func APIKeyAuth(database *db.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
				return
			}
			raw := strings.TrimPrefix(auth, "Bearer ")
			hash := signing.HashAPIKey(raw)

			key, err := database.GetAPIKeyByHash(r.Context(), hash)
			if err != nil {
				slog.Debug("api key lookup failed", "hash", hash[:8], "error", err)
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid api key"})
				return
			}

			ctx := context.WithValue(r.Context(), TenantIDKey, key.TenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TenantFromContext pulls the authenticated tenant UUID from ctx.
func TenantFromContext(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(TenantIDKey).(uuid.UUID)
	return v
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
