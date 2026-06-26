import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { AI } from "./ai";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "WealthStream OS",
  description: "Autonomous Wealth Management Operating System",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className={inter.className}>
        <AI>{children}</AI>
      </body>
    </html>
  );
}
