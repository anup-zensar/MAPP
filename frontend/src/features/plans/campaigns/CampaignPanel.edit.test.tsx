import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DETAILS, findDetails, findForm, renderPanel } from '@/testing/fixtures/campaign-panel';
import { budgetByMonth } from '@/testing/fixtures/phasing';
import { campaign, planAnswer, usd } from '@/testing/fixtures/campaigns';
import { reply } from '@/testing/replies';

describe('CampaignPanel — edit (feature 005, US3)', () => {
  it('saves the form with the version it started from and goes back to the details (US3#2, FR-018)', async () => {
    const { user, api, router } = renderPanel(`${DETAILS}/edit`, [
      reply(
        'PATCH /api/v1/campaigns/{campaignId}',
        {
          body: campaign({ budget: usd('450000.00'), version: 4 }),
        },
        { campaignId: 'C-1001' },
      ),
    ]);
    const form = await findForm();
    expect(within(form).getByLabelText('Campaign name')).toHaveValue('Helix launch');
    expect(within(form).getByLabelText('Campaign budget')).toHaveValue('400,000.00');
    // The plan's remaining budget counts this campaign's saved budget once.
    expect(within(form).getByText('$200,000.00')).toBeInTheDocument();

    const budget = within(form).getByLabelText('Campaign budget');
    await user.clear(budget);
    await user.type(budget, '450000');
    expect(within(form).getByText('$150,000.00')).toBeInTheDocument();
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(router.state.location.pathname).toBe(DETAILS));
    expect(api.bodiesOf('PATCH /api/v1/campaigns/C-1001')).toEqual([
      {
        name: 'Helix launch',
        description: 'Launch of the new Helix range.',
        ownerId: 'U-1001',
        startDate: '2026-01-01',
        endDate: '2026-12-31',
        budget: { amount: '450000.00', currency: 'USD' },
        overspend: 'warn',
        attributes: {},
        version: 3,
      },
    ]);
    await findDetails();
    // The details panel is open over the messages area, which stays readable for screen readers.
    await waitFor(() =>
      expect(screen.getByRole('region', { name: 'Messages (F8)', hidden: true })).toHaveTextContent(
        "Campaign 'Helix launch' saved",
      ),
    );
  });

  it('types the budget in the plan currency and says clearing it removes Budget by month (016 FR-011, FR-022)', async () => {
    const gbp = (amount: string) => ({ amount, currency: 'GBP' });
    const { user, api } = renderPanel(`${DETAILS}/edit`, [
      reply(
        'GET /api/v1/plans/{planId}',
        { body: { ...planAnswer(), currency: 'GBP' } },
        { planId: 'P-1001' },
      ),
      reply(
        'GET /api/v1/campaigns/{campaignId}',
        {
          body: campaign({
            budget: gbp('120000.00'),
            parentRemainingBudget: gbp('80000.00'),
            budgetPhasing: {
              method: 'evenly',
              months: [],
              monthly: Array.from({ length: 12 }, () => gbp('10000.00')),
              total: gbp('120000.00'),
              problem: null,
            },
          }),
        },
        { campaignId: 'C-1001' },
      ),
      reply('PATCH /api/v1/campaigns/{campaignId}', { body: campaign() }, { campaignId: 'C-1001' }),
      reply(
        'GET /api/v1/plans/{planId}/budget-by-month',
        { body: budgetByMonth() },
        { planId: 'P-1001' },
      ),
    ]);
    const form = await findForm();
    const budget = within(form).getByLabelText('Campaign budget');
    expect(within(form).queryByText('Budget by month will be removed when you save.')).toBeNull();

    await user.clear(budget);
    expect(
      within(form).getByText('Budget by month will be removed when you save.'),
    ).toBeInTheDocument();
    await user.type(budget, '150000');
    expect(within(form).queryByText('Budget by month will be removed when you save.')).toBeNull();
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(api.bodiesOf('PATCH /api/v1/campaigns/C-1001')).toHaveLength(1));
    expect(api.bodiesOf('PATCH /api/v1/campaigns/C-1001')[0]).toMatchObject({
      budget: { amount: '150000.00', currency: 'GBP' },
    });
  });

  it('shows the FR-007 problem under the dates and saves nothing (US3#3, FR-007)', async () => {
    const problem =
      'Digital & social (01 Feb – 30 Nov 2026) would be outside these dates. Change its dates first, or choose dates that include it.';
    const { user, router } = renderPanel(`${DETAILS}/edit`, [
      reply(
        'PATCH /api/v1/campaigns/{campaignId}',
        {
          status: 400,
          body: {
            error: {
              code: 'validation_failed',
              message: 'Some fields need your attention.',
              fields: [{ field: 'startDate', message: problem }],
            },
          },
        },
        { campaignId: 'C-1001' },
      ),
    ]);
    const form = await findForm();

    fireEvent.change(within(form).getByLabelText('Start date'), {
      target: { value: '2026-03-01' },
    });
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await within(form).findByText('1 problem needs your attention')).toBeInTheDocument();
    expect(within(form).getByLabelText('Start date')).toHaveAccessibleDescription(problem);
    expect(router.state.location.pathname).toBe(`${DETAILS}/edit`);
  });

  it('keeps the typed values after a conflict and saves again with the new version (FR-018)', async () => {
    let saves = 0;
    let reads = 0;
    const { user, api } = renderPanel(`${DETAILS}/edit`, [
      reply(
        'GET /api/v1/campaigns/{campaignId}',
        () => ({ body: campaign({ version: ++reads > 1 ? 4 : 3 }) }),
        { campaignId: 'C-1001' },
      ),
      reply(
        'PATCH /api/v1/campaigns/{campaignId}',
        () =>
          ++saves === 1
            ? {
                status: 409,
                body: {
                  error: {
                    code: 'version_conflict',
                    message: 'Not saved — Daniel Okoro changed this campaign',
                    conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-12T09:00:00Z' },
                    fields: [],
                  },
                },
              }
            : { body: campaign({ name: 'Helix relaunch', version: 5 }) },
        { campaignId: 'C-1001' },
      ),
    ]);
    const form = await findForm();

    const name = within(form).getByLabelText('Campaign name');
    await user.clear(name);
    await user.type(name, 'Helix relaunch');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(
      await within(form).findByText(/^Not saved — Daniel Okoro changed this campaign/),
    ).toBeInTheDocument();
    expect(name).toHaveValue('Helix relaunch');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(api.bodiesOf('PATCH /api/v1/campaigns/C-1001')).toHaveLength(2));
    expect(api.bodiesOf('PATCH /api/v1/campaigns/C-1001')).toEqual([
      expect.objectContaining({ version: 3 }),
      expect.objectContaining({ version: 4 }),
    ]);
  });

  it('lists the values a removed cluster clears, before saving, and guards unsaved changes (US3#4, FR-009, FR-018)', async () => {
    const { user, api } = renderPanel(`${DETAILS}/edit`, [
      reply(
        'GET /api/v1/campaigns/{campaignId}',
        {
          body: campaign({
            attributes: {
              'SA-CLUSTER': ['SO-RETAIL', 'SO-B2B'],
              'SA-REGION': ['SO-CHINA', 'SO-EU'],
            },
          }),
        },
        { campaignId: 'C-1001' },
      ),
      reply('PATCH /api/v1/campaigns/{campaignId}', { body: campaign() }, { campaignId: 'C-1001' }),
    ]);
    const form = await findForm();

    await user.click(within(form).getByRole('checkbox', { name: 'B2B' }));
    expect(within(form).getByText(/'B2B – EUROPE' will be cleared/)).toBeInTheDocument();

    // Closing asks first.
    await user.click(within(form).getByRole('button', { name: 'Close' }));
    const question = await screen.findByRole('alertdialog', { name: 'Discard changes?' });
    await user.click(within(question).getByRole('button', { name: 'Keep editing' }));

    await user.click(within(form).getByRole('button', { name: 'Save' }));
    const clear = await screen.findByRole('alertdialog', {
      name: 'Clear values that no longer fit?',
    });
    expect(within(clear).getByText('Region: B2B – EUROPE')).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/v1/campaigns/C-1001')).toHaveLength(0);
    await user.click(within(clear).getByRole('button', { name: 'Clear and save' }));

    await waitFor(() => expect(api.bodiesOf('PATCH /api/v1/campaigns/C-1001')).toHaveLength(1));
  });

  it('goes back to the details on Cancel when nothing changed', async () => {
    const { user, router } = renderPanel(`${DETAILS}/edit`);
    const form = await findForm();

    await user.click(within(form).getByRole('button', { name: 'Cancel' }));

    await waitFor(() => expect(router.state.location.pathname).toBe(DETAILS));
  });

  it('shows a failed save that is not about a field as a message and keeps the values', async () => {
    const { user, router } = renderPanel(`${DETAILS}/edit`, [
      reply(
        'PATCH /api/v1/campaigns/{campaignId}',
        {
          status: 500,
          body: {
            error: {
              code: 'internal_error',
              message: 'Something went wrong. Please try again.',
              fields: [],
            },
          },
        },
        { campaignId: 'C-1001' },
      ),
    ]);
    const form = await findForm();

    await user.type(within(form).getByLabelText('Campaign name'), ' 2');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    await waitFor(() =>
      expect(screen.getByRole('region', { name: 'Messages (F8)', hidden: true })).toHaveTextContent(
        'Something went wrong. Please try again.',
      ),
    );
    expect(within(form).getByLabelText('Campaign name')).toHaveValue('Helix launch 2');
    expect(router.state.location.pathname).toBe(`${DETAILS}/edit`);
  });
});
