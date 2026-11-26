import { defineConfig } from "@playwright/test";
import { env } from "node:process";

export default defineConfig({
  testDir: "./e2e",
  use: { baseURL: env.TAOTALE_E2E_URL ?? "http://127.0.0.1:18080" },
  projects: [
    { name: "desktop", use: { viewport: { width: 1440, height: 960 }, colorScheme: "light" } },
    { name: "tablet", use: { viewport: { width: 768, height: 1024 }, colorScheme: "dark" } },
  ],
});
