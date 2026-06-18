import { getEndpoints, getSubscriptions } from "@/lib/api";
import { CreateEndpointForm } from "@/components/create-endpoint-form";
import { EndpointsTable } from "@/components/endpoints-table";
import type { Endpoint, Subscription } from "@/lib/types";

export const dynamic = "force-dynamic";

export default async function EndpointsPage() {
  let endpoints: Endpoint[] = [];
  try {
    endpoints = (await getEndpoints()) ?? [];
  } catch {
    // API not reachable yet
  }

  // Fetch each endpoint's subscriptions server-side (keeps the API key off the client).
  const subs: Record<string, Subscription[]> = {};
  await Promise.all(
    endpoints.map(async (ep) => {
      try {
        subs[ep.id] = (await getSubscriptions(ep.id)) ?? [];
      } catch {
        subs[ep.id] = [];
      }
    })
  );

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

      <CreateEndpointForm />

      <EndpointsTable endpoints={endpoints} subs={subs} />
    </div>
  );
}
