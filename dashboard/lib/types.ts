export interface Event {
  id: string;
  tenant_id: string;
  source: string;
  type: string;
  payload: Record<string, unknown>;
  headers: Record<string, string>;
  idempotency_key: string;
  status: "received" | "processing" | "delivered" | "failed" | "dead_lettered";
  created_at: string;
}

export interface Endpoint {
  id: string;
  tenant_id: string;
  name: string;
  url: string;
  secret: string;
  enabled: boolean;
  timeout_ms: number;
  max_retries: number;
  created_at: string;
  updated_at: string;
}

export interface Subscription {
  id: string;
  endpoint_id: string;
  event_types: string[];
  filter_rules: Record<string, unknown> | null;
  created_at: string;
}

export interface Delivery {
  id: string;
  event_id: string;
  endpoint_id: string;
  status: "pending" | "in_flight" | "success" | "failed" | "dead_lettered";
  attempt_count: number;
  next_attempt_at: string;
  last_attempt_at: string | null;
  created_at: string;
}

export interface DeliveryAttempt {
  id: string;
  delivery_id: string;
  attempt_number: number;
  request_headers: Record<string, string>;
  request_body: string;
  response_status: number | null;
  response_headers: Record<string, string> | null;
  response_body: string | null;
  duration_ms: number | null;
  error: string | null;
  created_at: string;
}

export interface APIKey {
  id: string;
  tenant_id: string;
  name: string;
  created_at: string;
}
