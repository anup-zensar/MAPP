import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { expectCurrentSection, renderApp } from '@/testing/app';
import { reply } from '@/testing/replies';

const PAGE_ANSWERS: HttpHandler[] = [
  reply('GET /api/v1/settings/overview', {
    body: { models: 0, activityTypes: 0, shellAttributes: 0, currencies: 1, financeBudgetItems: 0 },
  }),
  reply('GET /api/v1/models', { body: { items: [], total: 0 } }),
  reply('GET /api/v1/activity-types', { body: { items: [], total: 0 } }),
  reply('GET /api/v1/currencies', { body: { items: [], total: 0 } }),
  reply('GET /api/v1/finance-budgets', {
    body: { items: [], total: 0, year: 2026, years: [2026] },
  }),
  reply('GET /api/v1/people', { body: { items: [], total: 0 } }),
  reply('GET /api/v1/teams', { body: { items: [], total: 0 } }),
  reply('GET /api/v1/approval-workflows', { body: { items: [], total: 0 } }),
  reply('GET /api/v1/roles', { body: { items: [], total: 0 } }),
  reply('GET /api/v1/field-help', { body: { items: [], total: 0, shellAttributesWithoutHelp: 0 } }),
  reply('GET /api/v1/exchange-rates/imports', { body: { items: [], total: 0 } }),
];

// The Shell attribute pages are in shell-attributes/routes.test.tsx.
describe('Settings routes (feature 002)', () => {
  it.each([
    ['/settings', 'Settings'],
    ['/settings/models', 'Models'],
    ['/settings/activity-types', 'Activity types'],
    ['/settings/currencies', 'Currencies'],
    ['/settings/finance-budget-items', 'Finance budget items'],
  ])('opens %s: "%s", with Settings current in the sidebar', async (path, title) => {
    renderApp(path, PAGE_ANSWERS);

    expect(await screen.findByRole('heading', { level: 1, name: title })).toBeInTheDocument();
    expect(document.title).toBe(`${title} · SP&A One`);
    expectCurrentSection('Settings');
  });
});
