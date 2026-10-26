import { screen, waitFor, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { reply } from '@/testing/replies';
import type { RecordTarget } from './campaign-rows';
import { CopyDialog } from './CopyDialog';
import {
  campaign,
  destination,
  destinationList,
  initiative,
  planDestination,
  usd,
} from './fixtures';

const HELIX: RecordTarget = {
  level: 'campaign',
  id: 'C-1001',
  name: 'Helix launch',
  planId: 'P-1001',
  year: 2026,
  version: 3,
  counts: { initiatives: 2, activities: 7, approvedActivities: 4, lineItems: 15 },
};

const DIGITAL: RecordTarget = {
  level: 'initiative',
  id: 'I-1001',
  name: 'Digital & social',
  planId: 'P-1001',
  year: 2026,
  version: 3,
  counts: { activities: 0, approvedActivities: 0, lineItems: 0 },
};

const CAMPAIGN_COPY_RULE = 'A copy can go into this plan or another plan of the same year';

const CAMPAIGN_DESTINATIONS = {
  body: destinationList(
    [
      planDestination({
        id: 'P-1001',
        name: 'FY2026 Lubricants Europe',
        current: true,
        planStatus: 'published',
        suggestedName: 'Helix launch (copy 2)',
        overAllocatedBy: usd('50000.00'),
      }),
      planDestination({ suggestedName: 'Helix launch (copy)' }),
    ],
    CAMPAIGN_COPY_RULE,
  ),
};

/** The dialog as a row or panel holds it: closing it forgets the target. */
function Harness({ target }: { target: RecordTarget }) {
  const [shown, setShown] = useState<RecordTarget | null>(target);
  return shown ? <CopyDialog target={shown} onClose={() => setShown(null)} /> : <p>Closed</p>;
}

function renderDialog(target: RecordTarget, answers: HttpHandler[]) {
  const api = mockApi(answers);
  const result = renderWithProviders(<Harness target={target} />, {
    user: testUser(),
    route: '/plans/P-1001',
  });
  return { ...result, api };
}

describe('CopyDialog (feature 005, US6#4–5, FR-026, FR-027)', () => {
  it('suggests the name in this plan, with the counts, R13 and the budget warning (US6#4)', async () => {
    renderDialog(HELIX, [
      reply('GET /api/v1/campaigns/{campaignId}/destinations', CAMPAIGN_DESTINATIONS, {
        campaignId: 'C-1001',
      }),
    ]);

    const dialog = screen.getByRole('dialog', { name: "Copy campaign 'Helix launch'" });
    const name = await within(dialog).findByRole('textbox', { name: /Name of the copy/ });
    expect(name).toHaveValue('Helix launch (copy 2)');
    expect(name).toHaveFocus();
    const into = within(dialog).getByRole('group', { name: 'Copy into' });
    const current = within(into).getByRole('radio', { name: 'FY2026 Lubricants Europe' });
    expect(current).toBeChecked();
    expect(current).toHaveAccessibleDescription('Current Published');
    expect(dialog).toHaveTextContent('It copies 2 initiatives, 7 activities and 15 line items.');
    expect(dialog).toHaveTextContent(
      'Copies get new IDs. Activities start as Created, without approvals, comments or history.',
    );
    expect(dialog).toHaveTextContent(
      "This will over-allocate 'FY2026 Lubricants Europe' by $50,000.00.",
    );
  });

  it('follows the chosen plan’s suggestion until the name is typed (FR-026)', async () => {
    const { user } = renderDialog(HELIX, [
      reply('GET /api/v1/campaigns/{campaignId}/destinations', CAMPAIGN_DESTINATIONS, {
        campaignId: 'C-1001',
      }),
    ]);

    const name = await screen.findByRole('textbox', { name: /Name of the copy/ });
    await user.click(screen.getByRole('radio', { name: 'FY2026 Mobility China' }));
    expect(name).toHaveValue('Helix launch (copy)');
    expect(screen.queryByText(/This will over-allocate/)).not.toBeInTheDocument();

    await user.clear(name);
    await user.type(name, 'Helix relaunch');
    await user.click(screen.getByRole('radio', { name: 'FY2026 Lubricants Europe' }));
    expect(name).toHaveValue('Helix relaunch');
  });

  it('copies and opens the plan with the copy highlighted and a message with View (US6#5, FR-027)', async () => {
    const { user, api, router } = renderDialog(HELIX, [
      reply('GET /api/v1/campaigns/{campaignId}/destinations', CAMPAIGN_DESTINATIONS, {
        campaignId: 'C-1001',
      }),
      reply(
        'POST /api/v1/campaigns/{campaignId}/copy',
        {
          status: 201,
          body: campaign({ id: 'C-1009', planId: 'P-1002', name: 'Helix launch (copy)' }),
        },
        { campaignId: 'C-1001' },
      ),
    ]);

    await screen.findByRole('textbox', { name: /Name of the copy/ });
    await user.click(screen.getByRole('radio', { name: 'FY2026 Mobility China' }));
    await user.click(screen.getByRole('button', { name: 'Copy campaign' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1002'));
    expect(router.state.location.state).toEqual({ highlight: 'C-1009' });
    expect(api.bodiesOf('POST /api/v1/campaigns/C-1001/copy')).toEqual([
      { planId: 'P-1002', name: 'Helix launch (copy)' },
    ]);
    const messages = await findMessages();
    expect(messages).toHaveTextContent("Campaign 'Helix launch (copy)' created");
    expect(screen.getByText('Closed')).toBeInTheDocument();

    await user.click(within(messages).getByRole('button', { name: 'View' }));
    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1002/campaigns/C-1009'),
    );
  });

  it('shows the API’s problems under the name and the destinations', async () => {
    const { user } = renderDialog(HELIX, [
      reply('GET /api/v1/campaigns/{campaignId}/destinations', CAMPAIGN_DESTINATIONS, {
        campaignId: 'C-1001',
      }),
      reply(
        'POST /api/v1/campaigns/{campaignId}/copy',
        {
          status: 400,
          body: {
            error: {
              code: 'validation_failed',
              message: 'Some fields need your attention.',
              fields: [
                {
                  field: 'name',
                  message:
                    "'FY2026 Lubricants Europe' already has a campaign called 'Helix launch (copy 2)'.",
                },
                {
                  field: 'planId',
                  message: 'This plan no longer exists — someone may have deleted it.',
                },
              ],
            },
          },
        },
        { campaignId: 'C-1001' },
      ),
    ]);

    await screen.findByRole('textbox', { name: /Name of the copy/ });
    await user.click(screen.getByRole('button', { name: 'Copy campaign' }));

    expect(
      await screen.findByText(
        "'FY2026 Lubricants Europe' already has a campaign called 'Helix launch (copy 2)'.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('group', { name: 'Copy into' })).toHaveAccessibleDescription(
      'This plan no longer exists — someone may have deleted it.',
    );
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('copies an initiative with nothing below it into its campaign', async () => {
    const { user, api, router } = renderDialog(DIGITAL, [
      reply(
        'GET /api/v1/initiatives/{initiativeId}/destinations',
        {
          body: destinationList(
            [
              destination({
                id: 'C-1001',
                name: 'Helix launch',
                current: true,
                suggestedName: 'Digital & social (copy)',
              }),
            ],
            'A copy can go into this campaign or another campaign of the same plan',
          ),
        },
        { initiativeId: 'I-1001' },
      ),
      reply(
        'POST /api/v1/initiatives/{initiativeId}/copy',
        {
          status: 201,
          body: initiative({ id: 'I-1009', name: 'Digital & social (copy)' }),
        },
        { initiativeId: 'I-1001' },
      ),
    ]);

    const dialog = await screen.findByRole('dialog', {
      name: "Copy initiative 'Digital & social'",
    });
    expect(
      await within(dialog).findByText("It copies the initiative only — there's nothing below it."),
    ).toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: 'Copy initiative' }));

    await waitFor(() => expect(router.state.location.state).toEqual({ highlight: 'I-1009' }));
    expect(api.bodiesOf('POST /api/v1/initiatives/I-1001/copy')).toEqual([
      { campaignId: 'C-1001', name: 'Digital & social (copy)' },
    ]);
    expect(await findMessages()).toHaveTextContent("Initiative 'Digital & social (copy)' created");
  });

  it('shows a failure that is not about a field in the dialog', async () => {
    const { user } = renderDialog(HELIX, [
      reply('GET /api/v1/campaigns/{campaignId}/destinations', CAMPAIGN_DESTINATIONS, {
        campaignId: 'C-1001',
      }),
      reply(
        'POST /api/v1/campaigns/{campaignId}/copy',
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

    await screen.findByRole('textbox', { name: /Name of the copy/ });
    await user.click(screen.getByRole('button', { name: 'Copy campaign' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'This campaign no longer exists — someone may have deleted it.',
    );
  });
});
