import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import {
  CURRENCIES,
  deleted,
  EUR,
  JPY,
  validationFailed,
  versionConflict,
} from '@/testing/fixtures/settings';
import { EUR_RATES, NO_IMPORTS, NO_RATES, RECENT_IMPORTS_PAGE } from '@/testing/fixtures/rates';
import { looseReply } from '@/testing/replies';
import { CurrenciesPage } from './CurrenciesPage';

const USD_LOCK = "USD is MAPP's base currency; its rate is always 1.";

function renderCurrencies(answers: HttpHandler[] = [], route = '/settings/currencies') {
  const api = mockApi([
    looseReply('GET /api/v1/currencies', { body: CURRENCIES }),
    looseReply('GET /api/v1/exchange-rates/imports', { body: NO_IMPORTS }),
    looseReply('GET /api/v1/currencies/{code}/rates', { body: EUR_RATES }, { code: 'EUR' }),
    looseReply('GET /api/v1/currencies/{code}/rates', { body: NO_RATES }, { code: 'JPY' }),
    ...answers,
  ]);
  const rendered = renderWithProviders(<CurrenciesPage />, {
    route,
    user: testUser(),
    path: '/settings/currencies',
  });
  return { ...rendered, api };
}

function row(code: string): HTMLElement {
  return screen.getByRole('rowheader', { name: code }).closest('tr') as HTMLElement;
}

async function chooseAction(
  code: string,
  action: string,
  user: ReturnType<typeof renderCurrencies>['user'],
) {
  await user.click(await screen.findByRole('button', { name: `Actions for ${code}` }));
  await user.click(await screen.findByRole('menuitem', { name: new RegExp(`^${action}`) }));
}

describe('CurrenciesPage: the list (FR-002, FR-028)', () => {
  it('shows each active currency with its rate both ways, its use and when the rate changed', async () => {
    renderCurrencies();

    expect(screen.getByRole('heading', { level: 1, name: 'Currencies' })).toBeInTheDocument();
    expect(document.title).toBe('Currencies · MAPP');
    await screen.findByRole('table');

    const eur = row('EUR');
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
    const { user } = renderCurrencies([], '/settings/currencies?status=all');
    await screen.findByRole('table');

    await user.type(screen.getByRole('searchbox', { name: 'Search currencies' }), 'yen');

    expect(screen.getAllByRole('rowheader').map((cell) => cell.textContent)).toEqual(['JPY']);
    expect(within(row('JPY')).getByText('Inactive')).toBeInTheDocument();
  });

  it('says when the currencies cannot be loaded', async () => {
    renderCurrencies([
      looseReply('GET /api/v1/currencies', () => Promise.reject(new TypeError('offline'))),
    ]);

    expect(await screen.findByRole('alert')).toHaveTextContent("can't reach its server");
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });
});

describe('CurrenciesPage: details (US4 scenarios 2 and 4)', () => {
  it('shows the rate both ways, where it is used and who last changed the rate', async () => {
    const { user, router } = renderCurrencies();

    await user.click(await screen.findByRole('button', { name: 'EUR' }));

    const panel = await screen.findByRole('dialog', { name: 'Euro (EUR)' });
    expect(router.state.location.search).toBe('?id=EUR');
    expect(panel).toHaveTextContent('1 USD = 0.92 EUR');
    expect(panel).toHaveTextContent('1 EUR = 1.086957 USD');
    expect(panel).toHaveTextContent('Rate last changed by Daniel Okoro on 14 Mar 2026');
    expect(panel).toHaveTextContent('Used by 14 line items · Display currency of 2 people');
  });

  it("shows USD's locks on its rate and status, and explains a locked action", async () => {
    const { user, api } = renderCurrencies([], '/settings/currencies?id=USD');

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
    const { user, router } = renderCurrencies([], '/settings/currencies?id=GBP');

    const panel = await screen.findByRole('dialog');
    expect(panel).toHaveTextContent(
      'This currency no longer exists — someone may have deleted it.',
    );
    await user.click(within(panel).getByRole('button', { name: 'Back to Currencies' }));

    await waitFor(() => expect(router.state.location.search).toBe(''));
  });
});

describe('CurrenciesPage: create and edit (US4 scenarios 1, 3, 4, 5)', () => {
  it('adds a currency, showing its rate both ways while it is typed (scenario 1)', async () => {
    const { user, api } = renderCurrencies([
      looseReply('POST /api/v1/currencies', { status: 201, body: { ...JPY, active: true } }),
    ]);

    await user.click(screen.getByRole('button', { name: 'New currency' }));
    const form = await screen.findByRole('dialog', { name: 'New currency' });
    expect(within(form).getByRole('textbox', { name: /^Code/ })).toHaveAccessibleDescription(
      'Required Three letters, as in ISO 4217 (EUR, JPY).',
    );
    await user.type(within(form).getByRole('textbox', { name: /^Code/ }), 'JPY');
    await user.type(within(form).getByRole('textbox', { name: /^Name/ }), 'Japanese yen');
    await user.type(within(form).getByRole('textbox', { name: /^Symbol/ }), '¥');
    await user.type(within(form).getByRole('textbox', { name: /^Rate/ }), '150');

    expect(form).toHaveTextContent('1 USD = 150.00 JPY · 1 JPY = 0.006667 USD');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Currency 'Japanese yen' created");
    expect(api.bodiesOf('POST /api/v1/currencies')).toEqual([
      { code: 'JPY', name: 'Japanese yen', symbol: '¥', rate: '150' },
    ]);
  });

  it('shows every problem at once (scenario 5)', async () => {
    const { user } = renderCurrencies([
      looseReply('POST /api/v1/currencies', {
        status: 400,
        body: validationFailed([
          { field: 'code', message: 'Enter a three-letter code such as JPY.' },
          { field: 'name', message: "A currency called 'Euro' already exists." },
          { field: 'rate', message: 'Enter a rate above 0, with at most six decimals.' },
        ]),
      }),
    ]);

    await user.click(screen.getByRole('button', { name: 'New currency' }));
    const form = await screen.findByRole('dialog', { name: 'New currency' });
    await user.type(within(form).getByRole('textbox', { name: /^Code/ }), 'EURO');
    await user.type(within(form).getByRole('textbox', { name: /^Name/ }), 'Euro');
    await user.type(within(form).getByRole('textbox', { name: /^Rate/ }), '0');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await within(form).findByText('3 problems need your attention')).toBeInTheDocument();
    expect(within(form).getByRole('textbox', { name: /^Rate/ })).toHaveAccessibleDescription(
      /Enter a rate above 0, with at most six decimals\./,
    );
    expect(within(form).getByRole('textbox', { name: /^Code/ })).toHaveAttribute(
      'aria-invalid',
      'true',
    );
  });

  it('locks the code, and confirms a new rate with what it recalculates (scenario 3)', async () => {
    const warning =
      '14 line items in 6 activities use EUR. Their EUR amounts stay the same; their USD amounts will be recalculated (52,173.91 → 50,526.32 USD). Unassigned planned cost of these activities will change. No activity goes back for approval.';
    const { user, api } = renderCurrencies(
      [
        looseReply(
          'POST /api/v1/currencies/{code}/impact',
          {
            body: { warnings: [{ kind: 'amountsRecalculated', message: warning, count: 14 }] },
          },
          { code: 'EUR' },
        ),
        looseReply(
          'PATCH /api/v1/currencies/{code}',
          { body: { ...EUR, rate: '0.95', version: 5 } },
          { code: 'EUR' },
        ),
      ],
      '/settings/currencies?id=EUR',
    );

    const details = await screen.findByRole('dialog', { name: 'Euro (EUR)' });
    await user.click(within(details).getByRole('button', { name: 'Edit' }));
    const form = await screen.findByRole('dialog', { name: 'Edit currency' });
    expect(within(form).queryByRole('textbox', { name: /^Code/ })).not.toBeInTheDocument();
    expect(form).toHaveTextContent("The code can't change once the currency is saved.");
    expect(form).toHaveTextContent('Rate last changed by Daniel Okoro on 14 Mar 2026');

    const rate = within(form).getByRole('textbox', { name: /^Rate/ });
    await user.clear(rate);
    await user.type(rate, '0.95');
    expect(form).toHaveTextContent('1 USD = 0.95 EUR · 1 EUR = 1.052632 USD');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    const impact = await screen.findByRole('alertdialog', {
      name: "Save changes to currency 'Euro'?",
    });
    expect(impact).toHaveTextContent(warning);
    await user.click(within(impact).getByRole('button', { name: 'Save changes' }));

    expect(await screen.findAllByText("Currency 'Euro' saved")).not.toHaveLength(0);
    expect(api.bodiesOf('POST /api/v1/currencies/EUR/impact')).toEqual([
      { rate: '0.95', version: 4 },
    ]);
    expect(api.bodiesOf('PATCH /api/v1/currencies/EUR')).toEqual([{ rate: '0.95', version: 4 }]);
  });

  it("shows USD's rate read-only with its lock; its name can still change (scenario 4)", async () => {
    const { user, api } = renderCurrencies([
      looseReply(
        'PATCH /api/v1/currencies/{code}',
        {
          body: { ...CURRENCIES.items[0], name: 'United States dollar' },
        },
        { code: 'USD' },
      ),
    ]);

    await chooseAction('USD', 'Edit', user);
    const form = await screen.findByRole('dialog', { name: 'Edit currency' });
    expect(within(form).queryByRole('textbox', { name: /^Rate/ })).not.toBeInTheDocument();
    expect(form).toHaveTextContent('1 USD = 1.00 USD');
    expect(form).toHaveTextContent(USD_LOCK);

    const name = within(form).getByRole('textbox', { name: /^Name/ });
    await user.clear(name);
    await user.type(name, 'United States dollar');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Currency 'United States dollar' saved");
    expect(api.bodiesOf('PATCH /api/v1/currencies/USD')).toEqual([
      { name: 'United States dollar', version: 1 },
    ]);
  });

  it('keeps the typed values after a conflict and names who saved first (FR-007)', async () => {
    const { user } = renderCurrencies([
      looseReply(
        'PATCH /api/v1/currencies/{code}',
        { status: 409, body: versionConflict() },
        { code: 'EUR' },
      ),
      looseReply(
        'GET /api/v1/currencies/{code}',
        { body: { ...EUR, version: 6 } },
        { code: 'EUR' },
      ),
    ]);

    await chooseAction('EUR', 'Edit', user);
    const form = await screen.findByRole('dialog', { name: 'Edit currency' });
    await user.type(within(form).getByRole('textbox', { name: /^Symbol/ }), '!');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await within(form).findByRole('alert')).toHaveTextContent(
      'Not saved — Daniel Okoro changed this currency on 14 Mar 2026 at 10:42',
    );
    expect(within(form).getByRole('textbox', { name: /^Symbol/ })).toHaveValue('€!');
  });
});

describe('CurrenciesPage: status and delete (US4 scenario 2, FR-004)', () => {
  it('offers Set inactive instead of deleting a currency in use', async () => {
    const { user, api } = renderCurrencies([
      looseReply(
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
    const { user } = renderCurrencies(
      [
        looseReply(
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
    const { user, api } = renderCurrencies(
      [
        looseReply(
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
    renderCurrencies([
      looseReply('GET /api/v1/currencies', { body: DATED }),
      looseReply('GET /api/v1/exchange-rates/imports', { body: RECENT_IMPORTS_PAGE }),
    ]);
    await screen.findAllByRole('table');

    const eur = row('EUR');
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
    renderCurrencies(
      [looseReply('GET /api/v1/currencies', { body: DATED })],
      '/settings/currencies?id=EUR',
    );

    const panel = await screen.findByRole('dialog', { name: 'Euro (EUR)' });
    expect(panel).toHaveTextContent('Rate of 19 Oct 2026 · GSAP import');
    expect(within(panel).getByRole('heading', { name: 'Rate history' })).toBeInTheDocument();
    expect(await within(panel).findByText('From 01 Jan 2027')).toBeInTheDocument();
  });

  it('has no rate history for USD, whose rate is always 1 (FR-001)', async () => {
    renderCurrencies([], '/settings/currencies?id=USD');

    const panel = await screen.findByRole('dialog', { name: 'US dollar (USD)' });
    expect(within(panel).queryByRole('heading', { name: 'Rate history' })).not.toBeInTheDocument();
  });
});
