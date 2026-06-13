import { getEvents } from "@/lib/api";
import { fmtDateTime } from "@/lib/fmt";
import { StatusBadge } from "@/components/status-badge";
import type { Event } from "@/lib/types";

export default async function EventsPage({
  searchParams,
}: {
  searchParams: Promise<{ type?: string; source?: string }>;
}) {
  const { type, source } = await searchParams;
  let events: Event[] = [];
  try {
    events = await getEvents({ type, source, limit: 100 }) ?? [];
  } catch {}

  return (
    <div>
      <div style={{ marginBottom: 28, display: "flex", alignItems: "flex-end", justifyContent: "space-between" }}>
        <div>
          <div style={{ fontSize: 10, letterSpacing: "0.15em", textTransform: "uppercase", color: "var(--color-muted)", marginBottom: 6 }}>
            Log
          </div>
          <h1 style={{ margin: 0, fontSize: 22, fontWeight: 600, letterSpacing: "-0.01em" }}>
            Events
          </h1>
        </div>
        <div style={{ fontSize: 11, color: "var(--color-muted)" }}>
          {events.length} events
        </div>
      </div>

      {/* Filter bar */}
      <form method="get" style={{ display: "flex", gap: 8, marginBottom: 16 }}>
        <input
          name="type"
          defaultValue={type}
          placeholder="filter by type…"
          style={{
            background: "var(--color-surface)",
            border: "1px solid var(--color-border)",
            color: "var(--color-text)",
            padding: "7px 12px",
            fontSize: 12,
            fontFamily: "inherit",
            width: 200,
            outline: "none",
          }}
        />
        <input
          name="source"
          defaultValue={source}
          placeholder="filter by source…"
          style={{
            background: "var(--color-surface)",
            border: "1px solid var(--color-border)",
            color: "var(--color-text)",
            padding: "7px 12px",
            fontSize: 12,
            fontFamily: "inherit",
            width: 200,
            outline: "none",
          }}
        />
        <button
          type="submit"
          style={{
            background: "var(--color-accent-dim)",
            border: "1px solid var(--color-accent)",
            color: "var(--color-accent)",
            padding: "7px 16px",
            fontSize: 11,
            fontFamily: "inherit",
            letterSpacing: "0.08em",
            cursor: "pointer",
          }}
        >
          FILTER
        </button>
      </form>

      <div style={{ background: "var(--color-surface)", border: "1px solid var(--color-border)", overflow: "hidden" }}>
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
          <thead>
            <tr style={{ borderBottom: "1px solid var(--color-border)", background: "var(--color-bg)" }}>
              {["ID", "Type", "Source", "Status", "Idempotency Key", "Received"].map((h) => (
                <th key={h} style={{ padding: "10px 16px", textAlign: "left", fontSize: 10, letterSpacing: "0.1em", textTransform: "uppercase", color: "var(--color-muted)", fontWeight: 500 }}>
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {events.length === 0 && (
              <tr>
                <td colSpan={6} style={{ padding: "40px 16px", textAlign: "center", color: "var(--color-muted)", fontSize: 12 }}>
                  No events match this filter
                </td>
              </tr>
            )}
            {events.map((ev) => (
              <tr
                key={ev.id}
                className="row-hover"
                style={{ borderBottom: "1px solid var(--color-border)" }}
              >
                <td style={{ padding: "10px 16px", fontSize: 11, color: "var(--color-muted)", fontVariantNumeric: "tabular-nums" }}>
                  {ev.id.slice(0, 8)}…
                </td>
                <td style={{ padding: "10px 16px" }}>
                  <span style={{ fontSize: 12, color: "var(--color-accent)", fontWeight: 500 }}>
                    {ev.type}
                  </span>
                </td>
                <td style={{ padding: "10px 16px", fontSize: 12 }}>{ev.source}</td>
                <td style={{ padding: "10px 16px" }}>
                  <StatusBadge status={ev.status} />
                </td>
                <td style={{ padding: "10px 16px", fontSize: 11, color: "var(--color-muted)" }}>
                  {ev.idempotency_key || "—"}
                </td>
                <td style={{ padding: "10px 16px", fontSize: 11, color: "var(--color-muted)", fontVariantNumeric: "tabular-nums" }}>
                  {fmtDateTime(ev.created_at)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Ingest example */}
      <div style={{ marginTop: 32, background: "var(--color-surface)", border: "1px solid var(--color-border)", padding: "20px 24px" }}>
        <div style={{ fontSize: 10, letterSpacing: "0.12em", textTransform: "uppercase", color: "var(--color-muted)", marginBottom: 12 }}>
          Send a Test Event
        </div>
        <pre style={{ margin: 0, fontSize: 12, color: "var(--color-text)", overflowX: "auto", lineHeight: 1.7 }}>
{`curl -X POST http://localhost:8080/ingest/my-service \\
  -H "X-Event-Type: order.created" \\
  -H "Idempotency-Key: evt_$(date +%s)" \\
  -H "Content-Type: application/json" \\
  -d '{"order_id": "ord_123", "amount": 4999}'`}
        </pre>
      </div>
    </div>
  );
}
