import { defineConfig, devices } from "@playwright/test";

const PORT = Number(process.env.E2E_PORT || 4173);
const baseURL = `http://127.0.0.1:${PORT}`;
const chromiumExecutablePath = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH;

// Viewports the design must stay responsive at. The mobile sizes reproduce the
// original radar horizontal-overflow bug.
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["github"], ["list"]] : "list",
  outputDir: "./e2e/.output",
  snapshotPathTemplate: "{testDir}/__screenshots__/{arg}{ext}",
  use: {
    baseURL,
    trace: "on-first-retry",
    launchOptions: chromiumExecutablePath ? { executablePath: chromiumExecutablePath } : undefined,
  },
  projects: [
    { name: "mobile-360", use: { browserName: "chromium", viewport: { width: 360, height: 740 } } },
    { name: "mobile-390", use: { ...devices["iPhone 13"], browserName: "chromium" } },
    { name: "tablet-768", use: { browserName: "chromium", viewport: { width: 768, height: 1024 } } },
    { name: "desktop-1280", use: { browserName: "chromium", viewport: { width: 1280, height: 900 } } },
  ],
  webServer: {
    command: "node e2e/serve.mjs",
    url: baseURL,
    reuseExistingServer: !process.env.CI,
    timeout: 30_000,
  },
});
