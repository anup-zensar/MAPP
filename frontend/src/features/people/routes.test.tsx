import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectCurrentSection, renderApp } from '@/test/app';

// Feature 003's addresses (build guide §5). Feature 003 keeps this file up to date as it replaces
// the stub pages (their own requests go into renderApp's answers).
describe('People, teams and profile routes (feature 003)', () => {
  it.each([
    ['/settings/people', 'People'],
    ['/settings/people/U-1002', 'People'],
    ['/settings/teams', 'Teams'],
    ['/settings/teams/T-1001', 'Teams'],
  ])('opens %s: "%s", with Settings current in the sidebar', async (path, title) => {
    renderApp(path);

    expect(await screen.findByRole('heading', { level: 1, name: title })).toBeInTheDocument();
    expect(document.title).toBe(`${title} · NorthStar`);
    expectCurrentSection('Settings');
  });

  it('opens My profile, which is not in the sidebar (FR-012 of 001)', async () => {
    renderApp('/profile');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Your profile' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('My profile · NorthStar');
  });
});
