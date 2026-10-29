/// <reference types="vitest/config" />
import { fileURLToPath, URL } from 'node:url';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';
import { developmentHeaders, securityHeaders } from './security-headers.ts';

// Where the development and preview servers forward /api requests. The end-to-end tests point it at
// their own API with MAPP_API_URL. 127.0.0.1 rather than "localhost": Node may resolve localhost
// to the IPv6 address, where the API does not listen.
const apiTarget = process.env.MAPP_API_URL ?? 'http://127.0.0.1:8080';

// Unit tests format dates and times, so they always run in the same time zone, whatever the laptop's.
if (process.env.VITEST) {
  process.env.TZ = 'UTC';
}

// xfwd: the API counts sign-in attempts per client, so it must see the browser's address (038 FR-014).
const proxy = { '/api': { target: apiTarget, xfwd: true } };

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  // Servers listen on this computer only (no firewall prompt) and never move to another port silently.
  server: { host: '127.0.0.1', port: 5173, strictPort: true, proxy, headers: developmentHeaders },
  preview: { host: '127.0.0.1', port: 4173, strictPort: true, proxy, headers: securityHeaders },
  build: {
    rolldownOptions: {
      // Each feature's code is one file named after the feature (app/pages.tsx loads them).
      output: { chunkFileNames: (chunk) => `assets/${featureOf(chunk.facadeModuleId)}-[hash].js` },
    },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}', 'security-headers.test.ts'],
    setupFiles: ['src/testing/setup.ts'],
    // Generous, because a busy laptop can run a single test for several seconds.
    testTimeout: 15000,
    coverage: {
      provider: 'v8',
      reporter: ['text-summary', 'lcov'],
      reportsDirectory: 'coverage',
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/**/*.test.{ts,tsx}', 'src/testing/**', 'src/api/generated/**'],
    },
  },
});

/** The feature a chunk starts from (`src/features/<name>/index.ts`), or Rolldown's own name. */
function featureOf(facadeModuleId: string | null): string {
  return /\/src\/features\/([^/]+)\/index\.ts$/.exec(facadeModuleId ?? '')?.[1] ?? '[name]';
}
