import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import { CURRENCIES, deleted, EUR, JPY } from '@/testing/fixtures/settings';
import { RECENT_IMPORTS_PAGE } from '@/testing/fixtures/rates';
import { chooseAction, renderCurrenciesPage, rowOf } from '@/testing/fixtures/settings-pages';
import { reply } from '@/testing/replies';

const USD_LOCK = "USD is MAPP's base currency; its rate is always 1.";

describe('CurrenciesPage: the list (FR-002, FR-028)', () => {
  it('shows each active currency with its rate both ways, its use and when the rate changed', async () => {
    renderCurrenciesPage();

    expect(screen.getByRole('heading', { level: 1, name: 'Currencies' })).toBeInTheDocument();
    expect(document.title).toBe('Currencies · MAPP');
    await screen.findByRole('table');

    const eur = rowOf('EUR');
    expect(eur).toHaveTextContent('Euro');
    expect(eur).toHaveTextContent('€');
    expect(eur).toHaveTextContent('1 USD = 0.92 EUR1 EUR = 1.086957 USD');
    expect(eur).toHaveTextContent('Used by 14 line items · Display currency of 2 people');
    expect(eur).toHaveTextContent('14 Mar 2026');
    expect(within(eur).getByText('Active')).toBeInTheDocument();
    // JPY is inactive.
    expect(screen.queryByRole('rowheader', { name: 'JPY' })).not.toBeInTheDocument();
    expect(screen.getByText('2 of 3 currencies')).toBeInTheDocument();
  });

  it('searches code or name', async () => {
    const { user } = renderCurrenciesPage([], '/settings/currencies?status=all');
    await screen.findByRole('table');

    await user.type(screen.getByRole('searchbox', { name: 'Search currencies' }), 'yen');

    expect(screen.getAllByRole('rowheader').map((cell) => cell.textContent)).toEqual(['JPY']);
    expect(within(rowOf('JPY')).getByText('Inactive')).toBeInTheDocument();
  });

  it('says when the currencies cannot be loaded', async () => {
    renderCurrenciesPage([
      reply('GET /api/v1/currencies', () => Promise.reject(new TypeError('offline'))),
    ]);

    expect(await screen.findByRole('alert')).toHaveTextContent("can't reach its server");
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });
});

describe('CurrenciesPage: details (US4 scenarios 2 and 4)', () => {
  it('shows the rate both ways, where it is used and who last changed the rate', async () => {
    const { user, router } = renderCurrenciesPage();

    await user.click(await screen.findByRole('button', { name: 'EUR' }));

    const panel = await screen.findByRole('dialog', { name: 'Euro (EUR)' });
    expect(router.state.location.search).toBe('?id=EUR');
    expect(panel).toHaveTextContent('1 USD = 0.92 EUR');
    expect(panel).toHaveTextContent('1 EUR = 1.086957 USD');
    expect(panel).toHaveTextContent('Rate last changed by Daniel Okoro on 14 Mar 2026');
    expect(panel).toHaveTextContent('Used by 14 line items · Display currency of 2 people');
  });

  it("shows USD's locks on its rate and status, and explains a locked action", async () => {
    const { user, api } = renderCurrenciesPage([], '/settings/currencies?id=USD');

    const panel = await screen.findByRole('dialog', { name: 'US dollar (USD)' });
    expect(within(panel).getAllByText(USD_LOCK)).toHaveLength(2);

    await user.click(within(panel).getByRole('button', { name: 'Actions for USD' }));
    const setInactive = await screen.findByRole('menuitem', { name: /^Set inactive/ });
    expect(setInactive).toHaveTextContent(USD_LOCK);
    await user.click(setInactive);

    expect(await screen.findAllByText(USD_LOCK)).not.toHaveLength(0);
    expect(api.requests.some((request) => request.method === 'POST')).toBe(false);
  });

  it('says when the currency of the address no longer exists', async () => {
    const { user, router } = renderCurrenciesPage([], '/settings/currencies?id=GBP');

    const panel = await screen.findByRole('dialog');
    expect(panel).toHaveTextContent(
      'This currency no longer exists — someone may have deleted it.',
    );
    await user.click(within(panel).getByRole('button', { name: 'Back to Currencies' }));

    await waitFor(() => expect(router.state.location.search).toBe(''));
  });
});

describe('CurrenciesPage: status and delete (US4 scenario 2, FR-004)', () => {
  it('offers Set inactive instead of deleting a currency in use', async () => {
    const { user, api } = renderCurrenciesPage([
      reply(
        'POST /api/v1/currencies/{code}/status',
        { body: { ...EUR, active: false, version: 5 } },
        { code: 'EUR' },
      ),
    ]);

    await chooseAction('EUR', 'Delete', user);
    const dialog = await screen.findByRole('alertdialog', { name: "'Euro' is in use" });
    expect(dialog).toHaveTextContent(
      'Used by 14 line items · Display currency of 2 people — set it inactive instead.',
    );
    await user.click(within(dialog).getByRole('button', { name: 'Set inactive' }));

    expect(await findMessages()).toHaveTextContent("Currency 'Euro' set inactive");
    expect(api.bodiesOf('POST /api/v1/currencies/EUR/status')).toEqual([
      { active: false, version: 4 },
    ]);
  });

  it('moves an unused currency to the recycle bin and offers Undo', async () => {
    const { user } = renderCurrenciesPage(
      [
        reply(
          'DELETE /api/v1/currencies/{code}',
          { body: deleted('currency', 'JPY', 'Japanese yen') },
          { code: 'JPY' },
        ),
      ],
      '/settings/currencies?status=inactive',
    );

    await chooseAction('JPY', 'Delete', user);
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete currency 'Japanese yen'?",
    });
    await user.click(within(dialog).getByRole('button', { name: 'Delete currency' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Currency 'Japanese yen' moved to the recycle bin");
    expect(within(messages).getByRole('button', { name: 'Undo' })).toBeInTheDocument();
  });

  it('reactivates an inactive currency', async () => {
    const { user, api } = renderCurrenciesPage(
      [
        reply(
          'POST /api/v1/currencies/{code}/status',
          { body: { ...JPY, active: true, version: 2 } },
          { code: 'JPY' },
        ),
      ],
      '/settings/currencies?status=inactive',
    );

    await chooseAction('JPY', 'Reactivate', user);

    expect(await findMessages()).toHaveTextContent("Currency 'Japanese yen' reactivated");
    expect(api.bodiesOf('POST /api/v1/currencies/JPY/status')).toEqual([
      { active: true, version: 1 },
    ]);
  });
});

describe('CurrenciesPage: dated rates and the GSAP import (016 US1)', () => {
  const EUR_DATED = {
    ...EUR,
    rateDate: '2026-10-19',
    rateSource: 'gsapImport' as const,
    rateImportFileName: 'gsap_rates_2026-10-19.csv',
  };
  const DATED = {
    ...CURRENCIES,
    items: CURRENCIES.items.map((c) => (c.code === 'EUR' ? EUR_DATED : c)),
  };

  it('shows the date and source of each current rate, Import GSAP rates and Recent imports', async () => {
    renderCurrenciesPage([
      reply('GET /api/v1/currencies', { body: DATED }),
      reply('GET /api/v1/exchange-rates/imports', { body: RECENT_IMPORTS_PAGE }),
    ]);
    await screen.findAllByRole('table');

    const eur = rowOf('EUR');
    expect(eur).toHaveTextContent('19 Oct 2026');
    expect(eur).toHaveTextContent('GSAP import');
    expect(screen.getByRole('link', { name: 'Import GSAP rates' })).toHaveAttribute(
      'href',
      '/settings/currencies/import',
    );
    // Import GSAP rates is a secondary link; New currency stays the one primary button.
    expect(screen.getByRole('button', { name: 'New currency' })).toBeInTheDocument();
    expect(
      await screen.findByRole('rowheader', { name: 'gsap_rates_2026-10-20.csv' }),
    ).toBeInTheDocument();
  });

  it("shows the current rate's date and source and the rate history in the details (US1-1)", async () => {
    renderCurrenciesPage(
      [reply('GET /api/v1/currencies', { body: DATED })],
      '/settings/currencies?id=EUR',
    );

    const panel = await screen.findByRole('dialog', { name: 'Euro (EUR)' });
    expect(panel).toHaveTextContent('Rate of 19 Oct 2026 · GSAP import');
    expect(within(panel).getByRole('heading', { name: 'Rate history' })).toBeInTheDocument();
    expect(await within(panel).findByText('From 01 Jan 2027')).toBeInTheDocument();
  });

  it('has no rate history for USD, whose rate is always 1 (FR-001)', async () => {
    renderCurrenciesPage([], '/settings/currencies?id=USD');

    const panel = await screen.findByRole('dialog', { name: 'US dollar (USD)' });
    expect(within(panel).queryByRole('heading', { name: 'Rate history' })).not.toBeInTheDocument();
  });
});
