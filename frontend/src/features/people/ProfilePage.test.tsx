import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { card, EUR, renderProfile, USD } from '@/testing/fixtures/profile';
import { testUser } from '@/testing/fixtures/users';
import { reply } from '@/testing/replies';

describe('ProfilePage (feature 003, US4)', () => {
  it('shows the details with the e-mail lock, the preferences and My teams (US4-1, FR-021)', async () => {
    renderProfile();

    expect(screen.getByRole('heading', { level: 1, name: 'My profile' })).toBeInTheDocument();
    expect(document.title).toBe('My profile · MAPP');

    const details = card('Your details');
    expect(details).toHaveTextContent('Priya Raman');
    expect(details).toHaveTextContent('Marketer');
    expect(details).toHaveTextContent('priya.raman@shell.com');
    expect(
      within(details).getByText(
        'Your e-mail is your sign-in name; it is changed in Settings › People.',
      ),
    ).toBeInTheDocument();
    expect(within(details).getByRole('button', { name: 'Edit details' })).toBeInTheDocument();

    const preferences = card('Display preferences');
    expect(await within(preferences).findByText('USD — US dollar')).toBeInTheDocument();
    expect(within(preferences).getByText('12 Mar 2026 (DD MMM YYYY)')).toBeInTheDocument();
    expect(
      within(preferences).getByText(
        "Amounts are shown in your display currency with today's rate in Settings; amounts typed in other currencies stay as they are.",
      ),
    ).toBeInTheDocument();
    expect(
      within(preferences).getByRole('button', { name: 'Edit preferences' }),
    ).toBeInTheDocument();

    const teams = card('My teams');
    expect(within(teams).getByRole('link', { name: 'Lubricants Europe' })).toHaveAttribute(
      'href',
      '/settings/teams/T-1001',
    );
    expect(
      within(teams).getByRole('link', { name: 'Manage teams in Settings › Teams' }),
    ).toHaveAttribute('href', '/settings/teams');
  });

  it('keeps a display currency that was set inactive, labelled "(inactive)" (edge case, D-063)', async () => {
    renderProfile(
      [
        reply('GET /api/v1/me/display-currencies', {
          body: {
            items: [EUR, { code: 'GBP', name: 'Pound sterling', active: false }, USD],
            total: 3,
          },
        }),
      ],
      testUser({ preferences: { ...testUser().preferences, displayCurrency: 'GBP' } }),
    );

    expect(
      await within(card('Display preferences')).findByText('GBP — Pound sterling (inactive)'),
    ).toBeInTheDocument();
  });

  it('says how to join a team when the person is in none (US4-5)', () => {
    renderProfile([], testUser({ teams: [] }));

    const teams = card('My teams');
    expect(teams).toHaveTextContent(
      "You're not in any team yet, so you can't own a plan. Teams are managed in Settings › Teams.",
    );
    expect(within(teams).getByRole('link', { name: 'Settings › Teams' })).toHaveAttribute(
      'href',
      '/settings/teams',
    );
  });

  it('uses USD, 12 Mar 2026 and no teams when an older answer leaves them out (D-191)', async () => {
    renderProfile(
      [],
      testUser({
        version: undefined,
        teams: undefined,
        preferences: { theme: 'system', sidebarCollapsed: false, pinnedPlanIds: [] },
      }),
    );

    const preferences = card('Display preferences');
    expect(await within(preferences).findByText('USD — US dollar')).toBeInTheDocument();
    expect(within(preferences).getByText('12 Mar 2026 (DD MMM YYYY)')).toBeInTheDocument();
    expect(card('My teams')).toHaveTextContent("You're not in any team yet");
  });
});
