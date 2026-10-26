import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { campaignForm } from '@/testing/fixtures/attribute-forms';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { reply } from '@/testing/replies';
import { campaign, initiative, PEOPLE, planAnswer } from '@/testing/fixtures/campaigns';
import { NewInitiativePanel } from './NewInitiativePanel';

const NEW_PATH = '/plans/P-1001/campaigns/C-1001/initiatives/new';
const HELIX = campaign({ startDate: '2026-03-01', endDate: '2026-12-31' });

function renderPanel(answers: HttpHandler[] = []) {
  const api = mockApi([
    reply('GET /api/v1/campaigns/{campaignId}', { body: HELIX }, { campaignId: 'C-1001' }),
    reply('GET /api/v1/plans/{planId}', { body: planAnswer() }, { planId: 'P-1001' }),
    reply(
      'GET /api/v1/shell-attribute-forms/{level}',
      {
        body: { ...campaignForm(), level: 'initiative' },
      },
      { level: 'initiative' },
    ),
    reply('GET /api/v1/people', { body: PEOPLE }),
    ...answers,
  ]);
  const rendered = renderWithProviders(<NewInitiativePanel />, {
    route: NEW_PATH,
    path: '/plans/:planId/campaigns/:campaignId/initiatives/new',
    user: testUser(),
  });
  return { api, ...rendered };
}

/** The panel once its form is shown (the loading panel is another one). */
async function findPanel() {
  await screen.findByLabelText('Initiative name');
  await screen.findByText('From the campaign');
  return screen.getByRole('dialog', { name: 'New initiative' });
}

describe('NewInitiativePanel (feature 005, US2)', () => {
  it('fixes the campaign and takes owner and dates from it, labelled (US2#1, FR-002, FR-003)', async () => {
    renderPanel();
    const panel = await findPanel();

    expect(within(panel).getByText("Inside campaign 'Helix launch'")).toBeInTheDocument();
    const field = within(panel).getByLabelText('Campaign');
    expect(field).toHaveValue('Helix launch');
    expect(field).toHaveAttribute('readonly');
    expect(field).toHaveAccessibleDescription('Change it with Move');
    expect(within(panel).getByText(/Alex Morgan/)).toBeInTheDocument();
    expect(within(panel).getByLabelText('Start date')).toHaveValue('2026-03-01');
    expect(within(panel).getByLabelText('End date')).toHaveValue('2026-12-31');
    expect(within(panel).getByText("The campaign's dates")).toBeInTheDocument();
    expect(document.title).toBe('New initiative · MAPP');
  });

  it('keeps the dates inside the campaign’s and shows the API’s R3 message under them (US2#2, FR-006)', async () => {
    const r3 = "Dates must be inside the campaign's dates (01 Mar 2026 – 31 Dec 2026).";
    const { user } = renderPanel([
      reply(
        'POST /api/v1/campaigns/{campaignId}/initiatives',
        {
          status: 400,
          body: {
            error: {
              code: 'validation_failed',
              message: 'Some fields need your attention.',
              fields: [{ field: 'startDate', message: r3 }],
            },
          },
        },
        { campaignId: 'C-1001' },
      ),
    ]);
    const panel = await findPanel();

    expect(
      within(panel).getByText("Must be inside the campaign's dates (01 Mar 2026 – 31 Dec 2026)"),
    ).toBeInTheDocument();
    expect(within(panel).getByLabelText('Start date')).toHaveAttribute('min', '2026-03-01');
    expect(within(panel).getByLabelText('End date')).toHaveAttribute('max', '2026-12-31');

    await user.type(within(panel).getByLabelText('Initiative name'), 'Early bird');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findByText('1 problem needs your attention')).toBeInTheDocument();
    expect(within(panel).getByLabelText('Start date')).toHaveAccessibleDescription(r3);
  });

  it('starts without dates when the campaign has none, and does not limit them (edge case)', async () => {
    renderPanel([
      reply(
        'GET /api/v1/campaigns/{campaignId}',
        { body: campaign({ startDate: null, endDate: null }) },
        { campaignId: 'C-1001' },
      ),
    ]);
    const panel = await findPanel();

    expect(within(panel).getByLabelText('Start date')).toHaveValue('');
    expect(within(panel).getByLabelText('Start date')).not.toHaveAttribute('min');
    expect(within(panel).queryByText("The campaign's dates")).not.toBeInTheDocument();
    expect(within(panel).queryByText(/Must be inside/)).not.toBeInTheDocument();
  });

  it('follows the typed budget in the remaining campaign budget (US2#3, FR-010)', async () => {
    const { user } = renderPanel();
    const panel = await findPanel();
    expect(within(panel).getByText('Remaining campaign budget')).toBeInTheDocument();

    await user.type(within(panel).getByLabelText('Initiative budget'), '150000');

    expect(within(panel).getByText('-$50,000.00')).toBeInTheDocument();
    expect(within(panel).getByText('Over-allocated by $50,000.00')).toBeInTheDocument();
  });

  it('saves with a name only and highlights the new initiative (US2#1, FR-004)', async () => {
    const { user, api, router } = renderPanel([
      reply(
        'POST /api/v1/campaigns/{campaignId}/initiatives',
        {
          status: 201,
          body: initiative({ id: 'I-1009', name: 'Digital & social' }),
        },
        { campaignId: 'C-1001' },
      ),
    ]);
    const panel = await findPanel();

    await user.type(within(panel).getByLabelText('Initiative name'), 'Digital & social');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001'));
    expect(router.state.location.state).toEqual({ highlight: 'I-1009' });
    expect(api.bodiesOf('POST /api/v1/campaigns/C-1001/initiatives')).toEqual([
      {
        name: 'Digital & social',
        description: '',
        ownerId: 'U-1001',
        startDate: '2026-03-01',
        endDate: '2026-12-31',
        budget: null,
        attributes: { 'SA-SMP': false },
      },
    ]);
    // The panel has closed; the messages area is no longer hidden behind it.
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    const messages = await findMessages();
    expect(messages).toHaveTextContent("Initiative 'Digital & social' created");
    await user.click(within(messages).getByRole('button', { name: 'View' }));
    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/initiatives/I-1009'),
    );
  });

  it('saves and opens a fresh panel for the same campaign (Save and add another)', async () => {
    const { user, router } = renderPanel([
      reply(
        'POST /api/v1/campaigns/{campaignId}/initiatives',
        {
          status: 201,
          body: initiative({ id: 'I-1009' }),
        },
        { campaignId: 'C-1001' },
      ),
    ]);
    const panel = await findPanel();

    await user.type(within(panel).getByLabelText('Initiative name'), 'Digital & social');
    await user.click(within(panel).getByRole('button', { name: 'Save and add another' }));

    await waitFor(() => expect(router.state.location.state).toEqual({ highlight: 'I-1009' }));
    expect(router.state.location.pathname).toBe(NEW_PATH);
    await waitFor(() => expect(screen.getByLabelText('Initiative name')).toHaveValue(''));
  });

  it('says when the campaign is gone', async () => {
    renderPanel([
      reply(
        'GET /api/v1/campaigns/{campaignId}',
        {
          status: 404,
          body: {
            error: {
              code: 'not_found',
              message: 'This campaign no longer exists — someone may have deleted it.',
              fields: [],
            },
          },
        },
        { campaignId: 'C-1001' },
      ),
    ]);

    expect(
      await screen.findByText('This campaign no longer exists — someone may have deleted it.'),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Back to the plan' })).toBeInTheDocument();
  });
});
