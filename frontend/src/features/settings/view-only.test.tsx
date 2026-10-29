import { screen, within } from '@testing-library/react';
import type { HttpHandler } from 'msw';
import type { ReactElement } from 'react';
import { describe, expect, it } from 'vitest';
import { TYPES } from '@/testing/fixtures/activity-types';
import { FIELD_HELP_PAGE } from '@/testing/fixtures/field-help';
import { BUDGETS } from '@/testing/fixtures/finance-budget-items';
import { EUR_RATES, NO_IMPORTS } from '@/testing/fixtures/rates';
import { CURRENCIES, MODELS } from '@/testing/fixtures/settings';
import { VIEWER } from '@/testing/fixtures/viewer';
import { expectNoAxeViolations } from '@/testing/axe';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { reply } from '@/testing/replies';
import { ActivityTypesPage } from './activity-types/ActivityTypesPage';
import { CurrenciesPage } from './currencies/CurrenciesPage';
import { RateHistorySection } from './currencies/RateHistorySection';
import { FieldHelpPage } from './field-help/FieldHelpPage';
import { FinanceBudgetItemsPage } from './finance-budget-items/FinanceBudgetItemsPage';

const SETTINGS_NOTE =
  'View only — only administrators change Settings. Ask Alex Morgan if something needs to change.';

const PAGES: [string, ReactElement, HttpHandler[]][] = [
  [
    'Activity types',
    <ActivityTypesPage key="types" />,
    [
      reply('GET /api/v1/activity-types', { body: TYPES }),
      reply('GET /api/v1/models', { body: MODELS }),
    ],
  ],
  [
    'Currencies',
    <CurrenciesPage key="currencies" />,
    [
      reply('GET /api/v1/currencies', { body: CURRENCIES }),
      reply('GET /api/v1/exchange-rates/imports', { body: NO_IMPORTS }),
    ],
  ],
  [
    'Finance budget items',
    <FinanceBudgetItemsPage key="budgets" />,
    [reply('GET /api/v1/finance-budgets', BUDGETS)],
  ],
  [
    'Field help',
    <FieldHelpPage key="help" />,
    [reply('GET /api/v1/field-help', { body: FIELD_HELP_PAGE })],
  ],
];

describe('014 FR-011, FR-012: a Settings list for someone who may not change it (US2 scenario 5)', () => {
  it.each(PAGES)('%s: one View only note and no change action', async (_name, page, answers) => {
    mockApi(answers);
    renderWithProviders(page, { user: VIEWER });

    expect(await screen.findAllByRole('table')).not.toHaveLength(0);
    expect(screen.getByRole('note')).toHaveTextContent(SETTINGS_NOTE);
    expect(screen.queryByRole('button', { name: /^New / })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: /^New |^Import/ })).not.toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('Field help: the labels are text, not buttons that open the editor', async () => {
    mockApi([reply('GET /api/v1/field-help', { body: FIELD_HELP_PAGE })]);
    renderWithProviders(<FieldHelpPage />, { user: VIEWER });

    const table = await screen.findByRole('table');
    expect(within(table).queryAllByRole('button')).toEqual([]);
  });

  it('Currencies: the rate history offers no Add rate and no row actions', async () => {
    mockApi([reply('GET /api/v1/currencies/{code}/rates', { body: EUR_RATES })]);
    renderWithProviders(<RateHistorySection code="EUR" />, { user: VIEWER });

    await screen.findByRole('table');
    expect(screen.queryByRole('button', { name: 'Add rate' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Actions for/ })).not.toBeInTheDocument();
  });
});
