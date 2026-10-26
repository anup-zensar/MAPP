import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { campaignForm, LUBRICANTS } from '@/testing/attributeForms';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
import { campaign, PEOPLE, planAnswer, planCampaigns } from './fixtures';
import { NewCampaignPanel } from './NewCampaignPanel';

const NEW_PATH = '/plans/P-1001/campaigns/new';

function renderPanel(answers: Record<string, MockHandler> = {}) {
  const api = mockApi({
    'GET /api/v1/plans/P-1001/campaigns': { body: planCampaigns() },
    'GET /api/v1/plans/P-1001': { body: planAnswer([{ id: LUBRICANTS, label: 'Lubricants' }]) },
    'GET /api/v1/shell-attribute-forms/campaign': { body: campaignForm() },
    'GET /api/v1/people': { body: PEOPLE },
    ...answers,
  });
  const rendered = renderWithProviders(<NewCampaignPanel />, {
    route: NEW_PATH,
    path: '/plans/:planId/campaigns/new',
    user: testUser(),
  });
  return { api, ...rendered };
}

/** The panel once its form is shown (the loading panel is another one). */
async function findPanel() {
  await screen.findByLabelText('Campaign name');
  // The owner field loads the people itself.
  await screen.findByText('From the plan');
  return screen.getByRole('dialog', { name: 'New campaign' });
}

describe('NewCampaignPanel (feature 005, US1)', () => {
  it('names the plan and pre-fills the plan’s owner and year, labelled (US1#1, FR-001, FR-003)', async () => {
    renderPanel();
    const panel = await findPanel();

    expect(within(panel).getByText("Inside plan 'FY2026 Lubricants Europe'")).toBeInTheDocument();
    expect(within(panel).getByText(/Priya Raman/)).toBeInTheDocument();
    expect(within(panel).getByLabelText('Start date')).toHaveValue('2026-01-01');
    expect(within(panel).getByLabelText('End date')).toHaveValue('2026-12-31');
    expect(within(panel).getByText("The plan's year")).toBeInTheDocument();
    // Yes/no attributes start as No; the plan's business unit shows its section.
    expect(within(panel).getByRole('checkbox', { name: 'SMP' })).not.toBeChecked();
    expect(within(panel).getByRole('checkbox', { name: 'Shell Helix' })).toBeInTheDocument();
    expect(within(panel).getByRole('button', { name: 'Save' })).toBeEnabled();
    expect(within(panel).getByRole('button', { name: 'Save and add another' })).toBeEnabled();
    expect(document.title).toBe('New campaign · MAPP');
  });

  it('drops a default’s label once it is changed (FR-003)', async () => {
    renderPanel();
    const panel = await findPanel();

    fireEvent.change(within(panel).getByLabelText('End date'), { target: { value: '2026-06-30' } });

    expect(within(panel).queryByText("The plan's year")).not.toBeInTheDocument();
  });

  it('locks Region until a cluster is chosen and fills in a single region (US1#2, FR-009)', async () => {
    const { user } = renderPanel();
    const panel = await findPanel();

    expect(within(panel).getByText('Choose a cluster first.')).toBeInTheDocument();
    await user.click(within(panel).getByRole('checkbox', { name: 'Retail' }));

    expect(
      within(panel).getByText("Region set to 'RETAIL – CHINA', the only region for this cluster."),
    ).toBeInTheDocument();
    expect(within(panel).getByRole('checkbox', { name: 'RETAIL – CHINA' })).toBeChecked();
  });

  it('follows the typed budget in the remaining plan budget, red when over-allocated (US1#3, FR-010)', async () => {
    const { user, api } = renderPanel({
      'POST /api/v1/plans/P-1001/campaigns': { status: 201, body: campaign({ id: 'C-1005' }) },
    });
    const panel = await findPanel();
    expect(within(panel).getByText('$200,000.00')).toBeInTheDocument();

    await user.type(within(panel).getByLabelText('Campaign budget'), '300000');

    expect(within(panel).getByText('-$100,000.00')).toBeInTheDocument();
    expect(within(panel).getByText('Over-allocated by $100,000.00')).toBeInTheDocument();

    await user.type(within(panel).getByLabelText('Campaign name'), 'Fleet growth');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));
    await waitFor(() =>
      expect(api.bodiesOf('POST /api/v1/plans/P-1001/campaigns')).toHaveLength(1),
    );
  });

  it('says when the plan has no budget to allocate from (FR-010)', async () => {
    renderPanel({
      'GET /api/v1/plans/P-1001/campaigns': {
        body: planCampaigns([], { planBudget: null, remainingPlanBudget: null }),
      },
    });
    const panel = await findPanel();

    expect(within(panel).getByText('No budget to allocate from')).toBeInTheDocument();
  });

  it('saves, announces the campaign with View and highlights its row (US1#4, FR-004)', async () => {
    const { user, api, router } = renderPanel({
      'POST /api/v1/plans/P-1001/campaigns': {
        status: 201,
        body: campaign({ id: 'C-1005', name: 'Fleet growth' }),
      },
    });
    const panel = await findPanel();

    await user.type(within(panel).getByLabelText('Campaign name'), 'Fleet growth');
    await user.type(within(panel).getByLabelText('Campaign budget'), '12,500.5');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001'));
    expect(router.state.location.state).toEqual({ highlight: 'C-1005' });
    expect(api.bodiesOf('POST /api/v1/plans/P-1001/campaigns')).toEqual([
      {
        name: 'Fleet growth',
        description: '',
        ownerId: 'U-1002',
        startDate: '2026-01-01',
        endDate: '2026-12-31',
        budget: { amount: '12500.50', currency: 'USD' },
        attributes: { 'SA-SMP': false },
      },
    ]);
    // The panel has closed; the messages area is no longer hidden behind it.
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    const messages = await findMessages();
    expect(messages).toHaveTextContent("Campaign 'Fleet growth' created");
    await user.click(within(messages).getByRole('button', { name: 'View' }));
    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/campaigns/C-1005'),
    );
  });

  it('saves and opens a fresh panel for the same plan (FR-001 Save and add another)', async () => {
    const { user, router } = renderPanel({
      'POST /api/v1/plans/P-1001/campaigns': {
        status: 201,
        body: campaign({ id: 'C-1005', name: 'Fleet growth' }),
      },
    });
    const panel = await findPanel();

    await user.type(within(panel).getByLabelText('Campaign name'), 'Fleet growth');
    await user.click(within(panel).getByRole('button', { name: 'Save and add another' }));

    await waitFor(() => expect(router.state.location.state).toEqual({ highlight: 'C-1005' }));
    expect(router.state.location.pathname).toBe(NEW_PATH);
    await waitFor(() => expect(screen.getByLabelText('Campaign name')).toHaveValue(''));
  });

  it('shows every problem of a refused save at once, the summary focused (US1#5, FR-005)', async () => {
    const { user } = renderPanel({
      'POST /api/v1/plans/P-1001/campaigns': {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Some fields need your attention.',
            fields: [
              {
                field: 'name',
                message: "'FY2026 Lubricants Europe' already has a campaign called 'Helix launch'.",
              },
              {
                field: 'endDate',
                message: 'End date must be on or after the start date (01 Jan 2026).',
              },
            ],
          },
        },
      },
    });
    const panel = await findPanel();

    await user.type(within(panel).getByLabelText('Campaign name'), 'helix launch');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    const summary = await within(panel).findByText('2 problems need your attention');
    await waitFor(() => expect(summary.closest('[tabindex="-1"]')).toHaveFocus());
    expect(within(panel).getByLabelText('Campaign name')).toHaveAccessibleDescription(
      expect.stringContaining('already has a campaign called'),
    );
    expect(within(panel).getByLabelText('End date')).toHaveAttribute('aria-invalid', 'true');
    // The links lead to the fields.
    await user.click(
      within(panel).getByRole('link', {
        name: 'End date must be on or after the start date (01 Jan 2026).',
      }),
    );
    expect(within(panel).getByLabelText('End date')).toHaveFocus();
  });

  it('warns when the dates leave the plan’s year, and saves anyway (US1#5, FR-008)', async () => {
    const { user, api } = renderPanel({
      'POST /api/v1/plans/P-1001/campaigns': { status: 201, body: campaign({ id: 'C-1005' }) },
    });
    const panel = await findPanel();

    fireEvent.change(within(panel).getByLabelText('End date'), { target: { value: '2027-03-31' } });
    expect(
      within(panel).getByText("This campaign runs outside 2026, the plan's year"),
    ).toBeInTheDocument();

    await user.type(within(panel).getByLabelText('Campaign name'), 'Long run');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));
    await waitFor(() =>
      expect(api.bodiesOf('POST /api/v1/plans/P-1001/campaigns')).toEqual([
        expect.objectContaining({ endDate: '2027-03-31' }),
      ]),
    );
  });

  it('asks before discarding typed values on Cancel (FR-001)', async () => {
    const { user, router } = renderPanel();
    const panel = await findPanel();

    await user.type(within(panel).getByLabelText('Campaign name'), 'Half typed');
    await user.click(within(panel).getByRole('button', { name: 'Cancel' }));

    const question = await screen.findByRole('alertdialog', { name: 'Discard changes?' });
    expect(within(question).getByRole('button', { name: 'Keep editing' })).toHaveFocus();
    await user.click(within(question).getByRole('button', { name: 'Discard changes' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001'));
  });

  it('closes at once on Cancel when nothing was typed', async () => {
    const { user, router } = renderPanel();
    const panel = await findPanel();

    await user.click(within(panel).getByRole('button', { name: 'Cancel' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001'));
  });

  it('shows a loading state, then says when the plan is gone (parent gone)', async () => {
    const { user, router } = renderPanel({
      'GET /api/v1/plans/P-1001/campaigns': {
        status: 404,
        body: {
          error: {
            code: 'not_found',
            message: 'This plan no longer exists — someone may have deleted it.',
          },
        },
      },
    });

    const panel = await screen.findByRole('dialog', { name: 'New campaign' });
    expect(
      await within(panel).findByText('This plan no longer exists — someone may have deleted it.'),
    ).toBeInTheDocument();
    await user.click(within(panel).getByRole('button', { name: 'Back to the plan' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001'));
  });

  it('shows a failure with Try again when the plan cannot be read', async () => {
    let calls = 0;
    const { user } = renderPanel({
      'GET /api/v1/plans/P-1001/campaigns': () =>
        ++calls === 1
          ? { status: 500, body: { error: { code: 'internal_error', message: 'Oops' } } }
          : {
              body: planCampaigns(),
            },
    });

    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByLabelText('Campaign name')).toBeInTheDocument();
  });
});
