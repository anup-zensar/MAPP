import { readdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import js from '@eslint/js';
import { defineConfig, globalIgnores } from 'eslint/config';
import importX, { createNodeResolver } from 'eslint-plugin-import-x';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import globals from 'globals';
import tseslint from 'typescript-eslint';

const SRC = fileURLToPath(new URL('./src', import.meta.url));
const FEATURES = readdirSync(`${SRC}/features`);
const LOWER_LAYERS = ['ui', 'api', 'session', 'format'].map((dir) => `./src/${dir}`);
const TESTS = ['src/**/*.test.{ts,tsx}', 'src/testing/**'];
const TEST_ONLY = {
  group: ['msw', 'msw/*', 'openapi-msw', '@/testing', '@/testing/*'],
  message: 'Test-only code never reaches the application.',
};
const API_CLIENT = {
  name: '@/api/client',
  message: 'Use a query hook from an api.ts module (F2).',
};

// Constitution F2: a feature is reached only through its index.ts, from app/ and other features.
const featureZones = FEATURES.map((feature) => ({
  target: ['./src/app', ...FEATURES.filter((f) => f !== feature).map((f) => `./src/features/${f}`)],
  from: `./src/features/${feature}`,
  except: ['./index.ts'],
  message: `Import this feature through '@/features/${feature}' (its index.ts).`,
}));

export default defineConfig([
  // Generated or built files are never linted.
  globalIgnores(['dist', 'coverage', 'test-results', 'playwright-report', 'src/api/generated']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 2023,
      globals: globals.browser,
    },
  },
  {
    files: ['src/**/*.{ts,tsx}'],
    plugins: { 'import-x': importX },
    settings: {
      'import-x/extensions': ['.ts', '.tsx'],
      'import-x/parsers': { '@typescript-eslint/parser': ['.ts', '.tsx'] },
      'import-x/resolver-next': [
        createNodeResolver({ alias: { '@': [SRC] }, extensions: ['.ts', '.tsx', '.d.ts'] }),
      ],
    },
    rules: {
      'import-x/no-cycle': ['error', { ignoreExternal: true }],
      'import-x/order': [
        'error',
        {
          groups: ['builtin', 'external', 'internal', ['parent', 'sibling', 'index']],
          pathGroups: [{ pattern: '@/**', group: 'internal' }],
          'newlines-between': 'never',
        },
      ],
      'import-x/no-restricted-paths': [
        'error',
        {
          zones: [
            {
              target: LOWER_LAYERS,
              from: ['./src/features', './src/app'],
              message: 'ui, api, session and format must not depend on features or app (F2).',
            },
            {
              target: './src/features',
              from: './src/app',
              message: 'Features must not import app (F2).',
            },
            ...featureZones,
          ],
        },
      ],
    },
  },
  {
    files: ['src/**/*.ts'],
    ignores: TESTS,
    rules: { 'no-restricted-imports': ['error', { patterns: [TEST_ONLY] }] },
  },
  {
    // Components draw and handle events; requests go through the query hooks of an api module.
    files: ['src/**/*.tsx'],
    ignores: TESTS,
    rules: { 'no-restricted-imports': ['error', { paths: [API_CLIENT], patterns: [TEST_ONLY] }] },
  },
  {
    files: ['src/**/*.{ts,tsx}'],
    ignores: [...TESTS, 'src/api/**'],
    rules: {
      'no-restricted-globals': [
        'error',
        { name: 'fetch', message: 'Only src/api/ calls fetch (F2).' },
      ],
    },
  },
  {
    files: ['vite.config.ts', 'playwright.config.ts', 'e2e/**/*.ts'],
    languageOptions: { globals: globals.node },
  },
]);
