import type { Metadata } from "next";
import "./globals.css";
import { SkipNav } from "@/components/SkipNav";
import { DisclaimerBanner } from "@/components/DisclaimerBanner";
import { KeyboardHelp } from "@/components/KeyboardHelp";

export const metadata: Metadata = {
  title: "EvidenceLens",
  description: "Free, public, agentic biomedical evidence search",
  applicationName: "EvidenceLens",
  authors: [{ name: "EvidenceLens contributors" }],
  robots: { index: true, follow: true },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <SkipNav />
        <DisclaimerBanner />
        <main id="main">{children}</main>
        <KeyboardHelp />
      </body>
    </html>
  );
}
