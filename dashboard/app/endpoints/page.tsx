import { getEndpoints, getSubscriptions } from "@/lib/api";
import { fmtDate } from "@/lib/fmt";
import { StatusBadge } from "@/components/status-badge";
import type { Endpoint } from "@/lib/types";

export default async function EndpointsPage() {
  let endpoints: Endpoint[] = [];
  try {
    endpoints = await getEndpoints() ?? [];
  } catch {
    // API not reachable yet
  }

  return (
    <div>
      <div style={{ marginBottom: 28, display: "flex", alignItems: "flex-end", justifyContent: "space-between" }}>
        <div>
          <div style={{ fontSize: 10, letterSpacing: "0.15em", textTransform: "uppercase", color: "var(--color-muted)", marginBottom: 6 }}>
            Management
          </div>
          <h1 style={{ margin: 0, fontSize: 22, fontWeight: 600, letterSpacing: "-0.01em" }}>
            Endpoints
          </h1>
        </div>
        <div style={{ fontSize: 11, color: "var(--color-muted)" }}>
          {endpoints.length} registered
        </div>
      </div>

      <div style={{ background: "var(--color-surface)", border: "1px solid var(--color-border)", overflow: "hidden" }}>
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
          <thead>
            <tr style={{ borderBottom: "1px solid var(--color-border)", background: "var(--color-bg)" }}>
              {["Name", "URL", "Status", "Timeout", "Max Retries", "Created"].map((h) => (
                <th key={h} style={{ padding: "10px 16px", textAlign: "left", fontSize: 10, letterSpacing: "0.1em", textTransform: "uppercase", color: "var(--color-muted)", fontWeight: 500 }}>
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {endpoints.length === 0 && (
              <tr>
                <td colSpan={6} style={{ padding: "40px 16px", textAlign: "center", color: "var(--color-muted)", fontSize: 12 }}>
                  No endpoints yet — POST /v1/endpoints to register one
                </td>
              </tr>
            )}
            {endpoints.map((ep) => (
              <EndpointRow key={ep.id} ep={ep} />
            ))}
          </tbody>
        </table>
      </div>

      {/* Quick-start card */}
      <div style={{ marginTop: 32, background: "var(--color-surface)", border: "1px solid var(--color-border)", padding: "20px 24px" }}>
        <div style={{ fontSize: 10, letterSpacing: "0.12em", textTransform: "uppercase", color: "var(--color-muted)", marginBottom: 12 }}>
          Quick Register
        </div>
        <pre style={{ margin: 0, fontSize: 12, color: "var(--color-text)", overflowX: "auto", lineHeight: 1.7 }}>
{`curl -X POST http://localhost:8080/v1/endpoints \\
  -H "Authorization: Bearer <api_key>" \\
  -H "Content-Type: application/json" \\
  -d '{
    "name": "my-service",
    "url":  "https://my-service.example.com/webhooks"
  }'`}
        </pre>
      </div>
    </div>
  );
}

async function EndpointRow({ ep }: { ep: Endpoint }) {
  let subCount = 0;
  try {
    const subs = await getSubscriptions(ep.id);
    subCount = subs?.length ?? 0;
  } catch {}

  return (
    <tr className="row-hover" style={{ borderBottom: "1px solid var(--color-border)" }}>
      <td style={{ padding: "12px 16px" }}>
        <div style={{ fontSize: 13, fontWeight: 500 }}>{ep.name}</div>
        <div style={{ fontSize: 10, color: "var(--color-muted)", marginTop: 2 }}>
          {subCount} subscription{subCount !== 1 ? "s" : ""}
        </div>
      </td>
      <td style={{ padding: "12px 16px" }}>
        <div style={{ fontSize: 12, color: "var(--color-text)", maxWidth: 280, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
          {ep.url}
        </div>
      </td>
      <td style={{ padding: "12px 16px" }}>
        <StatusBadge status={ep.enabled ? "success" : "failed"} />
      </td>
      <td style={{ padding: "12px 16px", fontSize: 12, color: "var(--color-muted)" }}>
        {ep.timeout_ms / 1000}s
      </td>
      <td style={{ padding: "12px 16px", fontSize: 12, color: "var(--color-muted)" }}>
        {ep.max_retries}
      </td>
      <td style={{ padding: "12px 16px", fontSize: 11, color: "var(--color-muted)" }}>
        {fmtDate(ep.created_at)}
      </td>
    </tr>
  );
}
