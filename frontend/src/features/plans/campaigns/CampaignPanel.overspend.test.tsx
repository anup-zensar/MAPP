import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { DETAILS, findForm, renderPanel } from '@/testing/fixtures/campaign-panel';
import { budgetAndCost, campaign, usd } from '@/testing/fixtures/campaigns';
import { apiError, reply } from '@/testing/replies';

const OVERSPEND = 'When planned cost goes over this budget';
const PATCH = 'PATCH /api/v1/campaigns/C-1001';

function withPlanned(overrides: Parameters<typeof campaign>[0] = {}) {
  return reply(
    'GET /api/v1/campaigns/{campaignId}',
    {
      body: campaign({
        budgetAndCost: budgetAndCost({ plannedCost: usd('380000.00') }),
        ...overrides,
      }),
    },
    { campaignId: 'C-1001' },
  );
}

describe('CampaignPanel — overspend control (037 US1)', () => {
  it('offers Warn or Refuse and saves Refuse with the form (037 FR-001, FR-002)', async () => {
    const { user, api } = renderPanel(`${DETAILS}/edit`, [
      withPlanned(),
      reply('PATCH /api/v1/campaigns/{campaignId}', { body: campaign() }, { campaignId: 'C-1001' }),
    ]);
    const form = await findForm();
    const group = within(form).getByRole('group', { name: OVERSPEND });
    expect(within(group).getByRole('radio', { name: /^Warn/ })).toBeChecked();
    await expectNoAxeViolations();

    await user.click(within(group).getByRole('radio', { name: /^Refuse/ }));
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(api.bodiesOf(PATCH)).toHaveLength(1));
    expect(api.bodiesOf(PATCH)[0]).toMatchObject({ overspend: 'refuse' });
  });

  it('says before saving that a Refuse budget cannot go below its planned cost (037 FR-015)', async () => {
    const { user, api } = renderPanel(`${DETAILS}/edit`, [withPlanned()]);
    const form = await findForm();
    await user.click(within(form).getByRole('radio', { name: /^Refuse/ }));
    const budget = within(form).getByLabelText('Campaign budget');
    await user.clear(budget);
    await user.type(budget, '300000');
    const refused =
      "The planned cost is 380,000.00 USD; this budget refuses overspending, so it can't be lower than that.";

    expect(within(form).getByText(refused)).toBeVisible();
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await within(form).findAllByText(refused)).not.toHaveLength(0);
    expect(api.bodiesOf(PATCH)).toHaveLength(0);
  });

  it('shows the setting read-only with the reason for someone who may not change it (037 FR-003)', async () => {
    const reason = "Only the budget's owner or someone who manages budgets can change this.";
    renderPanel(`${DETAILS}/edit`, [
      withPlanned({ overspend: { mode: 'refuse', canChange: { available: false, reason } } }),
    ]);
    const form = await findForm();

    expect(within(form).getByRole('radio', { name: /^Refuse/ })).toBeDisabled();
    expect(within(form).getByText(reason)).toBeVisible();
  });

  it('shows the API refusal in the panel and keeps the typed values (037 FR-016)', async () => {
    const message =
      "Not saved — the planned cost of plan 'FY2026 Lubricants Europe' would be 1,250,000.00 USD, over its budget of 1,200,000.00 USD by 50,000.00 USD. Its owner refuses overspending.";
    const { user } = renderPanel(`${DETAILS}/edit`, [
      withPlanned(),
      reply(
        'PATCH /api/v1/campaigns/{campaignId}',
        { status: 409, body: apiError('over_budget', message) },
        { campaignId: 'C-1001' },
      ),
    ]);
    const form = await findForm();
    const budget = within(form).getByLabelText('Campaign budget');
    await user.clear(budget);
    await user.type(budget, '450000');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText(message)).toBeVisible();
    expect(budget).toHaveValue('450,000.00');
  });
});
