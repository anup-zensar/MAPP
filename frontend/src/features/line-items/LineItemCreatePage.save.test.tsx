import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { approvedActivity, lineItem, usd } from '@/testing/fixtures/line-items';
import { formOptions, restartRefusal, validationError } from '@/testing/fixtures/line-item-forms';
import {
  CREATE,
  CREATE_ROUTE,
  findNameField,
  formButton,
  renderCreatePage,
} from '@/testing/fixtures/line-item-pages';
import { reply } from '@/testing/replies';

describe('LineItemCreatePage, saving (US2)', () => {
  it('shows every problem at once and saves nothing (scenario 5, FR-009)', async () => {
    const { user, router } = renderCreatePage([
      reply(
        'POST /api/v1/activities/{activityId}/line-items',
        validationError([
          { field: 'name', message: 'Enter a name for the line item.' },
          {
            field: 'plannedCost',
            message: "Planned cost can't be negative.",
          },
          { field: 'spread.months[0].amount', message: "Month amounts can't be negative." },
        ]),
        { activityId: 'A-1001' },
      ),
    ]);
    const name = await findNameField();

    await user.clear(name);
    await user.click(formButton('Save'));

    expect(await screen.findByText('3 problems need your attention')).toBeInTheDocument();
    expect(screen.getAllByText('Enter a name for the line item.')).toHaveLength(2);
    expect(screen.getAllByText("Planned cost can't be negative.")).toHaveLength(2);
    expect(router.state.location.pathname).toBe(CREATE_ROUTE);
  });

  it('creates the line item, says so and returns to the activity with the row highlighted (scenario 6, FR-010, FR-023)', async () => {
    const { user, api, router, queryClient } = renderCreatePage([
      reply(
        'POST /api/v1/activities/{activityId}/line-items',
        { status: 201, body: lineItem({ id: 'L-2001', name: 'Agency fee' }) },
        { activityId: 'A-1001' },
      ),
    ]);
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
    const name = await findNameField();

    await user.clear(name);
    await user.type(name, 'Agency fee');
    await user.click(formButton('Save'));

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
    const { user, router } = renderCreatePage([
      reply(
        'GET /api/v1/activities/{activityId}/line-items/form-options',
        () => ({
          body: round++ === 0 ? formOptions() : nothingLeft,
        }),
        { activityId: 'A-1001' },
      ),
      reply(
        'POST /api/v1/activities/{activityId}/line-items',
        { status: 201, body: lineItem({ name: 'Launch film – line item' }) },
        { activityId: 'A-1001' },
      ),
    ]);
    await findNameField();

    await user.click(formButton('Save and add another'));

    await waitFor(() =>
      expect(screen.getByLabelText('Line item name')).toHaveValue('Launch film – line item 2'),
    );
    expect(screen.getByLabelText('Planned cost')).toHaveValue('');
    expect(
      screen.getByText("Nothing left to assign — the activity's planned cost is fully assigned"),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('Line item name')).toHaveFocus();
    expect(router.state.location.pathname).toBe(CREATE_ROUTE);
  });

  it('asks before adding to an approved activity and confirms the restart (FR-031)', async () => {
    const options = formOptions({ activity: approvedActivity() });
    const { user, api } = renderCreatePage([
      reply(
        'GET /api/v1/activities/{activityId}/line-items/form-options',
        { body: options },
        { activityId: 'A-1001' },
      ),
      reply(
        'POST /api/v1/activities/{activityId}/line-items',
        { status: 201, body: lineItem() },
        { activityId: 'A-1001' },
      ),
    ]);
    await findNameField();

    await user.click(formButton('Save'));
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
    const { user } = renderCreatePage([
      reply('POST /api/v1/activities/{activityId}/line-items', restartRefusal(), {
        activityId: 'A-1001',
      }),
    ]);
    await findNameField();

    await user.click(formButton('Save'));

    expect(
      await screen.findByRole('alertdialog', { name: 'This activity will need approval again' }),
    ).toBeInTheDocument();
  });

  it('asks before discarding typed changes on Cancel', async () => {
    const { user, router } = renderCreatePage();
    const name = await findNameField();

    await user.type(name, ' extra');
    await user.click(formButton('Cancel'));
    const dialog = await screen.findByRole('alertdialog', { name: 'Discard changes?' });
    await user.click(within(dialog).getByRole('button', { name: 'Discard changes' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1001'),
    );
  });
});
