import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { expectNoAxeViolations } from '@/testing/axe';
import { nth } from '@/testing/dom';
import { deleted, validationFailed, versionConflict } from '@/testing/fixtures/settings';
import { EUR_CURRENT, EUR_FUTURE, EUR_ONLY_CURRENT, EUR_RATES } from '@/testing/fixtures/rates';
import { reply } from '@/testing/replies';
import { RateHistorySection } from './RateHistorySection';

const NO_IMPACT = { body: { warnings: [] } };

// RateDialog reads today from the clock; the rate of 01 Jan 2027 must stay a future one.
beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'], now: new Date('2026-10-20T12:00:00Z') });
});

afterEach(() => {
  vi.useRealTimers();
});

function renderHistory(answers: HttpHandler[] = []) {
  const api = mockApi([
    reply('GET /api/v1/currencies/{code}/rates', { body: EUR_RATES }, { code: 'EUR' }),
    ...answers,
  ]);
  const rendered = renderWithProviders(<RateHistorySection code="EUR" />, { user: testUser() });
  return { ...rendered, api };
}

function rows(): HTMLElement[] {
  return within(screen.getByRole('table')).getAllByRole('row').slice(1);
}

async function chooseAction(
  rateLabel: string,
  action: string,
  user: ReturnType<typeof renderHistory>['user'],
) {
  await user.click(await screen.findByRole('button', { name: `Actions for ${rateLabel}` }));
  await user.click(await screen.findByRole('menuitem', { name: new RegExp(`^${action}`) }));
}

describe('RateHistorySection (016 FR-001, FR-003)', () => {
  it('lists the rates newest first: "From" on a future rate, "Current" on the one in use', async () => {
    renderHistory();

    expect(screen.getByRole('heading', { level: 3, name: 'Rate history' })).toBeInTheDocument();
    await screen.findByRole('table');
    const listed = rows();
    const future = nth(listed, 0);
    const current = nth(listed, 1);
    const older = nth(listed, 2);
    expect(future).toHaveTextContent('From 01 Jan 2027');
    expect(future).toHaveTextContent('0.951 EUR = 1.052632 USD');
    expect(future).toHaveTextContent('Entered by hand');
    expect(future).toHaveTextContent('Daniel Okoro');
    expect(within(future).queryByText('Current')).not.toBeInTheDocument();
    expect(current).toHaveTextContent('19 Oct 2026');
    expect(within(current).getByText('Current')).toBeInTheDocument();
    expect(current).toHaveTextContent('GSAP import');
    expect(current).toHaveTextContent('gsap_rates_2026-10-19.csv');
    expect(older).toHaveTextContent('18 Oct 2026');
    expect(older).toHaveTextContent('0.9214');
    // The rate is a number: right-aligned under its header.
    const header = screen.getByRole('columnheader', { name: 'Rate per 1 USD' });
    expect(nth(within(current).getAllByRole('cell'), 0).className).toBe(header.className);
  });

  it('moves a rate to the recycle bin with Undo', async () => {
    const { user, api } = renderHistory([
      reply(
        'DELETE /api/v1/exchange-rates/{rateId}',
        {
          body: deleted('Exchange rate', 'XR-1101', 'EUR, 18 Oct 2026'),
        },
        { rateId: 'XR-1101' },
      ),
    ]);
    await screen.findByRole('table');

    await chooseAction('the rate of 18 Oct 2026', 'Delete', user);

    const messages = await findMessages();
    expect(
      await within(messages).findByText(
        "Exchange rate 'EUR, 18 Oct 2026' deleted",
      ),
    ).toBeInTheDocument();
    expect(within(messages).getByRole('button', { name: 'Undo' })).toBeInTheDocument();
    expect(api.requests.some((r) => r.method === 'DELETE')).toBe(true);
  });

  it('refuses to delete the last rate dated today or earlier, and says why (US1-5)', async () => {
    const { user, api } = renderHistory([
      reply('GET /api/v1/currencies/{code}/rates', { body: EUR_ONLY_CURRENT }, { code: 'EUR' }),
    ]);
    await screen.findByRole('table');

    await user.click(screen.getByRole('button', { name: 'Actions for the rate of 19 Oct 2026' }));
    const item = await screen.findByRole('menuitem', { name: /^Delete/ });
    expect(item).toHaveTextContent('EUR needs a rate dated today or earlier.');
    await user.click(item);

    expect(
      await within(await findMessages()).findByText('EUR needs a rate dated today or earlier.'),
    ).toBeInTheDocument();
    expect(api.requests.some((r) => r.method === 'DELETE')).toBe(false);
  });

  it('has no axe violations (WCAG 2.2 AA)', async () => {
    renderHistory();
    await screen.findByRole('table');

    await expectNoAxeViolations();
  });
});

describe('RateDialog: Add rate (016 FR-003, FR-005)', () => {
  it('adds a rate dated today, with the rate both ways while it is typed', async () => {
    const { user, api } = renderHistory([
      reply('POST /api/v1/currencies/{code}/rates/impact', NO_IMPACT, { code: 'EUR' }),
      reply(
        'POST /api/v1/currencies/{code}/rates',
        {
          status: 201,
          body: { ...EUR_CURRENT, id: 'XR-1104', date: '2026-10-20', rate: '0.93' },
        },
        { code: 'EUR' },
      ),
    ]);
    await screen.findByRole('table');

    await user.click(screen.getByRole('button', { name: 'Add rate' }));
    const dialog = await screen.findByRole('dialog', { name: 'Add rate of EUR' });
    expect(within(dialog).getByLabelText(/^Date/)).toHaveValue('2026-10-20');
    await user.type(within(dialog).getByRole('textbox', { name: /^Rate/ }), '0.93');
    expect(dialog).toHaveTextContent('1 USD = 0.93 EUR · 1 EUR = 1.075269 USD');
    await user.click(within(dialog).getByRole('button', { name: 'Save rate' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(api.bodiesOf('POST /api/v1/currencies/EUR/rates')).toEqual([
      { date: '2026-10-20', rate: '0.93' },
    ]);
    expect(
      await within(await findMessages()).findByText('Rate of EUR on 20 Oct 2026 saved'),
    ).toBeInTheDocument();
  });

  it('says from when a future rate counts', async () => {
    const { user } = renderHistory();
    await screen.findByRole('table');

    await user.click(screen.getByRole('button', { name: 'Add rate' }));
    const dialog = await screen.findByRole('dialog', { name: 'Add rate of EUR' });
    fireEvent.change(within(dialog).getByLabelText(/^Date/), { target: { value: '2099-01-01' } });

    expect(dialog).toHaveTextContent('No conversion changes before 01 Jan 2099.');
  });

  it('states what a new current rate recalculates; Keep editing saves nothing (US1-4)', async () => {
    const impact = 'Recalculates the USD values of 14 line items in EUR.';
    const plans =
      '2 plans are planned in EUR. Their EUR amounts stay the same; their USD values change (1,304,347.83 → 1,263,157.89 USD).';
    const { user, api } = renderHistory([
      reply(
        'POST /api/v1/currencies/{code}/rates/impact',
        {
          body: {
            warnings: [
              { kind: 'amountsRecalculated', message: impact, count: 14 },
              { kind: 'amountsRecalculated', message: plans, count: 2 },
            ],
          },
        },
        { code: 'EUR' },
      ),
      reply(
        'POST /api/v1/currencies/{code}/rates',
        { status: 201, body: { ...EUR_CURRENT, rate: '0.95' } },
        { code: 'EUR' },
      ),
    ]);
    await screen.findByRole('table');

    await user.click(screen.getByRole('button', { name: 'Add rate' }));
    const dialog = await screen.findByRole('dialog', { name: 'Add rate of EUR' });
    await user.type(within(dialog).getByRole('textbox', { name: /^Rate/ }), '0.95');
    await user.click(within(dialog).getByRole('button', { name: 'Save rate' }));

    const question = await screen.findByRole('alertdialog', {
      name: 'Save the rate of EUR on 20 Oct 2026?',
    });
    expect(question).toHaveTextContent(impact);
    expect(question).toHaveTextContent(plans);
    await user.click(within(question).getByRole('button', { name: 'Keep editing' }));
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/currencies/EUR/rates')).toEqual([]);

    await user.click(within(dialog).getByRole('button', { name: 'Save rate' }));
    await user.click(
      within(await screen.findByRole('alertdialog')).getByRole('button', { name: 'Save rate' }),
    );
    await waitFor(() => expect(api.bodiesOf('POST /api/v1/currencies/EUR/rates')).toHaveLength(1));
  });

  it('shows the problems the API finds under their fields', async () => {
    const { user } = renderHistory([
      reply(
        'POST /api/v1/currencies/{code}/rates/impact',
        {
          status: 400,
          body: validationFailed([
            { field: 'date', message: 'EUR already has a rate on 19 Oct 2026. Edit that one.' },
          ]),
        },
        { code: 'EUR' },
      ),
    ]);
    await screen.findByRole('table');

    await user.click(screen.getByRole('button', { name: 'Add rate' }));
    const dialog = await screen.findByRole('dialog', { name: 'Add rate of EUR' });
    await user.type(within(dialog).getByRole('textbox', { name: /^Rate/ }), '0.95');
    await user.click(within(dialog).getByRole('button', { name: 'Save rate' }));

    expect(
      await within(dialog).findByText('EUR already has a rate on 19 Oct 2026. Edit that one.'),
    ).toBeInTheDocument();
    expect(within(dialog).getByLabelText(/^Date/)).toHaveAttribute('aria-invalid', 'true');
  });
});

describe('RateDialog: Edit (016 FR-003)', () => {
  it("keeps the rate's date and saves the rate with its version", async () => {
    const { user, api } = renderHistory([
      reply('POST /api/v1/exchange-rates/{rateId}/impact', NO_IMPACT, { rateId: 'XR-1103' }),
      reply(
        'PATCH /api/v1/exchange-rates/{rateId}',
        { body: { ...EUR_FUTURE, rate: '0.96', version: 2 } },
        { rateId: 'XR-1103' },
      ),
    ]);
    await screen.findByRole('table');

    await chooseAction('the rate of 01 Jan 2027', 'Edit', user);
    const dialog = await screen.findByRole('dialog', { name: 'Edit rate of EUR' });
    expect(dialog).toHaveTextContent("A rate's date can't change");
    expect(dialog).toHaveTextContent('No conversion changes before 01 Jan 2027.');
    const rate = within(dialog).getByRole('textbox', { name: /^Rate/ });
    await user.clear(rate);
    await user.type(rate, '0.96');
    await user.click(within(dialog).getByRole('button', { name: 'Save rate' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(api.bodiesOf('PATCH /api/v1/exchange-rates/XR-1103')).toEqual([
      { rate: '0.96', version: 1 },
    ]);
  });

  it('names who saved first, keeps the typed rate and saves over it next time', async () => {
    let conflicted = false;
    const savedByDaniel = { ...EUR_FUTURE, version: 2 };
    const { user, api } = renderHistory([
      reply(
        'GET /api/v1/currencies/{code}/rates',
        () => ({
          body: conflicted
            ? { ...EUR_RATES, items: [savedByDaniel, ...EUR_RATES.items.slice(1)] }
            : EUR_RATES,
        }),
        { code: 'EUR' },
      ),
      reply('POST /api/v1/exchange-rates/{rateId}/impact', NO_IMPACT, { rateId: 'XR-1103' }),
      reply(
        'PATCH /api/v1/exchange-rates/{rateId}',
        () => {
          if (!conflicted) {
            conflicted = true;
            return { status: 409, body: versionConflict() };
          }
          return { body: { ...EUR_FUTURE, rate: '0.97', version: 3 } };
        },
        { rateId: 'XR-1103' },
      ),
    ]);
    await screen.findByRole('table');

    await chooseAction('the rate of 01 Jan 2027', 'Edit', user);
    const dialog = await screen.findByRole('dialog', { name: 'Edit rate of EUR' });
    const rate = within(dialog).getByRole('textbox', { name: /^Rate/ });
    await user.clear(rate);
    await user.type(rate, '0.97');
    await user.click(within(dialog).getByRole('button', { name: 'Save rate' }));

    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      'Not saved — Daniel Okoro changed this rate',
    );
    expect(rate).toHaveValue('0.97');

    await user.click(within(dialog).getByRole('button', { name: 'Save rate' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(api.bodiesOf('PATCH /api/v1/exchange-rates/XR-1103')).toEqual([
      { rate: '0.97', version: 1 },
      { rate: '0.97', version: 2 },
    ]);
  });
});
