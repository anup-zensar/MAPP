import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { CurrentUser } from '@/session/useSession';
import { validation } from '@/testing/fixtures/people';
import { card, CURRENCIES, renderProfile } from '@/testing/fixtures/profile';
import { testUser } from '@/testing/fixtures/users';
import { reply } from '@/testing/replies';

describe('ProfilePage (feature 003, US4): Edit display preferences', () => {
  it('saves the display currency and date format together; dates follow at once and every query is asked again (US4-3, US4-4, FR-023, FR-024, SC-006)', async () => {
    const preferences: CurrentUser['preferences'] = {
      theme: 'system',
      sidebarCollapsed: false,
      displayCurrency: 'EUR',
      dateFormat: 'DD/MM/YYYY',
      pinnedPlanIds: [],
    };
    const { api, user, queryClient } = renderProfile([
      reply('PATCH /api/v1/me/preferences', { body: preferences }),
      reply('GET /api/v1/me', { body: testUser({ preferences }) }),
    ]);
    // A list of another page, kept in the cache: its amounts must come again, converted.
    queryClient.setQueryData(['plans'], { items: [], total: 0 });
    await within(card('Display preferences')).findByText('USD — US dollar');

    await user.click(screen.getByRole('button', { name: 'Edit preferences' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit display preferences' });
    const currency = await within(panel).findByRole('combobox', { name: 'Display currency' });
    expect(currency).toHaveDisplayValue('USD — US dollar');
    const dateFormat = within(panel).getByRole('combobox', { name: 'Date format' });
    expect(
      within(dateFormat)
        .getAllByRole('option')
        .map((option) => option.textContent),
    ).toEqual([
      'Choose…',
      '12 Mar 2026 (DD MMM YYYY)',
      '12/03/2026 (DD/MM/YYYY)',
      '03/12/2026 (MM/DD/YYYY)',
      '2026-03-12 (YYYY-MM-DD)',
    ]);
    await user.selectOptions(currency, 'EUR — Euro');
    await user.selectOptions(dateFormat, '12/03/2026 (DD/MM/YYYY)');
    const currencyAsks = api.bodiesOf('GET /api/v1/me/display-currencies').length;
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Your preferences saved')).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/v1/me/preferences')).toEqual([
      { displayCurrency: 'EUR', dateFormat: 'DD/MM/YYYY' },
    ]);
    const shown = card('Display preferences');
    expect(within(shown).getByText('EUR — Euro')).toBeInTheDocument();
    expect(within(shown).getByText('12/03/2026 (DD/MM/YYYY)')).toBeInTheDocument();
    expect(screen.getByText('Account: Priya Raman (PR) · a date: 12/03/2026')).toBeInTheDocument();
    expect(queryClient.getQueryState(['plans'])?.isInvalidated).toBe(true);
    await waitFor(() =>
      expect(api.bodiesOf('GET /api/v1/me/display-currencies')).toHaveLength(currencyAsks + 1),
    );
  });

  it('shows every problem of the preferences at its field at once (SC-007)', async () => {
    const { user } = renderProfile([
      reply(
        'PATCH /api/v1/me/preferences',
        validation([
          { field: 'displayCurrency', message: 'Choose one of the active currencies.' },
          { field: 'dateFormat', message: 'Choose one of the date formats.' },
        ]),
      ),
    ]);

    await user.click(screen.getByRole('button', { name: 'Edit preferences' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit display preferences' });
    await within(panel).findByRole('combobox', { name: 'Display currency' });
    await user.selectOptions(within(panel).getByRole('combobox', { name: 'Date format' }), '');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findAllByText('Choose one of the date formats.')).toHaveLength(2);
    expect(within(panel).getAllByText('Choose one of the active currencies.')).toHaveLength(2);
    expect(screen.queryByText('Your preferences saved')).not.toBeInTheDocument();
  });

  it('shows a grey field while the currencies load, and Try again when they fail (plan "States")', async () => {
    let failing = true;
    const { user } = renderProfile([
      reply('GET /api/v1/me/display-currencies', () =>
        failing
          ? {
              status: 500,
              body: {
                error: {
                  code: 'internal_error',
                  message: 'Something went wrong. Please try again.',
                  fields: [],
                },
              },
            }
          : { body: CURRENCIES },
      ),
    ]);
    // Without the currencies' names, the card shows the code from the session.
    expect(card('Display preferences')).toHaveTextContent('USD');

    await user.click(screen.getByRole('button', { name: 'Edit preferences' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit display preferences' });
    expect(
      await within(panel).findByText('Something went wrong. Please try again.'),
    ).toBeInTheDocument();
    expect(within(panel).getByRole('combobox', { name: 'Date format' })).toBeInTheDocument();

    failing = false;
    await user.click(within(panel).getByRole('button', { name: 'Try again' }));
    expect(
      await within(panel).findByRole('combobox', { name: 'Display currency' }),
    ).toHaveDisplayValue('USD — US dollar');
  });
});
