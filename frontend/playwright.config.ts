import { defineConfig, devices } from '@playwright/test';

// The end-to-end tests start their own API and web app on ports of their own, on a PostgreSQL database of
// their own (mapp_e2e, created when missing) that the seed empties and reloads, so a development server
// and its database are never touched (research R19). E2E_API_PORT, E2E_WEB_PORT and E2E_DATABASE_URL move
// them, so two checkouts can run their tests at once.
const databaseURL = process.env.E2E_DATABASE_URL || 'postgres://mapp@127.0.0.1:5432/mapp_e2e?sslmode=disable';
const apiPort = process.env.E2E_API_PORT || '18080';
const webPort = process.env.E2E_WEB_PORT || '14173';

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
    // Without fades, axe checks a dialog's final colours instead of a frame of its fade-in.
    reducedMotion: 'reduce',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: [
    {
      // Migrate the database and reload the demo data, then start the API on 127.0.0.1 only.
      command: 'go run ./cmd/seed -reset && go run ./cmd/api',
      cwd: '../backend',
      env: { ...process.env, DATABASE_URL: databaseURL, APP_PORT: apiPort, APP_ENV: 'local' },
      url: `http://127.0.0.1:${apiPort}/healthz`,
      timeout: 180_000,
      reuseExistingServer: false,
    },
    {
      // The production build, served on 127.0.0.1 with /api forwarded to the test API.
      command: `npm run build && npx vite preview --host 127.0.0.1 --port ${webPort} --strictPort`,
      env: { ...process.env, MAPP_API_URL: `http://127.0.0.1:${apiPort}` },
      url: `http://127.0.0.1:${webPort}`,
      timeout: 180_000,
      reuseExistingServer: false,
    },
  ],
});
