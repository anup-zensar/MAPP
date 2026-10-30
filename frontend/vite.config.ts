/// <reference types="vitest/config" />
import { fileURLToPath, URL } from 'node:url';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';
import { developmentHeaders, securityHeaders } from './security-headers.ts';

// 127.0.0.1, not localhost: Node may resolve localhost to IPv6, where the API does not listen.
const apiTarget = process.env.MAPP_API_URL ?? 'http://127.0.0.1:8080';

// Unit tests format dates and times, so they always run in the same time zone, whatever the laptop's.
if (process.env.VITEST) {
  process.env.TZ = 'UTC';
}

// xfwd: the API counts sign-in attempts per client, so it must see the browser's address.
const proxy = { '/api': { target: apiTarget, xfwd: true } };

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: { host: '127.0.0.1', port: 5173, strictPort: true, proxy, headers: developmentHeaders },
  preview: { host: '127.0.0.1', port: 4173, strictPort: true, proxy, headers: securityHeaders },
  build: {
    rolldownOptions: {
      output: { chunkFileNames: (chunk) => `assets/${featureOf(chunk.facadeModuleId)}-[hash].js` },
    },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}', 'security-headers.test.ts'],
    setupFiles: ['src/testing/setup.ts'],
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

function featureOf(facadeModuleId: string | null): string {
  return /\/src\/features\/([^/]+)\/index\.ts$/.exec(facadeModuleId ?? '')?.[1] ?? '[name]';
}
