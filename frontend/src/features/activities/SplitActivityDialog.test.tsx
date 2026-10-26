import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
import type { ActivitySplitOptions, ActivitySplitPreview, ActivitySplitSide } from './splitApi';
import { SplitActivityDialog } from './SplitActivityDialog';
import { destination, launchFilm, OWN_INITIATIVE } from './testData';

const OPTIONS = 'GET /api/v1/activities/A-1042/split-options';
const PREVIEW = 'POST /api/v1/activities/A-1042/split-preview';
const SPLIT = 'POST /api/v1/activities/A-1042/split';

const gbp = (amount: string) => ({ amount, currency: 'GBP' });
const eur = (amount: string) => ({ amount, currency: 'EUR' });

/** "Launch film" planned in GBP: Venue 15,000.00 GBP and Agency fee (Ireland) 23,000.00 EUR. */
function options(overrides: Partial<ActivitySplitOptions> = {}): ActivitySplitOptions {
  return {
    activityId: 'A-1042',
    version: 3,
    status: 'created',
    plannedCost: gbp('40000.00'),
    planCurrency: 'GBP',
    suggestedName: 'Launch film (split)',
    needsRestart: false,
    lineItems: [
      {
        id: 'L-1009',
        name: 'Venue',
        plannedCost: gbp('15000.00'),
        plannedCostPlan: gbp('15000.00'),
      },
      {
        id: 'L-1010',
        name: 'Agency fee (Ireland)',
        plannedCost: eur('23000.00'),
        plannedCostPlan: gbp('19750.00'),
      },
    ],
    destinations: [
      OWN_INITIATIVE,
      destination({
        initiative: { type: 'initiative', id: 'I-1005', name: 'Ireland' },
      }),
    ],
    ...overrides,
  };
}

function side(overrides: Partial<ActivitySplitSide> = {}): ActivitySplitSide {
  return {
    name: 'Launch film',
    initiativeName: 'Digital & social',
    plannedCost: gbp('20250.00'),
    lineItemNames: ['Venue'],
    unassignedPlannedCost: gbp('5250.00'),
    unassignedState: 'left',
    status: 'created',
    ...overrides,
  };
}

const PREVIEWED: ActivitySplitPreview = {
  original: side(),
  created: side({
    name: 'Ireland roadshow',
    initiativeName: 'Ireland',
    plannedCost: gbp('19750.00'),
    lineItemNames: ['Agency fee (Ireland)'],
    unassignedPlannedCost: gbp('0.00'),
    unassignedState: 'allAssigned',
  }),
  problems: [],
};

function openDialog(answers: Record<string, MockHandler> = {}, activity = launchFilm()) {
  const onClose = vi.fn();
  const api = mockApi({
    [OPTIONS]: { body: options() },
    [PREVIEW]: { body: PREVIEWED },
    [SPLIT]: {
      body: {
        activity: launchFilm(),
        newActivity: launchFilm({ id: 'A-1050', name: 'Ireland roadshow' }),
      },
    },
    ...answers,
  });
  return {
    api,
    onClose,
    ...renderWithProviders(<SplitActivityDialog activity={activity} onClose={onClose} />, {
      user: testUser(),
    }),
  };
}

describe('SplitActivityDialog (016 US4, FR-026 – FR-029)', () => {
  it('moves the ticked line items with their amount, previews both activities and splits (scenario 1)', async () => {
    const { user, api, onClose } = openDialog();

    const dialog = await screen.findByRole('dialog', { name: "Split 'Launch film'" });
    const name = await within(dialog).findByRole('textbox', { name: /^New activity name/ });
    expect(name).toHaveValue('Launch film (split)');
    expect(within(dialog).getByRole('radio', { name: /Digital & social/ })).toBeChecked();
    expect(within(dialog).getByText('19,750.00 GBP')).toBeVisible();
    expect(within(dialog).getByText('(23,000.00 EUR)')).toBeVisible();

    await user.clear(name);
    await user.type(name, 'Ireland roadshow');
    await user.click(within(dialog).getByRole('radio', { name: /Ireland/ }));
    await user.click(within(dialog).getByRole('checkbox', { name: 'Agency fee (Ireland)' }));
    // The amount follows the ticks until typed.
    expect(within(dialog).getByRole('textbox', { name: /^Planned cost to move/ })).toHaveValue(
      '19,750.00',
    );

    const preview = await within(dialog).findByRole('region', { name: 'After the split' });
    expect(await within(preview).findByText('New: Ireland roadshow')).toBeVisible();
    expect(within(preview).getByText('20,250.00 GBP')).toBeVisible();
    expect(within(preview).getByText('5,250.00 GBP')).toBeVisible();
    expect(within(dialog).getByText("Goals stay on 'Launch film'.")).toBeVisible();

    await user.click(within(dialog).getByRole('button', { name: 'Split activity' }));

    await waitFor(() =>
      expect(api.bodiesOf(SPLIT)).toEqual([
        {
          name: 'Ireland roadshow',
          initiativeId: 'I-1005',
          lineItemIds: ['L-1010'],
          plannedCost: '19750.00',
          restartApproval: false,
          version: 3,
        },
      ]),
    );
    expect(onClose).toHaveBeenCalled();
    expect(
      await screen.findByText("'Launch film' split — 'Ireland roadshow' created."),
    ).toBeInTheDocument();
  });

  it('says what to choose when nothing moves, and sends nothing (scenario 4)', async () => {
    const { user, api } = openDialog();

    const dialog = await screen.findByRole('dialog', { name: "Split 'Launch film'" });
    await within(dialog).findByRole('checkbox', { name: 'Venue' });
    await user.click(within(dialog).getByRole('button', { name: 'Split activity' }));

    expect(within(dialog).getByRole('alert')).toHaveTextContent(
      'Choose line items or an amount to move.',
    );
    expect(api.bodiesOf(SPLIT)).toEqual([]);
    expect(api.bodiesOf(PREVIEW)).toEqual([]);
  });

  it('says an approved activity will need approval again and asks for it (scenario 3, FR-029)', async () => {
    const { user, api } = openDialog(
      { [OPTIONS]: { body: options({ status: 'approved', needsRestart: true }) } },
      launchFilm({ status: 'approved', displayStatus: 'approved' }),
    );

    const dialog = await screen.findByRole('dialog', { name: "Split 'Launch film'" });
    expect(
      await within(dialog).findByText('This activity will need approval again.'),
    ).toBeVisible();
    await user.click(within(dialog).getByRole('checkbox', { name: 'Venue' }));
    await user.click(within(dialog).getByRole('button', { name: 'Split activity' }));

    await waitFor(() => expect(api.bodiesOf(SPLIT)).toHaveLength(1));
    expect(api.bodiesOf(SPLIT)[0]).toMatchObject({
      restartApproval: true,
      lineItemIds: ['L-1009'],
    });
  });

  it('lists an initiative that cannot take the activity with Move’s reason (scenario 5)', async () => {
    openDialog({
      [OPTIONS]: {
        body: options({
          destinations: [
            OWN_INITIATIVE,
            destination({
              initiative: { type: 'initiative', id: 'I-1006', name: 'Retail events' },
              moveProblem: "Its dates (01 Jul – 30 Sep 2026) don't contain the activity's dates.",
            }),
          ],
        }),
      },
    });

    const dialog = await screen.findByRole('dialog', { name: "Split 'Launch film'" });
    expect(await within(dialog).findByRole('radio', { name: /Retail events/ })).toBeDisabled();
    expect(
      within(dialog).getByText(
        "Its dates (01 Jul – 30 Sep 2026) don't contain the activity's dates.",
      ),
    ).toBeVisible();
  });

  it('shows every problem the preview finds at once', async () => {
    const { user } = openDialog({
      [PREVIEW]: {
        body: {
          ...PREVIEWED,
          problems: [
            {
              field: 'plannedCost',
              message: "Move at most 40,000.00 GBP, the activity's planned cost.",
            },
            {
              field: 'name',
              message: "'Ireland roadshow' is already an activity of this initiative.",
            },
          ],
        },
      },
    });

    const dialog = await screen.findByRole('dialog', { name: "Split 'Launch film'" });
    await user.click(await within(dialog).findByRole('checkbox', { name: 'Venue' }));

    const alert = await within(dialog).findByRole('alert');
    await waitFor(() =>
      expect(alert).toHaveTextContent("Move at most 40,000.00 GBP, the activity's planned cost."),
    );
    expect(alert).toHaveTextContent(
      "'Ireland roadshow' is already an activity of this initiative.",
    );
  });
});
