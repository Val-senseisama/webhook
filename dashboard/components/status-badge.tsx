const STATUS_CONFIG: Record<
  string,
  { color: string; bg: string; dot: string; label: string }
> = {
  success:       { color: "#22c55e", bg: "rgba(34,197,94,0.1)",   dot: "#22c55e", label: "success"      },
  delivered:     { color: "#22c55e", bg: "rgba(34,197,94,0.1)",   dot: "#22c55e", label: "delivered"     },
  received:      { color: "#60a5fa", bg: "rgba(96,165,250,0.1)",  dot: "#60a5fa", label: "received"      },
  processing:    { color: "#60a5fa", bg: "rgba(96,165,250,0.1)",  dot: "#60a5fa", label: "processing"    },
  pending:       { color: "#eab308", bg: "rgba(234,179,8,0.1)",   dot: "#eab308", label: "pending"       },
  in_flight:     { color: "#f97316", bg: "rgba(249,115,22,0.1)",  dot: "#f97316", label: "in-flight"     },
  failed:        { color: "#ef4444", bg: "rgba(239,68,68,0.1)",   dot: "#ef4444", label: "failed"        },
  dead_lettered: { color: "#9ca3af", bg: "rgba(156,163,175,0.1)", dot: "#9ca3af", label: "dead-lettered" },
};

export function StatusBadge({ status }: { status: string }) {
  const cfg = STATUS_CONFIG[status] ?? { color: "#808080", bg: "rgba(128,128,128,0.1)", dot: "#808080", label: status };
  const pulse = status === "in_flight" || status === "processing";
  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 5,
        padding: "2px 8px",
        borderRadius: 2,
        fontSize: 10,
        letterSpacing: "0.08em",
        textTransform: "uppercase",
        fontWeight: 600,
        color: cfg.color,
        background: cfg.bg,
      }}
    >
      <span
        className={pulse ? "pulse" : ""}
        style={{
          width: 5,
          height: 5,
          borderRadius: "50%",
          background: cfg.dot,
          display: "inline-block",
          flexShrink: 0,
        }}
      />
      {cfg.label}
    </span>
  );
}
