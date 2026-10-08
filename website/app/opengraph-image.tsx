import { ImageResponse } from "next/og";

/**
 * Generates /opengraph-image at build time. Next reuses it for
 * `twitter:image` too.
 */

export const alt = "Vnytros — open-source, self-hosted DNS & email-authentication toolkit";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default function OpengraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          justifyContent: "space-between",
          background: "#08090b",
          padding: "72px 80px",
          position: "relative",
        }}
      >
        {/* Accent bloom, matching the hero backdrop. */}
        <div
          style={{
            position: "absolute",
            top: -260,
            right: -160,
            width: 720,
            height: 720,
            borderRadius: 9999,
            background:
              "radial-gradient(circle, rgba(139,124,255,0.30) 0%, rgba(139,124,255,0) 70%)",
            display: "flex",
          }}
        />

        <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
          <div
            style={{
              width: 44,
              height: 44,
              borderRadius: 12,
              background: "linear-gradient(135deg, #8b7cff 0%, #5fd4ff 100%)",
              display: "flex",
            }}
          />
          <div
            style={{
              fontSize: 30,
              color: "#f7f8fa",
              letterSpacing: "-0.02em",
              fontWeight: 600,
            }}
          >
            vnytros
          </div>
        </div>

        <div style={{ display: "flex", flexDirection: "column", gap: 26 }}>
          <div
            style={{
              fontSize: 70,
              lineHeight: 1.04,
              color: "#f7f8fa",
              letterSpacing: "-0.04em",
              fontWeight: 600,
              maxWidth: 940,
              display: "flex",
            }}
          >
            Self-hosted DNS & email-authentication toolkit.
          </div>
          <div
            style={{
              fontSize: 29,
              lineHeight: 1.4,
              color: "#b4bac6",
              letterSpacing: "-0.01em",
              maxWidth: 880,
              display: "flex",
            }}
          >
            Provider detection, DNS records on Route 53 and Hostinger, domain
            verification, and 15 DNS, email, TLS and HTTP checks via REST API
            and MCP.
          </div>
        </div>

        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 14,
            fontSize: 23,
            color: "#7e8593",
            letterSpacing: "-0.01em",
          }}
        >
          <div style={{ display: "flex" }}>github.com/Aadesh998/vnytros-domain-connect</div>
          <div style={{ display: "flex", color: "#4e5563" }}>·</div>
          <div style={{ display: "flex" }}>open source · AGPL-3.0</div>
        </div>
      </div>
    ),
    size,
  );
}
