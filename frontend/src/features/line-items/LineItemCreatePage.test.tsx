import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { LineItemCreatePage } from './LineItemCreatePage';
import { attribute, option } from '@/testing/fixtures/attribute-forms';
import {
  activity,
  approvedActivity,
  formAnswers,
  formOptions,
  lineItem,
  lineItemForm,
  notFound,
  restartRefusal,
  usd,
  validationError,
} from '@/testing/fixtures/line-items';

const PATH = '/activities/:activityId/line-items/new';
const ROUTE = '/activities/A-1001/line-items/new';
const CREATE = 'POST /api/v1/activities/A-1001/line-items';

function renderPage(answers: Record<string, MockHandler> = {}) {
  const api = mockApi({
    ...formAnswers(),
    'GET /api/v1/activities/A-1001/line-items/form-options': { body: formOptions() },
    ...answers,
  });
  return {
    api,
    ...renderWithProviders(<LineItemCreatePage />, { route: ROUTE, path: PATH, user: testUser() }),
  };
}

/** A button outside the dialogs, found quickly in the long form. */
function button(name: string) {
  return screen.getByText(name, { selector: 'button' });
}

async function findForm() {
  return screen.findByLabelText('Line item name');
}

describe('LineItemCreatePage (US2)', () => {
  it('opens filled in with the defaults, each labelled with its source (scenario 1, FR-008)', async () => {
    renderPage();

    expect(await findForm()).toHaveValue('Launch film – line item');
    expect(document.title).toBe('New line item · MAPP');
    expect(screen.getByRole('heading', { level: 1, name: 'New line item' })).toBeInTheDocument();
    expect(
      within(screen.getByRole('navigation', { name: 'Breadcrumb' })).getByRole('link', {
        name: 'Launch film',
      }),
    ).toBeInTheDocument();
    expect(screen.getByText("From the activity's name")).toBeInTheDocument();
    expect(screen.getByLabelText('Currency')).toHaveValue('USD');
    expect(screen.getByLabelText('Planned cost')).toHaveValue('20,000.00');
    expect(screen.getByText("The activity's unassigned planned cost")).toBeInTheDocument();
    expect(screen.getByLabelText('Evenly')).toBeChecked();
    expect(screen.getByText("The plan's default")).toBeInTheDocument();
    expect(screen.getByLabelText('Finance budget year')).toHaveValue('2026');
    expect(screen.getByText("The plan's year")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Items of '2026 Lubricants Budget', the plan's default finance budget, are listed first.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('Request SWBS code from GSAP')).toBeChecked();
    expect(screen.getByText(/On by default/)).toBeInTheDocument();
    expect(screen.getByText('Filled by GSAP later')).toBeInTheDocument();

    const summary = screen.getByLabelText('Activity summary');
    expect(summary).toHaveTextContent('Activity planned cost120,000.00 USD');
    expect(summary).toHaveTextContent('Unassigned now20,000.00 USD');
    expect(summary).toHaveTextContent('After saving0.00 USD');

    const breadcrumb = screen.getByLabelText('Breadcrumb');
    expect(within(breadcrumb).getByRole('link', { name: 'Launch film' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1001',
    );
    expect(within(breadcrumb).getByText('New line item')).toHaveAttribute('aria-current', 'page');
  });

  it('shows the sections in order, each a card, without a finance section (FR-007, 015 FR-017)', async () => {
    renderPage();
    await findForm();

    const headings = [...document.querySelectorAll('form h2')].map((h) => h.textContent);
    expect(headings).toEqual([
      'Basics',
      'Cost and currency',
      'Spread over time',
      'Finance coding',
      'Link to finance budget',
      'GSAP request',
    ]);
    expect(screen.queryByText('From finance (coming later)')).not.toBeInTheDocument();
    expect(screen.queryByText('Total actuals')).not.toBeInTheDocument();
    expect(screen.getByText(/of \d+ needed fields? filled$/)).toBeInTheDocument();
  });

  it('converts the balance when the currency changes and shows the USD amount live (scenario 2, FR-011, FR-012)', async () => {
    const { user } = renderPage();
    await findForm();

    await user.selectOptions(screen.getByLabelText('Currency'), 'EUR');
    const cost = screen.getByLabelText('Planned cost');
    expect(cost).toHaveValue('18,400.00');
    expect(screen.getByText('Changed to 18,400.00 EUR = 20,000.00 USD')).toBeInTheDocument();

    await user.clear(cost);
    await user.type(cost, '10000');
    expect(
      await screen.findByText(/^= 10,869\.57 USD at 0\.92 EUR per USD \(rate of 0?1 Mar 2026\)$/),
    ).toBeInTheDocument();
    expect(screen.getByText('Calculated')).toBeInTheDocument();
    expect(screen.getByLabelText('Activity summary')).toHaveTextContent('After saving9,130.43 USD');
  });

  it('calculates in the plan currency of a GBP plan, with both rates and their date (016 FR-009, FR-015)', async () => {
    const gbp = (amount: string) => ({ amount, currency: 'GBP' });
    const { user } = renderPage({
      'GET /api/v1/activities/A-1001/line-items/form-options': {
        body: formOptions({
          balance: {
            plannedCost: gbp('40000.00'),
            assigned: gbp('15000.00'),
            balance: gbp('25000.00'),
            state: 'left',
            roundingHint: false,
          },
          defaults: {
            ...formOptions().defaults,
            currency: 'GBP',
            plannedCost: gbp('25000.00'),
          },
        }),
      },
    });
    await findForm();

    // The pre-filled balance keeps its plan-currency value in another currency.
    await user.selectOptions(screen.getByLabelText('Currency'), 'EUR');
    const cost = screen.getByLabelText('Planned cost');
    expect(cost).toHaveValue('29,113.92');
    expect(screen.getByText('Changed to 29,113.92 EUR = 25,000.00 GBP')).toBeInTheDocument();

    await user.clear(cost);
    await user.type(cost, '23000');
    expect(
      await screen.findByText(
        /^= 19,750\.00 GBP at 0\.92 EUR and 0\.79 GBP per USD \(rates of 0?1 Mar 2026\)$/,
      ),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('Activity summary')).toHaveTextContent('After saving5,250.00 GBP');
  });

  it('makes the GL code depend on the spend type (scenarios 3 and 4, FR-014)', async () => {
    const { user, api } = renderPage({ [CREATE]: { status: 201, body: lineItem() } });
    await findForm();

    const glCode = screen.getByLabelText('GL code');
    expect(glCode).toBeDisabled();
    expect(screen.getByText('Choose a spend type first.')).toBeInTheDocument();

    await user.selectOptions(screen.getByLabelText('Spend type'), 'ATL – Media');
    await user.selectOptions(glCode, '7250080 – Advertising digital');
    await user.selectOptions(screen.getByLabelText('Spend type'), 'BTL – Event');
    expect(
      screen.getByText(
        "GL code '7250080 – Advertising digital' will be cleared — it doesn't fit 'BTL – Event'.",
      ),
    ).toBeInTheDocument();

    await user.click(button('Save'));
    await waitFor(() => expect(api.bodiesOf(CREATE)).toHaveLength(1));
    expect(api.bodiesOf(CREATE)[0]).toMatchObject({
      attributes: { 'SA-ST': 'SO-ST-BTL', 'SA-GL': null },
    });
  });

  it('fills in the only GL code of a spend type (scenario 3)', async () => {
    const { user } = renderPage();
    await findForm();

    await user.selectOptions(screen.getByLabelText('Spend type'), 'BTL – Event');
    expect(screen.getByLabelText('GL code')).toHaveValue('SO-GL-EVT');
  });

  it('shows every problem at once and saves nothing (scenario 5, FR-009)', async () => {
    const { user, router } = renderPage({
      [CREATE]: validationError([
        { field: 'name', message: 'Enter a name for the line item.' },
        {
          field: 'plannedCost',
          message: "Planned cost can't be negative.",
        },
        { field: 'spread.months[0].amount', message: "Month amounts can't be negative." },
      ]),
    });
    const name = await findForm();

    await user.clear(name);
    await user.click(button('Save'));

    expect(await screen.findByText('3 problems need your attention')).toBeInTheDocument();
    expect(screen.getAllByText('Enter a name for the line item.')).toHaveLength(2);
    expect(screen.getAllByText("Planned cost can't be negative.")).toHaveLength(2);
    expect(router.state.location.pathname).toBe(ROUTE);
  });

  it('creates the line item, says so and returns to the activity with the row highlighted (scenario 6, FR-010, FR-023)', async () => {
    const { user, api, router, queryClient } = renderPage({
      [CREATE]: { status: 201, body: lineItem({ id: 'L-2001', name: 'Agency fee' }) },
    });
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
    const name = await findForm();

    await user.clear(name);
    await user.type(name, 'Agency fee');
    await user.click(button('Save'));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1001'),
    );
    expect(router.state.location.hash).toBe('#line-items');
    expect(router.state.location.state).toEqual({ highlightLineItemId: 'L-2001' });
    expect(await screen.findByText("Line item 'Agency fee' created")).toBeInTheDocument();
    expect(api.bodiesOf(CREATE)[0]).toEqual({
      name: 'Agency fee',
      ownerId: 'U-1002',
      description: '',
      currency: 'USD',
      plannedCost: '20000.00',
      spread: { method: 'evenly', months: [] },
      attributes: {},
      contractedAmount: null,
      requestSwbs: true,
      financeBudgetYear: 2026,
      financeBudgetItemId: null,
      restartApproval: false,
    });
    const keys = invalidate.mock.calls.map(([filters]) => filters?.queryKey);
    expect(keys).toEqual(
      expect.arrayContaining([['line-items'], ['activity'], ['plan-activities'], ['plans']]),
    );
  });

  it('opens a new form with the new balance after Save and add another (scenario 7)', async () => {
    let round = 0;
    const nothingLeft = formOptions();
    nothingLeft.defaults = {
      ...nothingLeft.defaults,
      name: 'Launch film – line item 2',
      plannedCost: null,
      plannedCostSource: 'nothingLeftToAssign',
    };
    nothingLeft.balance = { ...nothingLeft.balance, balance: usd('0.00'), state: 'allAssigned' };
    const { user, router } = renderPage({
      'GET /api/v1/activities/A-1001/line-items/form-options': () => ({
        body: round++ === 0 ? formOptions() : nothingLeft,
      }),
      [CREATE]: { status: 201, body: lineItem({ name: 'Launch film – line item' }) },
    });
    await findForm();

    await user.click(button('Save and add another'));

    await waitFor(() =>
      expect(screen.getByLabelText('Line item name')).toHaveValue('Launch film – line item 2'),
    );
    expect(screen.getByLabelText('Planned cost')).toHaveValue('');
    expect(
      screen.getByText("Nothing left to assign — the activity's planned cost is fully assigned"),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('Line item name')).toHaveFocus();
    expect(router.state.location.pathname).toBe(ROUTE);
  });

  it('asks before adding to an approved activity and confirms the restart (FR-031)', async () => {
    const options = formOptions({ activity: approvedActivity() });
    const { user, api } = renderPage({
      'GET /api/v1/activities/A-1001/line-items/form-options': { body: options },
      [CREATE]: { status: 201, body: lineItem() },
    });
    await findForm();

    await user.click(button('Save'));
    const dialog = await screen.findByRole('alertdialog', {
      name: 'This activity will need approval again',
    });
    expect(dialog).toHaveTextContent(
      "'Launch film' is approved. Saving sends it back to Created, and it must be submitted for approval again.",
    );
    expect(dialog).toHaveTextContent("New line item 'Launch film – line item': 20,000.00 USD");
    expect(within(dialog).getByRole('button', { name: 'Keep editing' })).toHaveFocus();
    expect(api.bodiesOf(CREATE)).toHaveLength(0);

    await user.click(within(dialog).getByRole('button', { name: 'Save and restart approval' }));
    await waitFor(() => expect(api.bodiesOf(CREATE)).toHaveLength(1));
    expect(api.bodiesOf(CREATE)[0]).toMatchObject({ restartApproval: true });
  });

  it('asks for the restart when the activity was submitted meanwhile (D-146)', async () => {
    const { user } = renderPage({ [CREATE]: restartRefusal() });
    await findForm();

    await user.click(button('Save'));

    expect(
      await screen.findByRole('alertdialog', { name: 'This activity will need approval again' }),
    ).toBeInTheDocument();
  });

  it('says when the activity was deleted (edge case)', async () => {
    const message =
      'This activity was deleted — restore it from the recycle bin to keep working on its line items.';
    renderPage({ 'GET /api/v1/activities/A-1001/line-items/form-options': notFound(message) });

    expect(await screen.findByText(message)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to the recycle bin' })).toHaveAttribute(
      'href',
      '/recycle-bin',
    );
  });

  it('shows a load failure with Try again', async () => {
    renderPage({
      'GET /api/v1/activities/A-1001/line-items/form-options': {
        status: 500,
        body: {
          error: {
            code: 'internal_error',
            message: 'Something went wrong. Please try again.',
            fields: [],
          },
        },
      },
    });

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('asks before discarding typed changes on Cancel', async () => {
    const { user, router } = renderPage();
    const name = await findForm();

    await user.type(name, ' extra');
    await user.click(button('Cancel'));
    const dialog = await screen.findByRole('alertdialog', { name: 'Discard changes?' });
    await user.click(within(dialog).getByRole('button', { name: 'Discard changes' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1001'),
    );
  });

  it('starts with an empty planned cost when the activity has none (D-144)', async () => {
    const options = formOptions({
      activity: activity({ plannedCost: null }),
      balance: {
        plannedCost: null,
        assigned: usd('0.00'),
        balance: null,
        state: 'noPlannedCost',
        roundingHint: false,
      },
    });
    options.defaults = {
      ...options.defaults,
      plannedCost: null,
      plannedCostSource: 'noActivityPlannedCost',
    };
    renderPage({ 'GET /api/v1/activities/A-1001/line-items/form-options': { body: options } });

    await findForm();
    expect(
      screen.getByText('This activity has no planned cost yet, so there is nothing to pre-fill'),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        'This activity has no planned cost yet — the unassigned planned cost appears once it has one.',
      ),
    ).toBeInTheDocument();
  });

  /** The line-item form with an attribute shown only for the Mobility business unit. */
  function mobilityOnlyForm() {
    const form = lineItemForm();
    form.sections[0].attributes.push(
      attribute({
        id: 'SA-MOB',
        label: 'Mobility channel',
        conditionOptionIds: ['SO-BU-MOB'],
        options: [option('SO-MOB-1', 'Fleet')],
      }),
    );
    return { body: form };
  }

  it('hides a line-item attribute of another business unit than the plan has (002 FR-023, D-212)', async () => {
    renderPage({ 'GET /api/v1/shell-attribute-forms/lineItem': mobilityOnlyForm() });

    await findForm();
    expect(screen.queryByLabelText('Mobility channel')).not.toBeInTheDocument();
  });

  it("shows it from the activity's Business unit options alone, without asking for the plan (D-212)", async () => {
    const { api } = renderPage({
      'GET /api/v1/shell-attribute-forms/lineItem': mobilityOnlyForm(),
      'GET /api/v1/activities/A-1001/line-items/form-options': {
        body: formOptions({ activity: activity({ businessUnitOptionIds: ['SO-BU-MOB'] }) }),
      },
    });

    await findForm();
    expect(screen.getByLabelText('Mobility channel')).toBeInTheDocument();
    expect(api.requests.some((request) => request.path.startsWith('/api/v1/plans'))).toBe(false);
  });
});
