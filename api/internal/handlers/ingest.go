package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"webhook/internal/db"
	"webhook/internal/jobs"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/riverqueue/river"
)

const maxBodySize = 256 * 1024 // 256 KB

type IngestHandler struct {
	DB              *db.DB
	River           *river.Client[pgx.Tx]
	DefaultTenantID uuid.UUID
}

func (h *IngestHandler) Handle(w http.ResponseWriter, r *http.Request) {
	source := chi.URLParam(r, "source")

	eventType := r.Header.Get("X-Event-Type")
	if eventType == "" {
		eventType = r.URL.Query().Get("type")
	}
	if eventType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "X-Event-Type header required"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read body"})
		return
	}
	if !json.Valid(body) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "payload must be valid JSON"})
		return
	}

	rawHeaders := map[string]string{
		"X-Event-Type":    eventType,
		"X-Forwarded-For": r.Header.Get("X-Forwarded-For"),
		"Content-Type":    r.Header.Get("Content-Type"),
		"Idempotency-Key": r.Header.Get("Idempotency-Key"),
	}
	headersJSON, _ := json.Marshal(rawHeaders)

	event, err := h.DB.CreateEvent(r.Context(), db.CreateEventParams{
		TenantID:       h.DefaultTenantID,
		Source:         source,
		Type:           eventType,
		Payload:        body,
		Headers:        headersJSON,
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "duplicate idempotency key"})
			return
		}
		slog.Error("create event", "source", source, "type", eventType, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if _, err := h.River.Insert(r.Context(), jobs.FanoutArgs{EventID: event.ID.String()}, nil); err != nil {
		slog.Error("enqueue fanout", "event_id", event.ID, "error", err)
	}

	slog.Info("event ingested", "event_id", event.ID, "source", source, "type", eventType)
	writeJSON(w, http.StatusAccepted, map[string]string{"event_id": event.ID.String()})
}
