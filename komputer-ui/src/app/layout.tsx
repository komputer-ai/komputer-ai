import type { Metadata } from "next";
import { DM_Sans, JetBrains_Mono } from "next/font/google";
import { AppShell } from "@/components/layout/app-shell";
import "./globals.css";

const dmSans = DM_Sans({
  subsets: ["latin"],
  variable: "--font-sans",
});

const jetbrainsMono = JetBrains_Mono({
  subsets: ["latin"],
  variable: "--font-mono",
});

export const metadata: Metadata = {
  description: "Distributed Claude agent platform",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={`${dmSans.variable} ${jetbrainsMono.variable} dark h-full antialiased`} suppressHydrationWarning>
      <head>
        <script src="/config.js" />
      </head>
      <body className="h-full flex font-[family-name:var(--font-sans)]">
        <AppShell>{children}</AppShell>
      </body>
    </html>
  );
}
