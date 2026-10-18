/// <reference types="vitest/config" />
import { fileURLToPath, URL } from 'node:url';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

// Where the development and preview servers forward /api requests. The end-to-end tests point it at
// their own API with NORTHSTAR_API_URL. 127.0.0.1 rather than "localhost": Node may resolve localhost
// to the IPv6 address, where the API does not listen.
const apiTarget = process.env.NORTHSTAR_API_URL ?? 'http://127.0.0.1:8080';

// Unit tests format dates and times, so they always run in the same time zone, whatever the laptop's.
if (process.env.VITEST) {
  process.env.TZ = 'UTC';
}

const proxy = { '/api': { target: apiTarget } };

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  // Servers listen on this computer only (no firewall prompt) and never move to another port silently.
  server: { host: '127.0.0.1', port: 5173, strictPort: true, proxy },
  preview: { host: '127.0.0.1', port: 4173, strictPort: true, proxy },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}'],
    setupFiles: ['src/test/setup.ts'],
  },
});
