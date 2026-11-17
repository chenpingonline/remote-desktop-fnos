import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "tests",
  workers: 1,
  timeout: 60000,
  use: {
    baseURL: "http://127.0.0.1:18080/app/remote-desktop/",
    browserName: "chromium",
    reducedMotion: "reduce",
    launchOptions: { timeout: 15000 },
    viewport: { width: 1280, height: 800 },
  },
  reporter: "list",
});
