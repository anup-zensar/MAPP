import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectCurrentSection, renderApp } from '@/test/app';

// Feature 002's addresses (build guide §5). Feature 002 keeps this file up to date as it replaces
// the stub pages (their own requests go into renderApp's answers).
describe('Settings routes (feature 002)', () => {
  it.each([
    ['/settings', 'Settings'],
    ['/settings/models', 'Models'],
    ['/settings/activity-types', 'Activity types'],
    ['/settings/shell-attributes', 'Shell attributes'],
    ['/settings/shell-attributes/new', 'New Shell attribute'],
    ['/settings/shell-attributes/SA-1001', 'Shell attribute'],
    ['/settings/shell-attributes/SA-1001/edit', 'Edit Shell attribute'],
    ['/settings/currencies', 'Currencies'],
    ['/settings/finance-budget-items', 'Finance budget items'],
  ])('opens %s: "%s", with Settings current in the sidebar', async (path, title) => {
    renderApp(path);

    expect(await screen.findByRole('heading', { level: 1, name: title })).toBeInTheDocument();
    expect(document.title).toBe(`${title} · NorthStar`);
    expectCurrentSection('Settings');
  });

  it('lists every Settings section, those of features 003 and 008 included (FR-001)', async () => {
    renderApp('/settings');

    const sections = await screen.findByRole('region', { name: 'Sections' });
    expect(
      within(sections)
        .getAllByRole('link')
        .map((link) => link.textContent),
    ).toEqual([
      'Models',
      'Activity types',
      'Shell attributes',
      'Currencies',
      'Finance budget items',
      'People',
      'Teams',
      'Approval workflows',
    ]);
  });
});
