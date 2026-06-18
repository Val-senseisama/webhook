import type { Metadata } from "next";
import { Geist_Mono } from "next/font/google";
import "./globals.css";
import { Sidebar } from "@/components/sidebar";

const geistMono = Geist_Mono({ subsets: ["latin"], variable: "--font-mono" });

export const metadata: Metadata = {
  title: "Webhook System",
  description: "Production webhook delivery platform",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className={geistMono.variable}>
      <body>
        <Sidebar />
        <main
          style={{
            flex: 1,
            overflowY: "auto",
            padding: "32px",
            maxWidth: "calc(100vw - 220px)",
          }}
        >
          {children}
        </main>
      </body>
    </html>
  );
}
