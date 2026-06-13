import type { Delivery, DeliveryAttempt, Endpoint, Event, Subscription } from "./types";

const BASE = process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
const API_KEY = process.env.WEBHOOK_API_KEY ?? process.env.NEXT_PUBLIC_API_KEY ?? "";

function headers(): HeadersInit {
  return {
    "Content-Type": "application/json",
    Authorization: `Bearer ${API_KEY}`,
  };
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: { ...headers(), ...(init?.headers ?? {}) },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status}: ${text}`);
  }
  return res.json() as Promise<T>;
}

// ---- Events ----
export const getEvents = (params?: { type?: string; source?: string; limit?: number }) =>
  req<Event[]>(`/v1/events?${new URLSearchParams(params as Record<string, string> ?? {})}`);

export const getEvent = (id: string) => req<Event>(`/v1/events/${id}`);

export const redeliverEvent = (id: string) =>
  req<{ status: string }>(`/v1/events/${id}/redeliver`, { method: "POST" });

// ---- Endpoints ----
export const getEndpoints = () => req<Endpoint[]>("/v1/endpoints");

export const getEndpoint = (id: string) => req<Endpoint>(`/v1/endpoints/${id}`);

export const createEndpoint = (body: {
  name: string;
  url: string;
  timeout_ms?: number;
  max_retries?: number;
}) => req<Endpoint>("/v1/endpoints", { method: "POST", body: JSON.stringify(body) });

export const updateEndpoint = (
  id: string,
  body: Partial<Pick<Endpoint, "name" | "url" | "enabled" | "timeout_ms" | "max_retries">>
) => req<Endpoint>(`/v1/endpoints/${id}`, { method: "PATCH", body: JSON.stringify(body) });

export const deleteEndpoint = (id: string) =>
  fetch(`${BASE}/v1/endpoints/${id}`, { method: "DELETE", headers: headers() });

export const rotateSecret = (id: string) =>
  req<{ secret: string }>(`/v1/endpoints/${id}/rotate-secret`, { method: "POST" });

// ---- Subscriptions ----
export const getSubscriptions = (endpointID: string) =>
  req<Subscription[]>(`/v1/endpoints/${endpointID}/subscriptions`);

export const createSubscription = (endpointID: string, eventTypes: string[]) =>
  req<Subscription>(`/v1/endpoints/${endpointID}/subscriptions`, {
    method: "POST",
    body: JSON.stringify({ event_types: eventTypes }),
  });

export const deleteSubscription = (endpointID: string, subID: string) =>
  fetch(`${BASE}/v1/endpoints/${endpointID}/subscriptions/${subID}`, {
    method: "DELETE",
    headers: headers(),
  });

// ---- Deliveries ----
export const getDeliveries = (params?: {
  endpoint_id?: string;
  event_id?: string;
  status?: string;
  limit?: number;
}) => req<Delivery[]>(`/v1/deliveries?${new URLSearchParams(params as Record<string, string> ?? {})}`);

export const getDelivery = (id: string) => req<Delivery>(`/v1/deliveries/${id}`);

export const getAttempts = (deliveryID: string) =>
  req<DeliveryAttempt[]>(`/v1/deliveries/${deliveryID}/attempts`);

export const retryDelivery = (id: string) =>
  req<{ status: string }>(`/v1/deliveries/${id}/retry`, { method: "POST" });
