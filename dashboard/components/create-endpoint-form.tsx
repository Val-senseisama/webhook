"use client";

import { useState, useTransition } from "react";
import { createEndpointAction } from "@/app/endpoints/actions";
import { SecretReveal, ErrorText, btn, input, label } from "@/components/ui";

export function CreateEndpointForm() {
  const [open, setOpen] = useState(false);
  const [secret, setSecret] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, start] = useTransition();

  function onSubmit(formData: FormData) {
    setError(null);
    const name = String(formData.get("name") ?? "").trim();
    const url = String(formData.get("url") ?? "").trim();
    const timeout_ms = Number(formData.get("timeout_ms")) || undefined;
    const max_retries = Number(formData.get("max_retries")) || undefined;
    if (!name || !url) {
      setError("name and url are required");
      return;
    }
    start(async () => {
      const res = await createEndpointAction({ name, url, timeout_ms, max_retries });
      if (res.ok) {
        setSecret(res.secret);
        setOpen(false);
      } else {
        setError(res.error);
      }
    });
  }

  return (
    <div style={{ marginBottom: 24 }}>
      <div style={{ display: "flex", justifyContent: "flex-end" }}>
        <button
          type="button"
          style={btn("solid")}
          onClick={() => {
            setOpen((v) => !v);
            setError(null);
          }}
        >
          {open ? "cancel" : "+ new endpoint"}
        </button>
      </div>

      {open && (
        <form
          action={onSubmit}
          style={{
            marginTop: 12,
            background: "var(--color-surface)",
            border: "1px solid var(--color-border)",
            padding: "20px 24px",
          }}
        >
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "1fr 1fr",
              gap: 16,
            }}
          >
            <div style={{ gridColumn: "1 / -1" }}>
              <label style={label}>Name</label>
              <input name="name" style={input} placeholder="my-service" autoFocus />
            </div>
            <div style={{ gridColumn: "1 / -1" }}>
              <label style={label}>URL</label>
              <input name="url" style={input} placeholder="https://my-service.example.com/webhooks" />
            </div>
            <div>
              <label style={label}>Timeout (ms)</label>
              <input name="timeout_ms" style={input} defaultValue={30000} inputMode="numeric" />
            </div>
            <div>
              <label style={label}>Max retries</label>
              <input name="max_retries" style={input} defaultValue={5} inputMode="numeric" />
            </div>
          </div>

          {error && <ErrorText>{error}</ErrorText>}

          <div style={{ marginTop: 16, display: "flex", justifyContent: "flex-end" }}>
            <button type="submit" style={btn("solid")} disabled={pending}>
              {pending ? "creating…" : "create endpoint"}
            </button>
          </div>
        </form>
      )}

      {secret && (
        <SecretReveal
          secret={secret}
          note="Endpoint signing secret"
          onDismiss={() => setSecret(null)}
        />
      )}
    </div>
  );
}
