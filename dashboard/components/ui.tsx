"use client";

import { useState, type CSSProperties, type ReactNode } from "react";

export const btn = (variant: "solid" | "ghost" | "danger" = "ghost"): CSSProperties => ({
  fontFamily: "var(--font-mono)",
  fontSize: 11,
  letterSpacing: "0.04em",
  padding: "6px 12px",
  borderRadius: 2,
  cursor: "pointer",
  border: "1px solid var(--color-border-hi)",
  background:
    variant === "solid" ? "var(--color-accent)" : "transparent",
  color:
    variant === "solid"
      ? "#080808"
      : variant === "danger"
      ? "var(--color-error)"
      : "var(--color-text)",
  borderColor:
    variant === "solid"
      ? "var(--color-accent)"
      : variant === "danger"
      ? "rgba(239,68,68,0.4)"
      : "var(--color-border-hi)",
  transition: "opacity 0.15s",
});

export const input: CSSProperties = {
  fontFamily: "var(--font-mono)",
  fontSize: 12,
  padding: "8px 10px",
  background: "var(--color-bg)",
  border: "1px solid var(--color-border-hi)",
  borderRadius: 2,
  color: "var(--color-text)",
  width: "100%",
  outline: "none",
};

export const label: CSSProperties = {
  fontSize: 10,
  letterSpacing: "0.1em",
  textTransform: "uppercase",
  color: "var(--color-muted)",
  marginBottom: 6,
  display: "block",
};

/** One-time secret reveal. The secret is never retrievable after this. */
export function SecretReveal({
  secret,
  note,
  onDismiss,
}: {
  secret: string;
  note: string;
  onDismiss: () => void;
}) {
  const [copied, setCopied] = useState(false);
  return (
    <div
      style={{
        marginTop: 12,
        padding: "14px 16px",
        background: "var(--color-accent-dim)",
        border: "1px solid var(--color-accent)",
        borderRadius: 2,
      }}
    >
      <div style={{ fontSize: 11, color: "var(--color-accent)", marginBottom: 8 }}>
        {note} — copy it now, it cannot be shown again.
      </div>
      <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
        <code
          style={{
            flex: 1,
            fontSize: 12,
            color: "var(--color-text)",
            background: "var(--color-bg)",
            padding: "8px 10px",
            borderRadius: 2,
            overflowX: "auto",
            whiteSpace: "nowrap",
          }}
        >
          {secret}
        </code>
        <button
          type="button"
          style={btn("ghost")}
          onClick={() => {
            navigator.clipboard?.writeText(secret);
            setCopied(true);
          }}
        >
          {copied ? "copied" : "copy"}
        </button>
        <button type="button" style={btn("ghost")} onClick={onDismiss}>
          dismiss
        </button>
      </div>
    </div>
  );
}

export function ErrorText({ children }: { children: ReactNode }) {
  return (
    <div style={{ fontSize: 11, color: "var(--color-error)", marginTop: 8 }}>
      {children}
    </div>
  );
}
