import { getDeliveries, getAttempts } from "@/lib/api";
import { fmtDateTime } from "@/lib/fmt";
import { StatusBadge } from "@/components/status-badge";
import type { Delivery, DeliveryAttempt } from "@/lib/types";

export default async function DeliveriesPage({
  searchParams,
}: {
  searchParams: Promise<{ status?: string; endpoint_id?: string }>;
}) {
  const { status, endpoint_id } = await searchParams;
  let deliveries: Delivery[] = [];
  try {
    deliveries = await getDeliveries({ status, endpoint_id, limit: 100 }) ?? [];
  } catch {}

  const dlq = deliveries.filter((d) => d.status === "dead_lettered");
  const inFlight = deliveries.filter((d) => d.status === "in_flight");
  const pending = deliveries.filter((d) => d.status === "pending");

  return (
    <div>
      <div style={{ marginBottom: 28, display: "flex", alignItems: "flex-end", justifyContent: "space-between" }}>
        <div>
          <div style={{ fontSize: 10, letterSpacing: "0.15em", textTransform: "uppercase", color: "var(--color-muted)", marginBottom: 6 }}>
            Queue
          </div>
          <h1 style={{ margin: 0, fontSize: 22, fontWeight: 600, letterSpacing: "-0.01em" }}>
            Deliveries
          </h1>
        </div>
        <div style={{ display: "flex", gap: 16, fontSize: 11, color: "var(--color-muted)" }}>
          <span>{inFlight.length} in-flight</span>
          <span>{pending.length} pending</span>
          {dlq.length > 0 && (
            <span style={{ color: "var(--color-error)" }}>{dlq.length} dead-lettered</span>
          )}
        </div>
      </div>

      {/* DLQ alert */}
      {dlq.length > 0 && (
        <div
          style={{
            background: "rgba(239,68,68,0.07)",
            border: "1px solid rgba(239,68,68,0.3)",
            padding: "12px 16px",
            marginBottom: 16,
            fontSize: 12,
            color: "var(--color-error)",
            display: "flex",
            alignItems: "center",
            gap: 8,
          }}
        >
          <span>⚠</span>
          {dlq.length} delivery{dlq.length !== 1 ? "ies" : ""} dead-lettered — manual intervention required.
          Use the retry button or POST /v1/deliveries/:id/retry to re-queue.
        </div>
      )}

      {/* Filter bar */}
      <form method="get" style={{ display: "flex", gap: 8, marginBottom: 16 }}>
        {(["", "pending", "in_flight", "success", "failed", "dead_lettered"] as const).map((s) => (
          <a
            key={s}
            href={s ? `?status=${s}` : "?"}
            style={{
              padding: "6px 12px",
              fontSize: 10,
              letterSpacing: "0.08em",
              textTransform: "uppercase",
              textDecoration: "none",
              border: "1px solid",
              borderColor: status === s || (!status && !s) ? "var(--color-accent)" : "var(--color-border)",
              color: status === s || (!status && !s) ? "var(--color-accent)" : "var(--color-muted)",
              background: status === s || (!status && !s) ? "var(--color-accent-dim)" : "transparent",
            }}
          >
            {s || "all"}
          </a>
        ))}
      </form>

      <div style={{ background: "var(--color-surface)", border: "1px solid var(--color-border)", overflow: "hidden" }}>
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
          <thead>
            <tr style={{ borderBottom: "1px solid var(--color-border)", background: "var(--color-bg)" }}>
              {["Delivery ID", "Event ID", "Endpoint", "Status", "Attempts", "Last Attempt", "Actions"].map((h) => (
                <th key={h} style={{ padding: "10px 16px", textAlign: "left", fontSize: 10, letterSpacing: "0.1em", textTransform: "uppercase", color: "var(--color-muted)", fontWeight: 500 }}>
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {deliveries.length === 0 && (
              <tr>
                <td colSpan={7} style={{ padding: "40px 16px", textAlign: "center", color: "var(--color-muted)", fontSize: 12 }}>
                  No deliveries found
                </td>
              </tr>
            )}
            {deliveries.map((d) => (
              <DeliveryRow key={d.id} delivery={d} />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

async function DeliveryRow({ delivery: d }: { delivery: Delivery }) {
  let attempts: DeliveryAttempt[] = [];
  try {
    attempts = await getAttempts(d.id) ?? [];
  } catch {}

  const lastAttempt = attempts[attempts.length - 1];

  return (
    <tr className="row-hover" style={{ borderBottom: "1px solid var(--color-border)" }}>
      <td style={{ padding: "10px 16px", fontSize: 11, color: "var(--color-muted)" }}>
        {d.id.slice(0, 8)}…
      </td>
      <td style={{ padding: "10px 16px", fontSize: 11, color: "var(--color-muted)" }}>
        {d.event_id.slice(0, 8)}…
      </td>
      <td style={{ padding: "10px 16px", fontSize: 11, color: "var(--color-muted)" }}>
        {d.endpoint_id.slice(0, 8)}…
      </td>
      <td style={{ padding: "10px 16px" }}>
        <StatusBadge status={d.status} />
      </td>
      <td style={{ padding: "10px 16px" }}>
        <span style={{ fontSize: 12, fontVariantNumeric: "tabular-nums" }}>
          {d.attempt_count}
        </span>
        {lastAttempt?.response_status && (
          <span style={{ fontSize: 10, color: "var(--color-muted)", marginLeft: 6 }}>
            ({lastAttempt.response_status})
          </span>
        )}
        {lastAttempt?.duration_ms && (
          <span style={{ fontSize: 10, color: "var(--color-muted)", marginLeft: 4 }}>
            {lastAttempt.duration_ms}ms
          </span>
        )}
      </td>
      <td style={{ padding: "10px 16px", fontSize: 11, color: "var(--color-muted)" }}>
        {d.last_attempt_at ? fmtDateTime(d.last_attempt_at) : "—"}
      </td>
      <td style={{ padding: "10px 16px" }}>
        {(d.status === "failed" || d.status === "dead_lettered") && (
          <form action={`/api/retry/${d.id}`} method="post">
            <button
              type="submit"
              style={{
                background: "transparent",
                border: "1px solid var(--color-border-hi)",
                color: "var(--color-muted)",
                padding: "3px 10px",
                fontSize: 10,
                letterSpacing: "0.08em",
                fontFamily: "inherit",
                cursor: "pointer",
                textTransform: "uppercase",
              }}
            >
              Retry
            </button>
          </form>
        )}
      </td>
    </tr>
  );
}
