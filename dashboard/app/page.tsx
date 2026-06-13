import { getDeliveries, getEndpoints, getEvents } from "@/lib/api";
import { fmtDateTime } from "@/lib/fmt";

async function StatCard({
  label,
  value,
  sub,
  accent,
}: {
  label: string;
  value: string | number;
  sub?: string;
  accent?: boolean;
}) {
  return (
    <div
      style={{
        background: "var(--color-surface)",
        border: "1px solid var(--color-border)",
        padding: "20px 24px",
        display: "flex",
        flexDirection: "column",
        gap: 6,
      }}
    >
      <div
        style={{
          fontSize: 10,
          letterSpacing: "0.12em",
          textTransform: "uppercase",
          color: "var(--color-muted)",
        }}
      >
        {label}
      </div>
      <div
        style={{
          fontSize: 32,
          fontWeight: 700,
          letterSpacing: "-0.02em",
          color: accent ? "var(--color-accent)" : "var(--color-text)",
          lineHeight: 1,
        }}
      >
        {value}
      </div>
      {sub && (
        <div style={{ fontSize: 11, color: "var(--color-muted)" }}>{sub}</div>
      )}
    </div>
  );
}

export default async function OverviewPage() {
  const [endpoints, events, deliveries] = await Promise.allSettled([
    getEndpoints(),
    getEvents({ limit: 100 }),
    getDeliveries({ limit: 200 }),
  ]);

  const eps = endpoints.status === "fulfilled" ? endpoints.value ?? [] : [];
  const evs = events.status === "fulfilled" ? events.value ?? [] : [];
  const dels = deliveries.status === "fulfilled" ? deliveries.value ?? [] : [];

  const activeEndpoints = eps.filter((e) => e.enabled).length;
  const successRate =
    dels.length > 0
      ? ((dels.filter((d) => d.status === "success").length / dels.length) * 100).toFixed(1)
      : "—";
  const dlq = dels.filter((d) => d.status === "dead_lettered").length;
  const inFlight = dels.filter((d) => d.status === "in_flight" || d.status === "pending").length;

  const recentEvents = evs.slice(0, 8);

  return (
    <div>
      {/* Header */}
      <div style={{ marginBottom: 32 }}>
        <div
          style={{
            fontSize: 10,
            letterSpacing: "0.15em",
            textTransform: "uppercase",
            color: "var(--color-muted)",
            marginBottom: 6,
          }}
        >
          Dashboard
        </div>
        <h1 style={{ margin: 0, fontSize: 22, fontWeight: 600, letterSpacing: "-0.01em" }}>
          Overview
        </h1>
      </div>

      {/* Stats grid */}
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(4, 1fr)",
          gap: 1,
          marginBottom: 1,
          background: "var(--color-border)",
        }}
      >
        <StatCard label="Active Endpoints" value={activeEndpoints} sub={`${eps.length} total`} />
        <StatCard label="Success Rate" value={`${successRate}%`} sub={`${dels.length} deliveries sampled`} accent={parseFloat(successRate as string) < 95} />
        <StatCard label="In Queue" value={inFlight} sub="pending + in-flight" />
        <StatCard
          label="Dead Letter Queue"
          value={dlq}
          sub="require attention"
          accent={dlq > 0}
        />
      </div>

      <div style={{ height: 32 }} />

      {/* Recent events */}
      <div
        style={{
          fontSize: 10,
          letterSpacing: "0.12em",
          textTransform: "uppercase",
          color: "var(--color-muted)",
          marginBottom: 12,
        }}
      >
        Recent Events
      </div>

      <div
        style={{
          background: "var(--color-surface)",
          border: "1px solid var(--color-border)",
          overflow: "hidden",
        }}
      >
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
          <thead>
            <tr
              style={{
                borderBottom: "1px solid var(--color-border)",
                background: "var(--color-bg)",
              }}
            >
              {["Event ID", "Type", "Source", "Status", "Time"].map((h) => (
                <th
                  key={h}
                  style={{
                    padding: "10px 16px",
                    textAlign: "left",
                    fontSize: 10,
                    letterSpacing: "0.1em",
                    textTransform: "uppercase",
                    color: "var(--color-muted)",
                    fontWeight: 500,
                  }}
                >
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {recentEvents.length === 0 && (
              <tr>
                <td
                  colSpan={5}
                  style={{
                    padding: "32px 16px",
                    textAlign: "center",
                    color: "var(--color-muted)",
                    fontSize: 12,
                  }}
                >
                  No events yet — send a POST to /ingest/:source
                </td>
              </tr>
            )}
            {recentEvents.map((ev) => (
              <tr
                key={ev.id}
                className="row-hover"
                style={{ borderBottom: "1px solid var(--color-border)" }}
              >
                <td style={{ padding: "10px 16px", fontSize: 11, color: "var(--color-muted)" }}>
                  {ev.id.slice(0, 8)}…
                </td>
                <td style={{ padding: "10px 16px", fontSize: 12, color: "var(--color-accent)" }}>
                  {ev.type}
                </td>
                <td style={{ padding: "10px 16px", fontSize: 12 }}>{ev.source}</td>
                <td style={{ padding: "10px 16px" }}>
                  <span
                    style={{
                      fontSize: 10,
                      color:
                        ev.status === "delivered"
                          ? "var(--color-success)"
                          : ev.status === "failed" || ev.status === "dead_lettered"
                          ? "var(--color-error)"
                          : "var(--color-muted)",
                    }}
                  >
                    {ev.status}
                  </span>
                </td>
                <td style={{ padding: "10px 16px", fontSize: 11, color: "var(--color-muted)" }}>
                  {fmtDateTime(ev.created_at)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
