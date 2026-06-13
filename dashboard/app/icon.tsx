import { ImageResponse } from "next/og";

export const size = { width: 32, height: 32 };
export const contentType = "image/png";

export default function Icon() {
  return new ImageResponse(
    (
      <div
        style={{
          width: 32,
          height: 32,
          background: "#080808",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        {/* Hexagon shape via clip-path */}
        <div
          style={{
            width: 22,
            height: 22,
            background: "#f97316",
            clipPath:
              "polygon(50% 0%, 93% 25%, 93% 75%, 50% 100%, 7% 75%, 7% 25%)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
          }}
        >
          {/* Inner dark hex cutout for ring effect */}
          <div
            style={{
              width: 13,
              height: 13,
              background: "#080808",
              clipPath:
                "polygon(50% 0%, 93% 25%, 93% 75%, 50% 100%, 7% 75%, 7% 25%)",
            }}
          />
        </div>
      </div>
    ),
    { ...size }
  );
}
