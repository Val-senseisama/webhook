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

type DeliveriesHandler struct {
	DB    *db.DB
	River *river.Client[pgx.Tx]
}

func (h *DeliveriesHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.list)
		r.Get("/{id}", h.get)
		r.Get("/{id}/attempts", h.listAttempts)
		r.Post("/{id}/retry", h.retry)
	}
}

func (h *DeliveriesHandler) list(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromContext(r.Context())
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	var endpointID *uuid.UUID
	if s := q.Get("endpoint_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			endpointID = &id
		}
	}
	var eventID *uuid.UUID
	if s := q.Get("event_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			eventID = &id
		}
	}

	deliveries, err := h.DB.ListDeliveries(r.Context(), db.ListDeliveriesParams{
		TenantID:   tenantID,
		EndpointID: endpointID,
		EventID:    eventID,
		Status:     q.Get("status"),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		slog.Error("list deliveries", "error", err)
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}
	writeJSON(w, http.StatusOK, deliveries)
}

func (h *DeliveriesHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid id"))
		return
	}
	dd, err := h.DB.GetDeliveryDetail(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errMsg("delivery not found"))
		return
	}
	writeJSON(w, http.StatusOK, dd)
}

func (h *DeliveriesHandler) listAttempts(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid id"))
		return
	}
	attempts, err := h.DB.ListAttempts(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}
	writeJSON(w, http.StatusOK, attempts)
}

func (h *DeliveriesHandler) retry(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid id"))
		return
	}
	if err := h.DB.ResetDeliveryForRetry(r.Context(), id); err != nil {
		slog.Error("reset delivery", "delivery_id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}
	if _, err := h.River.Insert(r.Context(), jobs.DeliveryArgs{DeliveryID: id.String()},
		&river.InsertOpts{Queue: "delivery"}); err != nil {
		slog.Error("enqueue retry", "delivery_id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, errMsg("failed to enqueue"))
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}
