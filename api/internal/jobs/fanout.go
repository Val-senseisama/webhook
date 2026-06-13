package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"webhook/internal/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type FanoutArgs struct {
	EventID string `json:"event_id"`
}

func (FanoutArgs) Kind() string { return "fanout" }

type FanoutWorker struct {
	river.WorkerDefaults[FanoutArgs]
	DB    *db.DB
	River *river.Client[pgx.Tx]
}

func (w *FanoutWorker) Work(ctx context.Context, job *river.Job[FanoutArgs]) error {
	eventID, err := uuid.Parse(job.Args.EventID)
	if err != nil {
		return fmt.Errorf("invalid event id: %w", err)
	}

	event, err := w.DB.GetEvent(ctx, eventID)
	if err != nil {
		return fmt.Errorf("get event: %w", err)
	}

	endpoints, err := w.DB.GetMatchingEndpoints(ctx, event.TenantID, event.Type)
	if err != nil {
		return fmt.Errorf("get matching endpoints: %w", err)
	}

	if len(endpoints) == 0 {
		slog.Info("no matching endpoints", "event_id", eventID, "type", event.Type)
		return nil
	}

	for _, ep := range endpoints {
		delivery, err := w.DB.CreateDelivery(ctx, eventID, ep.ID)
		if err != nil {
			slog.Error("create delivery", "event_id", eventID, "endpoint_id", ep.ID, "error", err)
			continue
		}
		if _, err := w.River.Insert(ctx, DeliveryArgs{DeliveryID: delivery.ID.String()},
			&river.InsertOpts{Queue: "delivery"}); err != nil {
			slog.Error("enqueue delivery", "delivery_id", delivery.ID, "error", err)
		}
	}
	return nil
}
