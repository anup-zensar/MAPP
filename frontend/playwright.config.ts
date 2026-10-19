import os from 'node:os';
import path from 'node:path';
import { defineConfig, devices } from '@playwright/test';

// The end-to-end tests start their own API and web app on ports of their own, with freshly seeded demo
// data in a temporary folder, so a development server and backend/data are never touched (research R19).
const dataDir = path.join(os.tmpdir(), 'northstar-e2e');
const apiPort = '18080';
const webPort = '14173';

export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.spec.ts',
  // One test at a time: the tests share one API and its demo people.
  workers: 1,
  fullyParallel: false,
  retries: 0,
  reporter: 'list',
  timeout: 60_000,
  expect: { timeout: 10_000 },
  use: {
    baseURL: `http://127.0.0.1:${webPort}`,
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: [
    {
      // Load the demo data into an empty temporary folder, then start the API on 127.0.0.1 only.
      command: 'go run ./cmd/seed -reset && go run ./cmd/api',
      cwd: '../backend',
      env: { ...process.env, DATA_DIR: dataDir, APP_PORT: apiPort, APP_ENV: 'local' },
      url: `http://127.0.0.1:${apiPort}/api/health`,
      timeout: 180_000,
      reuseExistingServer: false,
    },
    {
      // The production build, served on 127.0.0.1 with /api forwarded to the test API.
      command: `npm run build && npx vite preview --host 127.0.0.1 --port ${webPort} --strictPort`,
      env: { ...process.env, NORTHSTAR_API_URL: `http://127.0.0.1:${apiPort}` },
      url: `http://127.0.0.1:${webPort}`,
      timeout: 180_000,
      reuseExistingServer: false,
    },
  ],
});
