import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import { closestTo } from '@/testing/dom';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { FORM_ANSWERS, plan, planDefaults } from '@/testing/fixtures/plans';
import { reply } from '@/testing/replies';
import { PlanCreatePage } from './PlanCreatePage';

function renderPage(answers: HttpHandler[] = []) {
  const api = mockApi([
    ...FORM_ANSWERS,
    reply('GET /api/v1/plan-defaults', { body: planDefaults() }),
    ...answers,
  ]);
  const result = renderWithProviders(<PlanCreatePage />, {
    user: testUser(),
    route: '/plans/new',
    path: '/plans/new',
  });
  return { ...result, api };
}

describe('PlanCreatePage (US1)', () => {
  it('opens with the defaults from the API (US1#1, FR-006)', async () => {
    renderPage();

    expect(screen.getByRole('heading', { level: 1, name: 'New plan' })).toBeInTheDocument();
    expect(document.title).toBe('New plan · MAPP');
    expect(await screen.findByRole('combobox', { name: /Year/ })).toHaveValue('2026');
    expect(screen.getByRole('combobox', { name: /Goal stage/ })).toHaveValue('inq');
  });

  it('creates the plan, opens its page and says what to do next (US1#2)', async () => {
    const { api, user, router } = renderPage([
      reply('POST /api/v1/plans', {
        status: 201,
        body: plan({ id: 'P-1004', name: 'FY2026 Lubricants Test' }),
      }),
    ]);

    await user.type(
      await screen.findByRole('textbox', { name: /Plan name/ }),
      'FY2026 Lubricants Test',
    );
    await user.selectOptions(screen.getByRole('combobox', { name: /Model/ }), 'MOD-1001');
    await user.type(screen.getByRole('textbox', { name: /Plan budget/ }), '1,200,000');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1004'));
    expect(
      await screen.findByText("Plan 'FY2026 Lubricants Test' created — add campaigns next."),
    ).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/plans')).toEqual([
      {
        name: 'FY2026 Lubricants Test',
        year: 2026,
        modelId: 'MOD-1001',
        teamIds: ['T-1001'],
        ownerId: 'U-1002',
        aop: false,
        description: '',
        budget: { amount: '1200000.00', currency: 'USD' },
        defaultFinanceBudgetId: null,
        defaultSpread: 'evenly',
        bcpir: false,
        goalStage: 'inq',
        goalAmount: null,
        attributes: {},
      },
    ]);
  });

  it('shows every problem of the API at once and stays (US1#3, FR-010)', async () => {
    const { user, router } = renderPage([
      reply('POST /api/v1/plans', {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Some fields need your attention.',
            fields: [
              { field: 'name', message: 'Enter a plan name.' },
              { field: 'modelId', message: 'Choose a model.' },
            ],
          },
        },
      }),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Save' }));

    const summary = await screen.findByText('2 problems need your attention');
    expect(within(closestTo(summary, 'div')).getByText('Enter a plan name.')).toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole('alert').parentElement).toHaveFocus());
    expect(screen.getAllByText('Choose a model.')).toHaveLength(2);
    expect(router.state.location.pathname).toBe('/plans/new');
  });

  it('asks before typed values are discarded', async () => {
    const { user } = renderPage();

    await user.type(await screen.findByRole('textbox', { name: /Plan name/ }), 'Draft idea');
    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(
      await screen.findByRole('alertdialog', { name: 'Discard changes?' }),
    ).toBeInTheDocument();
  });

  it('says when the defaults could not be loaded, with Try again', async () => {
    renderPage([reply('GET /api/v1/plan-defaults', { status: 500 })]);

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent("MAPP can't reach its server");
  });
});
