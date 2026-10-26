import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { SESSION_KEY } from '@/app/useSession';
import type { CurrentUser } from '@/app/useSession';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { findMessages, renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { CurrencyPicker } from './CurrencyPicker';
import { DISPLAY_CURRENCIES } from './testData';

const CURRENCY_LIST = { body: { items: DISPLAY_CURRENCIES, total: DISPLAY_CURRENCIES.length } };

function renderPicker(answers: Record<string, MockHandler> = {}, user: CurrentUser = testUser()) {
  const api = mockApi({ 'GET /api/v1/me/display-currencies': CURRENCY_LIST, ...answers });
  const rendered = renderWithProviders(<CurrencyPicker />, { user });
  return { ...rendered, api };
}

function picker() {
  return screen.getByRole('combobox', { name: 'Currency' });
}

function optionTexts() {
  return within(picker())
    .getAllByRole('option')
    .map((option) => option.textContent);
}

describe('CurrencyPicker', () => {
  it('offers the currencies My profile offers and starts on the display currency (FR-002, US1 scenario 1, D-217)', async () => {
    renderPicker();

    await waitFor(() =>
      expect(optionTexts()).toEqual(['EUR — Euro', 'GBP — Pound sterling', 'USD — US dollar']),
    );
    expect(picker()).toHaveValue('USD');
  });

  it('keeps an inactive display currency in the list, because it stays in use (D-201)', async () => {
    renderPicker(
      {
        'GET /api/v1/me/display-currencies': {
          body: {
            items: [{ code: 'CNY', name: 'Chinese yuan', active: false }, ...DISPLAY_CURRENCIES],
            total: 4,
          },
        },
      },
      testUser({ preferences: { ...testUser().preferences, displayCurrency: 'CNY' } }),
    );

    await waitFor(() => expect(optionTexts()).toContain('CNY — Chinese yuan'));
    expect(picker()).toHaveValue('CNY');
  });

  it('shows the display currency while the list is still loading', () => {
    renderPicker({ 'GET /api/v1/me/display-currencies': () => new Promise<never>(() => {}) });

    expect(optionTexts()).toEqual(['USD']);
    expect(picker()).toHaveValue('USD');
  });

  it('saves the chosen currency as display currency and asks every card again (FR-002, SC-005)', async () => {
    const { user, api, queryClient } = renderPicker({
      'PATCH /api/v1/me/preferences': {
        body: { ...testUser().preferences, displayCurrency: 'EUR' },
      },
    });
    queryClient.setQueryData(['home', 'summary', 'myTeams'], { scope: 'myTeams' });
    await waitFor(() => expect(optionTexts()).toHaveLength(3));

    await user.selectOptions(picker(), 'EUR');

    await waitFor(() =>
      expect(queryClient.getQueryData<CurrentUser>(SESSION_KEY)?.preferences.displayCurrency).toBe(
        'EUR',
      ),
    );
    expect(api.bodiesOf('PATCH /api/v1/me/preferences')).toEqual([{ displayCurrency: 'EUR' }]);
    expect(queryClient.getQueryState(['home', 'summary', 'myTeams'])?.isInvalidated).toBe(true);
    expect(picker()).toHaveValue('EUR');
  });

  it('says when the choice could not be saved, goes back, and saves on Try again (FR-002)', async () => {
    let attempts = 0;
    const { user, api } = renderPicker({
      'PATCH /api/v1/me/preferences': () =>
        ++attempts === 1
          ? { status: 500 }
          : { body: { ...testUser().preferences, displayCurrency: 'EUR' } },
    });
    await waitFor(() => expect(optionTexts()).toHaveLength(3));

    await user.selectOptions(picker(), 'EUR');

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Your choice couldn't be saved.");
    expect(picker()).toHaveValue('USD');

    await user.click(within(messages).getByRole('button', { name: 'Try again' }));

    await waitFor(() => expect(picker()).toHaveValue('EUR'));
    expect(api.bodiesOf('PATCH /api/v1/me/preferences')).toEqual([
      { displayCurrency: 'EUR' },
      { displayCurrency: 'EUR' },
    ]);
  });
});
