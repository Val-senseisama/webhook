"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const links = [
  { href: "/",            label: "Overview",   icon: "▦" },
  { href: "/endpoints",   label: "Endpoints",  icon: "◈" },
  { href: "/events",      label: "Events",     icon: "◎" },
  { href: "/deliveries",  label: "Deliveries", icon: "◷" },
];

export function Sidebar() {
  const path = usePathname();

  return (
    <aside
      style={{
        width: 220,
        minHeight: "100dvh",
        background: "var(--color-surface)",
        borderRight: "1px solid var(--color-border)",
        display: "flex",
        flexDirection: "column",
        flexShrink: 0,
      }}
    >
      {/* Wordmark */}
      <div
        style={{
          padding: "20px 20px 0",
          borderBottom: "1px solid var(--color-border)",
          paddingBottom: 20,
        }}
      >
        <div
          style={{
            fontSize: 11,
            letterSpacing: "0.15em",
            color: "var(--color-muted)",
            textTransform: "uppercase",
            marginBottom: 4,
          }}
        >
          System
        </div>
        <div
          style={{
            fontSize: 16,
            fontWeight: 600,
            color: "var(--color-text)",
            display: "flex",
            alignItems: "center",
            gap: 8,
          }}
        >
          <span style={{ color: "var(--color-accent)" }}>⬡</span>
          Webhook
        </div>
      </div>

      {/* Nav */}
      <nav style={{ padding: "12px 8px", flex: 1 }}>
        {links.map((l) => {
          const active = l.href === "/" ? path === "/" : path.startsWith(l.href);
          return (
            <Link
              key={l.href}
              href={l.href}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 10,
                padding: "8px 12px",
                marginBottom: 2,
                textDecoration: "none",
                fontSize: 13,
                borderRadius: 2,
                color: active ? "var(--color-text)" : "var(--color-muted)",
                background: active ? "var(--color-accent-dim)" : "transparent",
                borderLeft: active
                  ? "2px solid var(--color-accent)"
                  : "2px solid transparent",
                transition: "all 0.1s",
              }}
            >
              <span
                style={{
                  fontSize: 14,
                  color: active ? "var(--color-accent)" : "var(--color-muted)",
                }}
              >
                {l.icon}
              </span>
              {l.label}
            </Link>
          );
        })}
      </nav>

      {/* Footer */}
      <div
        style={{
          padding: "12px 20px",
          borderTop: "1px solid var(--color-border)",
          fontSize: 10,
          color: "var(--color-dim)",
          letterSpacing: "0.05em",
        }}
      >
        GO · SUPABASE · RIVER
      </div>
    </aside>
  );
}
