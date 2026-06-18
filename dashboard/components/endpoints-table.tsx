"use client";

import { useState, useTransition } from "react";
import type { Endpoint, Subscription } from "@/lib/types";
import { StatusBadge } from "@/components/status-badge";
import { SecretReveal, ErrorText, btn, input, label } from "@/components/ui";
import { fmtDate } from "@/lib/fmt";
import {
  setEnabledAction,
  deleteEndpointAction,
  rotateSecretAction,
  addSubscriptionAction,
  deleteSubscriptionAction,
} from "@/app/endpoints/actions";

const COLS = ["Name", "URL", "Status", "Timeout", "Retries", "Created", ""];

export function EndpointsTable({
  endpoints,
  subs,
}: {
  endpoints: Endpoint[];
  subs: Record<string, Subscription[]>;
}) {
  const [expanded, setExpanded] = useState<string | null>(null);

  return (
    <div
      style={{
        background: "var(--color-surface)",
        border: "1px solid var(--color-border)",
        overflow: "hidden",
      }}
    >
      <table style={{ width: "100%", borderCollapse: "collapse" }}>
        <thead>
          <tr style={{ borderBottom: "1px solid var(--color-border)", background: "var(--color-bg)" }}>
            {COLS.map((h, i) => (
              <th
                key={i}
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
          {endpoints.length === 0 && (
            <tr>
              <td colSpan={COLS.length} style={{ padding: "40px 16px", textAlign: "center", color: "var(--color-muted)", fontSize: 12 }}>
                No endpoints yet — create one above.
              </td>
            </tr>
          )}
          {endpoints.map((ep) => (
            <EndpointRows
              key={ep.id}
              ep={ep}
              subs={subs[ep.id] ?? []}
              open={expanded === ep.id}
              onToggle={() => setExpanded((cur) => (cur === ep.id ? null : ep.id))}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function EndpointRows({
  ep,
  subs,
  open,
  onToggle,
}: {
  ep: Endpoint;
  subs: Subscription[];
  open: boolean;
  onToggle: () => void;
}) {
  return (
    <>
      <tr className="row-hover" style={{ borderBottom: open ? "none" : "1px solid var(--color-border)" }}>
        <td style={{ padding: "12px 16px" }}>
          <div style={{ fontSize: 13, fontWeight: 500 }}>{ep.name}</div>
          <div style={{ fontSize: 10, color: "var(--color-muted)", marginTop: 2 }}>
            {subs.length} subscription{subs.length !== 1 ? "s" : ""}
          </div>
        </td>
        <td style={{ padding: "12px 16px" }}>
          <div style={{ fontSize: 12, maxWidth: 260, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
            {ep.url}
          </div>
        </td>
        <td style={{ padding: "12px 16px" }}>
          <StatusBadge status={ep.enabled ? "success" : "failed"} />
        </td>
        <td style={{ padding: "12px 16px", fontSize: 12, color: "var(--color-muted)" }}>{ep.timeout_ms / 1000}s</td>
        <td style={{ padding: "12px 16px", fontSize: 12, color: "var(--color-muted)" }}>{ep.max_retries}</td>
        <td style={{ padding: "12px 16px", fontSize: 11, color: "var(--color-muted)" }}>{fmtDate(ep.created_at)}</td>
        <td style={{ padding: "12px 16px", textAlign: "right" }}>
          <button type="button" style={btn("ghost")} onClick={onToggle}>
            {open ? "close" : "manage"}
          </button>
        </td>
      </tr>
      {open && (
        <tr style={{ borderBottom: "1px solid var(--color-border)", background: "var(--color-bg)" }}>
          <td colSpan={COLS.length} style={{ padding: "20px 24px" }}>
            <EndpointDetail ep={ep} subs={subs} />
          </td>
        </tr>
      )}
    </>
  );
}

function EndpointDetail({ ep, subs }: { ep: Endpoint; subs: Subscription[] }) {
  const [pending, start] = useTransition();
  const [error, setError] = useState<string | null>(null);
  const [secret, setSecret] = useState<string | null>(null);

  function run(fn: () => Promise<{ ok: boolean; error?: string }>) {
    setError(null);
    start(async () => {
      const res = await fn();
      if (!res.ok) setError(res.error ?? "request failed");
    });
  }

  function onAddSub(formData: FormData) {
    const raw = String(formData.get("event_types") ?? "").trim();
    const types = raw
      ? raw.split(",").map((s) => s.trim()).filter(Boolean)
      : ["*"];
    run(() => addSubscriptionAction(ep.id, types));
  }

  return (
    <div style={{ display: "grid", gap: 24, gridTemplateColumns: "1fr 1fr" }}>
      {/* Subscriptions */}
      <div>
        <div style={label}>Subscriptions</div>
        <div style={{ display: "flex", flexWrap: "wrap", gap: 6, marginBottom: 12 }}>
          {subs.length === 0 && (
            <span style={{ fontSize: 11, color: "var(--color-muted)" }}>
              none — endpoint receives nothing until subscribed
            </span>
          )}
          {subs.map((s) => (
            <span
              key={s.id}
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 6,
                fontSize: 11,
                padding: "3px 8px",
                background: "var(--color-surface-hi)",
                border: "1px solid var(--color-border-hi)",
                borderRadius: 2,
              }}
            >
              {s.event_types.join(", ")}
              <button
                type="button"
                onClick={() => run(() => deleteSubscriptionAction(ep.id, s.id))}
                style={{
                  border: "none",
                  background: "none",
                  color: "var(--color-muted)",
                  cursor: "pointer",
                  fontSize: 13,
                  lineHeight: 1,
                  padding: 0,
                }}
                aria-label="remove subscription"
              >
                ×
              </button>
            </span>
          ))}
        </div>
        <form action={onAddSub} style={{ display: "flex", gap: 8 }}>
          <input
            name="event_types"
            style={input}
            placeholder="payment.succeeded, order.shipped  (blank = * all)"
          />
          <button type="submit" style={btn("ghost")} disabled={pending}>
            add
          </button>
        </form>
      </div>

      {/* Actions */}
      <div>
        <div style={label}>Actions</div>
        <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
          <button
            type="button"
            style={btn("ghost")}
            disabled={pending}
            onClick={() => run(() => setEnabledAction(ep, !ep.enabled))}
          >
            {ep.enabled ? "disable" : "enable"}
          </button>
          <button
            type="button"
            style={btn("ghost")}
            disabled={pending}
            onClick={() =>
              run(async () => {
                const res = await rotateSecretAction(ep.id);
                if (res.ok) setSecret(res.secret);
                return res;
              })
            }
          >
            rotate secret
          </button>
          <button
            type="button"
            style={btn("danger")}
            disabled={pending}
            onClick={() => {
              if (confirm(`Delete endpoint "${ep.name}"? This cannot be undone.`)) {
                run(() => deleteEndpointAction(ep.id));
              }
            }}
          >
            delete
          </button>
        </div>
        {error && <ErrorText>{error}</ErrorText>}
        {secret && (
          <SecretReveal
            secret={secret}
            note="New signing secret"
            onDismiss={() => setSecret(null)}
          />
        )}
      </div>
    </div>
  );
}
