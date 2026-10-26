import { readdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import js from '@eslint/js';
import { defineConfig, globalIgnores } from 'eslint/config';
import importX, { createNodeResolver } from 'eslint-plugin-import-x';
import jsxA11y from 'eslint-plugin-jsx-a11y-x';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import globals from 'globals';
import tseslint from 'typescript-eslint';

const SRC = fileURLToPath(new URL('./src', import.meta.url));
const FEATURES = readdirSync(`${SRC}/features`);
const LOWER_LAYERS = ['ui', 'api', 'session', 'format'].map((dir) => `./src/${dir}`);
const TESTS = ['src/**/*.test.{ts,tsx}', 'src/testing/**'];
// Features whose code still has type assertions (026 aligns them one by one; then this goes).
const ALIGNED = ['sign-in', 'people', 'notifications', 'recycle-bin', 'placeholders'];
const NOT_ALIGNED_YET = FEATURES.filter((feature) => !ALIGNED.includes(feature));
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
      jsxA11y.configs.recommended,
    ],
    languageOptions: {
      ecmaVersion: 2023,
      globals: globals.browser,
    },
    rules: {
      // Safari drops the list role of a list without bullets, so lists say role="list".
      'jsx-a11y-x/no-redundant-roles': ['error', { ul: ['list'], ol: ['list'] }],
      // A region or group that scrolls takes the focus, so the keyboard can scroll it (WCAG 2.1.1).
      'jsx-a11y-x/no-noninteractive-tabindex': ['error', { roles: ['region', 'group'] }],
      'jsx-a11y-x/label-has-associated-control': ['error', { depth: 3 }],
    },
  },
  {
    // F5: no `as T` and no `!`; feature folders join as they are aligned.
    files: ['src/**/*.{ts,tsx}', 'e2e/**/*.ts'],
    ignores: NOT_ALIGNED_YET.map((feature) => `src/features/${feature}/**`),
    rules: {
      '@typescript-eslint/no-non-null-assertion': 'error',
      '@typescript-eslint/consistent-type-assertions': ['error', { assertionStyle: 'never' }],
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
    // Fast refresh never loads test code.
    files: TESTS,
    rules: { 'react-refresh/only-export-components': 'off' },
  },
  {
    files: ['vite.config.ts', 'playwright.config.ts', 'e2e/**/*.ts'],
    languageOptions: { globals: globals.node },
  },
]);
