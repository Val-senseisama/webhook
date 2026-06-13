package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"webhook/internal/db"
	"webhook/internal/jobs"
	"webhook/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type EventsHandler struct {
	DB    *db.DB
	River *river.Client[pgx.Tx]
}

func (h *EventsHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.list)
		r.Get("/{id}", h.get)
		r.Post("/{id}/redeliver", h.redeliver)
	}
}

func (h *EventsHandler) list(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromContext(r.Context())
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	events, err := h.DB.ListEvents(r.Context(), db.ListEventsParams{
		TenantID: tenantID,
		Type:     q.Get("type"),
		Source:   q.Get("source"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		slog.Error("list events", "error", err)
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *EventsHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid id"))
		return
	}
	event, err := h.DB.GetEvent(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errMsg("event not found"))
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (h *EventsHandler) redeliver(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid id"))
		return
	}
	if _, err := h.DB.GetEvent(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, errMsg("event not found"))
		return
	}
	if _, err := h.River.Insert(r.Context(), jobs.FanoutArgs{EventID: id.String()}, nil); err != nil {
		slog.Error("redeliver enqueue", "event_id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, errMsg("failed to enqueue"))
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}
