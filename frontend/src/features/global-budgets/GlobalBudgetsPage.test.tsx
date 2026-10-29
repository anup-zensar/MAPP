import { screen, waitFor, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { PEOPLE } from '@/testing/fixtures/approval-workflows';
import { closestTo, nth } from '@/testing/dom';
import { renderWithProviders } from '@/testing/render';
import { mockApi } from '@/testing/mock-api';
import { testUser } from '@/testing/fixtures/users';
import { budgetSummary, globalBudget } from '@/testing/fixtures/global-budgets';
import { reply } from '@/testing/replies';
import { GlobalBudgetsPage } from './GlobalBudgetsPage';

const AVAILABLE = { available: true, reason: '' };

function listOf(items = [budgetSummary()]) {
  return { body: { items, total: items.length, years: [2027, 2026], create: AVAILABLE } };
}

function renderPage(route = '/budgets', answers: HttpHandler[] = []) {
  const api = mockApi([reply('GET /api/v1/global-budgets', listOf()), ...answers]);
  return {
    ...renderWithProviders(<GlobalBudgetsPage />, { route, path: '/budgets', user: testUser() }),
    api,
  };
}

describe('GlobalBudgetsPage (014 US6, FR-036)', () => {
  // The page opens on the current year, and the year choices start from it.
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'], now: new Date('2026-03-12T09:00:00Z') });
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('lists the global budgets of the year with their four figures', async () => {
    const { api } = renderPage();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Global budgets' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Global budgets · SP&A One');
    const link = await screen.findByRole('link', { name: 'FY2026 Global marketing budget' });
    expect(link).toHaveAttribute('href', '/budgets/GB-1001');
    const card = closestTo(link, 'li');
    expect(card).toHaveTextContent('2026 · Owner: Carlos Mendes · Allocated to 2 plans');
    const figure = (label: string) =>
      within(card).getByText(label, { selector: 'dt' }).nextElementSibling;
    expect(figure('Global budget')).toHaveTextContent('$2,500,000.00');
    expect(figure('Allocated')).toHaveTextContent('$2,000,000.00');
    expect(figure('Unallocated')).toHaveTextContent('$500,000.00');
    expect(figure('Planned cost')).toHaveTextContent('$670,000.00');
    expect(screen.getByText('1 global budget')).toBeInTheDocument();
    expect(nth(api.requests, 0).search).toBe('?year=2026');
  });

  it('shows an over-allocated budget in red with words (US6 scenario 5)', async () => {
    renderPage('/budgets', [
      reply(
        'GET /api/v1/global-budgets',
        listOf([
          budgetSummary({
            figures: {
              ...globalBudget().figures,
              allocated: { amount: '2600000.00', currency: 'USD' },
              unallocated: { amount: '-100000.00', currency: 'USD' },
              overAllocated: true,
            },
          }),
        ]),
      ),
    ]);

    expect(await screen.findByText('Over-allocated by $100,000.00')).toBeInTheDocument();
    expect(screen.getByText('-$100,000.00')).toBeInTheDocument();
  });

  it('changes the year through the address, and says when a year has none', async () => {
    const { api, router, user } = renderPage('/budgets', [
      reply('GET /api/v1/global-budgets', (request) =>
        new URL(request.url).searchParams.get('year') === '2027'
          ? { body: { items: [], total: 0, years: [2027, 2026], create: AVAILABLE } }
          : listOf(),
      ),
    ]);
    await screen.findByRole('link', { name: 'FY2026 Global marketing budget' });

    await user.selectOptions(screen.getByRole('combobox', { name: 'Year' }), '2027');
    await waitFor(() => expect(router.state.location.search).toBe('?year=2027'));
    expect(await screen.findByText('No global budgets for 2027 yet.')).toBeInTheDocument();

    await user.selectOptions(screen.getByRole('combobox', { name: 'Year' }), 'All years');
    await waitFor(() => expect(router.state.location.search).toBe('?year=all'));
    await waitFor(() => expect(api.requests.at(-1)?.search).toBe(''));
  });

  it('creates a global budget from New global budget and opens it', async () => {
    const { api, router, user } = renderPage('/budgets', [
      reply('GET /api/v1/people', PEOPLE),
      reply('POST /api/v1/global-budgets', {
        status: 201,
        body: globalBudget({ id: 'GB-1002', name: 'FY2027 Global brand' }),
      }),
    ]);
    await screen.findByRole('link', { name: 'FY2026 Global marketing budget' });

    await user.click(screen.getByRole('button', { name: 'New global budget' }));
    const panel = await screen.findByRole('dialog', { name: 'New global budget' });
    await user.click(within(panel).getByRole('textbox', { name: /Name/ }));
    await user.paste('FY2027 Global brand');
    await user.selectOptions(within(panel).getByRole('combobox', { name: /Year/ }), '2027');
    await user.type(within(panel).getByRole('textbox', { name: /Amount/ }), '1.5m');
    await user.click(within(panel).getByRole('combobox', { name: /Owner/ }));
    await user.click(await screen.findByRole('option', { name: /Carlos Mendes/ }));
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/budgets/GB-1002'));
    expect(api.bodiesOf('POST /api/v1/global-budgets')).toEqual([
      {
        name: 'FY2027 Global brand',
        year: 2027,
        amount: { amount: '1500000.00', currency: 'USD' },
        ownerId: 'U-1005',
        description: '',
        overspend: 'warn',
      },
    ]);
  });

  it('shows the field problems of the API all at once', async () => {
    const { user } = renderPage('/budgets', [
      reply('GET /api/v1/people', PEOPLE),
      reply('POST /api/v1/global-budgets', {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Check the highlighted fields.',
            fields: [
              {
                field: 'name',
                message: "A global budget called 'FY2026 Global marketing budget' already exists.",
              },
              {
                field: 'ownerId',
                message: 'Choose a person who can manage global budgets for all markets.',
              },
            ],
          },
        },
      }),
    ]);
    await screen.findByRole('link', { name: 'FY2026 Global marketing budget' });

    await user.click(screen.getByRole('button', { name: 'New global budget' }));
    const panel = await screen.findByRole('dialog', { name: 'New global budget' });
    await user.click(within(panel).getByRole('textbox', { name: /Name/ }));
    await user.paste('FY2026 Global marketing budget');
    await user.type(within(panel).getByRole('textbox', { name: /Amount/ }), '100');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(
      (
        await within(panel).findAllByText(
          "A global budget called 'FY2026 Global marketing budget' already exists.",
        )
      ).length,
    ).toBeGreaterThan(0);
    expect(
      within(panel).getAllByText('Choose a person who can manage global budgets for all markets.')
        .length,
    ).toBeGreaterThan(0);
  });

  it('checks the amount before saving', async () => {
    const { api, user } = renderPage('/budgets', [reply('GET /api/v1/people', PEOPLE)]);
    await screen.findByRole('link', { name: 'FY2026 Global marketing budget' });

    await user.click(screen.getByRole('button', { name: 'New global budget' }));
    const panel = await screen.findByRole('dialog', { name: 'New global budget' });
    await user.type(within(panel).getByRole('textbox', { name: /Amount/ }), '-5');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(
      (await within(panel).findAllByText('Enter an amount of 0 or more.')).length,
    ).toBeGreaterThan(0);
    expect(api.bodiesOf('POST /api/v1/global-budgets')).toEqual([]);
  });
});
