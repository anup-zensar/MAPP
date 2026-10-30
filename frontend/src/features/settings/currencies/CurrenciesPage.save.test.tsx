import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import { EUR, JPY, USD, validationFailed, versionConflict } from '@/testing/fixtures/settings';
import { chooseAction, renderCurrenciesPage } from '@/testing/fixtures/settings-pages';
import { reply } from '@/testing/replies';

const USD_LOCK = "USD is SP&A One's base currency; its rate is always 1.";

describe('CurrenciesPage: create and edit (US4 scenarios 1, 3, 4, 5)', () => {
  it('adds a currency, showing its rate both ways while it is typed (scenario 1)', async () => {
    const { user, api } = renderCurrenciesPage([
      reply('POST /api/v1/currencies', { status: 201, body: { ...JPY, active: true } }),
    ]);

    await user.click(screen.getByRole('button', { name: 'New currency' }));
    const form = await screen.findByRole('dialog', { name: 'New currency' });
    expect(within(form).getByRole('textbox', { name: /^Code/ })).toHaveAccessibleDescription(
      'required Three letters, as in ISO 4217 (EUR, JPY).',
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
    const { user } = renderCurrenciesPage([
      reply('POST /api/v1/currencies', {
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
    const { user, api } = renderCurrenciesPage(
      [
        reply(
          'POST /api/v1/currencies/{code}/impact',
          {
            body: { warnings: [{ kind: 'amountsRecalculated', message: warning, count: 14 }] },
          },
          { code: 'EUR' },
        ),
        reply(
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
    const { user, api } = renderCurrenciesPage([
      reply(
        'PATCH /api/v1/currencies/{code}',
        {
          body: { ...USD, name: 'United States dollar' },
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
    await user.paste('United States dollar');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Currency 'United States dollar' saved");
    expect(api.bodiesOf('PATCH /api/v1/currencies/USD')).toEqual([
      { name: 'United States dollar', version: 1 },
    ]);
  });

  it('keeps the typed values after a conflict and names who saved first (FR-007)', async () => {
    const { user } = renderCurrenciesPage([
      reply(
        'PATCH /api/v1/currencies/{code}',
        { status: 409, body: versionConflict() },
        { code: 'EUR' },
      ),
      reply('GET /api/v1/currencies/{code}', { body: { ...EUR, version: 6 } }, { code: 'EUR' }),
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
