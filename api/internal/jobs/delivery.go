package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"webhook/internal/db"
	"webhook/internal/signing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type DeliveryArgs struct {
	DeliveryID string `json:"delivery_id"`
}

func (DeliveryArgs) Kind() string { return "delivery" }

// RetryPolicy maps River attempt numbers to our custom backoff schedule with jitter.
type RetryPolicy struct{}

var retryDelays = []time.Duration{
	30 * time.Second,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
}

func (RetryPolicy) NextRetry(job *rivertype.JobRow) time.Time {
	idx := job.Attempt - 1
	if idx >= len(retryDelays) {
		idx = len(retryDelays) - 1
	}
	base := retryDelays[idx]
	jitter := time.Duration(rand.Int64N(int64(base / 10)))
	return time.Now().Add(base + jitter)
}

type DeliveryWorker struct {
	river.WorkerDefaults[DeliveryArgs]
	DB         *db.DB
	HTTPClient *http.Client
}

func (w *DeliveryWorker) Work(ctx context.Context, job *river.Job[DeliveryArgs]) error {
	deliveryID, err := uuid.Parse(job.Args.DeliveryID)
	if err != nil {
		return fmt.Errorf("invalid delivery id: %w", err)
	}

	dd, err := w.DB.GetDeliveryDetail(ctx, deliveryID)
	if err != nil {
		return fmt.Errorf("get delivery detail: %w", err)
	}

	if err := w.DB.MarkDeliveryInFlight(ctx, deliveryID); err != nil {
		return fmt.Errorf("mark in_flight: %w", err)
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sig := signing.Sign(dd.EndpointSecret, dd.DeliveryID.String(), timestamp, dd.EventPayload)

	reqHeaders := map[string]string{
		"Content-Type":            "application/json",
		"X-Webhook-Id":            dd.DeliveryID.String(),
		"X-Webhook-Timestamp":     timestamp,
		"X-Webhook-Signature-256": sig,
		"X-Webhook-Event-Type":    dd.EventType,
		"User-Agent":              "WebhookService/1.0",
	}
	reqHeadersJSON, _ := json.Marshal(reqHeaders)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dd.EndpointURL,
		bytes.NewReader(dd.EventPayload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	for k, v := range reqHeaders {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := w.HTTPClient.Do(req)
	durationMs := int(time.Since(start).Milliseconds())
	attemptNum := job.Attempt

	if err != nil {
		errStr := err.Error()
		_ = w.DB.RecordAttempt(ctx, db.RecordAttemptParams{
			DeliveryID:     deliveryID,
			AttemptNumber:  attemptNum,
			RequestHeaders: reqHeadersJSON,
			RequestBody:    string(dd.EventPayload),
			DurationMs:     &durationMs,
			Error:          &errStr,
		})
		if job.Attempt >= dd.MaxRetries {
			_ = w.DB.MarkDeliveryDeadLettered(ctx, deliveryID)
			slog.Warn("delivery dead-lettered", "delivery_id", deliveryID, "endpoint", dd.EndpointURL)
			return nil
		}
		return fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	bodyStr := string(body)
	respHdrs := map[string]string{}
	for k := range resp.Header {
		respHdrs[k] = resp.Header.Get(k)
	}
	respHdrsJSON, _ := json.Marshal(respHdrs)
	status := resp.StatusCode

	_ = w.DB.RecordAttempt(ctx, db.RecordAttemptParams{
		DeliveryID:      deliveryID,
		AttemptNumber:   attemptNum,
		RequestHeaders:  reqHeadersJSON,
		RequestBody:     string(dd.EventPayload),
		ResponseStatus:  &status,
		ResponseHeaders: respHdrsJSON,
		ResponseBody:    &bodyStr,
		DurationMs:      &durationMs,
	})

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_ = w.DB.MarkDeliverySuccess(ctx, deliveryID)
		slog.Info("delivery succeeded",
			"delivery_id", deliveryID,
			"status", resp.StatusCode,
			"duration_ms", durationMs,
		)
		return nil
	}

	if job.Attempt >= dd.MaxRetries {
		_ = w.DB.MarkDeliveryDeadLettered(ctx, deliveryID)
		slog.Warn("delivery dead-lettered", "delivery_id", deliveryID, "status", resp.StatusCode)
		return nil
	}

	return fmt.Errorf("non-2xx response: %d", resp.StatusCode)
}
