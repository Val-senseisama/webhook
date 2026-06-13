package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"

	"webhook/internal/db"
	"webhook/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type EndpointsHandler struct {
	DB *db.DB
}

func (h *EndpointsHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Patch("/{id}", h.update)
		r.Delete("/{id}", h.delete)
		r.Post("/{id}/rotate-secret", h.rotateSecret)

		// Subscriptions nested under endpoint
		r.Get("/{id}/subscriptions", h.listSubs)
		r.Post("/{id}/subscriptions", h.createSub)
		r.Delete("/{id}/subscriptions/{subID}", h.deleteSub)
	}
}

func (h *EndpointsHandler) list(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromContext(r.Context())
	endpoints, err := h.DB.ListEndpoints(r.Context(), tenantID)
	if err != nil {
		slog.Error("list endpoints", "error", err)
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}
	writeJSON(w, http.StatusOK, endpoints)
}

func (h *EndpointsHandler) create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromContext(r.Context())
	var body struct {
		Name       string `json:"name"`
		URL        string `json:"url"`
		TimeoutMs  int    `json:"timeout_ms"`
		MaxRetries int    `json:"max_retries"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid JSON"))
		return
	}
	if body.Name == "" || body.URL == "" {
		writeJSON(w, http.StatusBadRequest, errMsg("name and url required"))
		return
	}
	if body.TimeoutMs == 0 {
		body.TimeoutMs = 30000
	}
	if body.MaxRetries == 0 {
		body.MaxRetries = 5
	}

	secret := generateSecret()
	ep, err := h.DB.CreateEndpoint(r.Context(), db.CreateEndpointParams{
		TenantID:   tenantID,
		Name:       body.Name,
		URL:        body.URL,
		Secret:     secret,
		TimeoutMs:  body.TimeoutMs,
		MaxRetries: body.MaxRetries,
	})
	if err != nil {
		slog.Error("create endpoint", "error", err)
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}
	writeJSON(w, http.StatusCreated, ep)
}

func (h *EndpointsHandler) get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid id"))
		return
	}
	ep, err := h.DB.GetEndpoint(r.Context(), id, tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errMsg("endpoint not found"))
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}
	writeJSON(w, http.StatusOK, ep)
}

func (h *EndpointsHandler) update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid id"))
		return
	}
	var body struct {
		Name       string `json:"name"`
		URL        string `json:"url"`
		Enabled    bool   `json:"enabled"`
		TimeoutMs  int    `json:"timeout_ms"`
		MaxRetries int    `json:"max_retries"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid JSON"))
		return
	}
	ep, err := h.DB.UpdateEndpoint(r.Context(), db.UpdateEndpointParams{
		ID:         id,
		TenantID:   tenantID,
		Name:       body.Name,
		URL:        body.URL,
		Enabled:    body.Enabled,
		TimeoutMs:  body.TimeoutMs,
		MaxRetries: body.MaxRetries,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errMsg("endpoint not found"))
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}
	writeJSON(w, http.StatusOK, ep)
}

func (h *EndpointsHandler) delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid id"))
		return
	}
	if err := h.DB.DeleteEndpoint(r.Context(), id, tenantID); err != nil {
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *EndpointsHandler) rotateSecret(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid id"))
		return
	}
	newSecret := generateSecret()
	if err := h.DB.RotateEndpointSecret(r.Context(), id, tenantID, newSecret); err != nil {
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"secret": newSecret})
}

func (h *EndpointsHandler) listSubs(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid id"))
		return
	}
	subs, err := h.DB.ListSubscriptions(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

func (h *EndpointsHandler) createSub(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid id"))
		return
	}
	var body struct {
		EventTypes []string `json:"event_types"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid JSON"))
		return
	}
	if len(body.EventTypes) == 0 {
		body.EventTypes = []string{"*"}
	}
	sub, err := h.DB.CreateSubscription(r.Context(), db.CreateSubscriptionParams{
		EndpointID: id,
		EventTypes: body.EventTypes,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}
	writeJSON(w, http.StatusCreated, sub)
}

func (h *EndpointsHandler) deleteSub(w http.ResponseWriter, r *http.Request) {
	endpointID, _ := uuid.Parse(chi.URLParam(r, "id"))
	subID, err := uuid.Parse(chi.URLParam(r, "subID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errMsg("invalid sub id"))
		return
	}
	if err := h.DB.DeleteSubscription(r.Context(), subID, endpointID); err != nil {
		writeJSON(w, http.StatusInternalServerError, errMsg("internal error"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func generateSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "whsec_" + hex.EncodeToString(b)
}

func errMsg(msg string) map[string]string {
	return map[string]string{"error": msg}
}
