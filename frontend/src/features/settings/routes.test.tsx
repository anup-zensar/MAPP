import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectCurrentSection, renderApp } from '@/testing/app';
import type { MockHandler } from '@/testing/mockApi';

/** The requests the section pages make when they open, answered with empty lists. */
const PAGE_ANSWERS: Record<string, MockHandler> = {
  'GET /api/v1/settings/overview': {
    body: { models: 0, activityTypes: 0, shellAttributes: 0, currencies: 1, financeBudgetItems: 0 },
  },
  'GET /api/v1/models': { body: { items: [], total: 0 } },
  'GET /api/v1/activity-types': { body: { items: [], total: 0 } },
  'GET /api/v1/currencies': { body: { items: [], total: 0 } },
  'GET /api/v1/finance-budgets': { body: { items: [], total: 0, year: 2026, years: [2026] } },
  // The People, Teams and Approval workflows cards count their active records (003, 008).
  'GET /api/v1/people': { body: { items: [], total: 0 } },
  'GET /api/v1/teams': { body: { items: [], total: 0 } },
  'GET /api/v1/approval-workflows': { body: { items: [], total: 0 } },
  // The Roles and access card counts the roles (014).
  'GET /api/v1/roles': { body: { items: [], total: 0 } },
  // The Field help card counts the edited texts (016); Currencies lists the recent imports.
  'GET /api/v1/field-help': { body: { items: [], total: 0, shellAttributesWithoutHelp: 0 } },
  'GET /api/v1/exchange-rates/imports': { body: { items: [], total: 0 } },
};

// Feature 002's addresses (build guide §5). Feature 002 keeps this file up to date as it replaces
// the stub pages (their own requests go into renderApp's answers). The Shell attribute pages are
// in shell-attributes/routes.test.tsx.
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
    expect(document.title).toBe(`${title} · MAPP`);
    expectCurrentSection('Settings');
  });

  it('lists every Settings section, those of features 003 and 008 included (FR-001)', async () => {
    renderApp('/settings', PAGE_ANSWERS);

    const sections = await screen.findByRole('list', { name: 'Sections' });
    expect(
      within(sections)
        .getAllByRole('heading')
        .map((heading) => heading.textContent),
    ).toEqual([
      'Models',
      'Activity types',
      'Shell attributes',
      'Currencies',
      'Finance budget items',
      'Field help',
      'People',
      'Teams',
      'Approval workflows',
      'Roles and access',
    ]);
    expect(await within(sections).findByText('1 active')).toBeInTheDocument();
  });
});
