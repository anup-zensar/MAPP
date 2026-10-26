import { screen, waitFor, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { findMessages, renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import type { RecordTarget } from './campaignRows';
import {
  campaign,
  destination,
  destinationList,
  initiative,
  planDestination,
  usd,
} from './fixtures';
import { MoveDialog } from './MoveDialog';

const DIGITAL: RecordTarget = {
  level: 'initiative',
  id: 'I-1001',
  name: 'Digital & social',
  planId: 'P-1001',
  year: 2026,
  version: 3,
  counts: { activities: 3, approvedActivities: 1, lineItems: 6 },
};

const HELIX: RecordTarget = {
  level: 'campaign',
  id: 'C-1001',
  name: 'Helix launch',
  planId: 'P-1001',
  year: 2026,
  version: 3,
  counts: { initiatives: 1, activities: 0, approvedActivities: 0, lineItems: 0 },
};

const INITIATIVE_RULE = 'Initiatives move only between campaigns of the same plan';
const CAMPAIGN_RULE = 'Campaigns move only between plans of the same year';
const DATES_DONT_FIT =
  "Dates don't fit (01 Mar – 31 Dec 2026) — change the initiative's dates first";

/** The dialog as a row or panel holds it: closing it forgets the target. */
function Harness({ target }: { target: RecordTarget }) {
  const [shown, setShown] = useState<RecordTarget | null>(target);
  return shown ? <MoveDialog target={shown} onClose={() => setShown(null)} /> : <p>Closed</p>;
}

function renderDialog(target: RecordTarget, answers: Record<string, MockHandler>) {
  const api = mockApi(answers);
  const result = renderWithProviders(<Harness target={target} />, {
    user: testUser(),
    route: '/plans/P-1001/initiatives/I-1001',
  });
  return { ...result, api };
}

const INITIATIVE_DESTINATIONS = {
  body: destinationList(
    [
      destination({
        id: 'C-1003',
        name: 'Helix 2',
        unavailableReason: DATES_DONT_FIT,
        available: false,
      }),
      destination(),
    ],
    INITIATIVE_RULE,
  ),
};

describe('MoveDialog (feature 005, US6#1–3, FR-023 – FR-025)', () => {
  it('moves an initiative to another campaign and opens the plan with it highlighted (US6#1, FR-025)', async () => {
    const { user, api, router } = renderDialog(DIGITAL, {
      'GET /api/v1/initiatives/I-1001/destinations': INITIATIVE_DESTINATIONS,
      'POST /api/v1/initiatives/I-1001/move': {
        body: initiative({ campaignId: 'C-1002', version: 4 }),
      },
    });

    const dialog = screen.getByRole('dialog', { name: "Move initiative 'Digital & social'" });
    expect(dialog).toHaveAccessibleDescription(
      'Choose the campaign to move it to. Its activities and line items move with it and keep their IDs, dates and statuses.',
    );
    expect(await within(dialog).findByText(INITIATIVE_RULE)).toBeInTheDocument();
    const rimula = within(dialog).getByRole('radio', { name: 'Rimula fleet' });
    expect(rimula).toHaveAccessibleDescription('01 Mar – 31 Dec 2026');

    await user.click(rimula);
    await user.click(within(dialog).getByRole('button', { name: 'Move initiative' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001'));
    expect(router.state.location.state).toEqual({ highlight: 'I-1001' });
    expect(api.bodiesOf('POST /api/v1/initiatives/I-1001/move')).toEqual([
      { campaignId: 'C-1002', version: 3 },
    ]);
    const messages = await findMessages();
    expect(messages).toHaveTextContent("Initiative 'Digital & social' moved to 'Rimula fleet'");
    expect(screen.getByText('Closed')).toBeInTheDocument();

    await user.click(within(messages).getByRole('button', { name: 'View' }));
    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/initiatives/I-1001'),
    );
  });

  it('lists a destination that can’t take it, with the reason and the fix (US6#2, FR-023)', async () => {
    const { user, api } = renderDialog(DIGITAL, {
      'GET /api/v1/initiatives/I-1001/destinations': INITIATIVE_DESTINATIONS,
    });

    const helix2 = await screen.findByRole('radio', { name: 'Helix 2' });
    expect(helix2).toBeDisabled();
    expect(helix2).toHaveAccessibleDescription(`01 Mar – 31 Dec 2026 ${DATES_DONT_FIT}`);

    await user.click(screen.getByRole('button', { name: 'Move initiative' }));
    expect(await screen.findByText('Choose where to move it.')).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/initiatives/I-1001/move')).toEqual([]);

    await user.click(screen.getByRole('radio', { name: 'Rimula fleet' }));
    expect(screen.queryByText('Choose where to move it.')).not.toBeInTheDocument();
  });

  it('offers a campaign only plans of its year, with the consequences and the budget warning (US6#3, FR-024)', async () => {
    const { user, api, router } = renderDialog(HELIX, {
      'GET /api/v1/campaigns/C-1001/destinations': {
        body: destinationList(
          [
            planDestination({
              overAllocatedBy: usd('50000.00'),
              consequences: {
                modelName: 'Mobility 1.0',
                modelChanges: true,
                businessUnitChanges: true,
                typesNotAllowed: [{ id: 'A-1042', name: 'Launch film', typeName: 'Event' }],
                groupsHidden: ['Lubricants attributes (activities)'],
                groupsShown: ['Mobility attributes (activities)'],
              },
            }),
            planDestination({ id: 'P-1004', name: 'FY2026 Retail', planStatus: 'published' }),
          ],
          CAMPAIGN_RULE,
        ),
      },
      'POST /api/v1/campaigns/C-1001/move': { body: campaign({ planId: 'P-1002', version: 4 }) },
    });

    const dialog = await screen.findByRole('dialog', { name: "Move campaign 'Helix launch'" });
    expect(await within(dialog).findByText(CAMPAIGN_RULE)).toBeInTheDocument();
    expect(
      within(dialog).getByRole('radio', { name: 'FY2026 Retail' }),
    ).toHaveAccessibleDescription('Published');
    const china = within(dialog).getByRole('radio', { name: 'FY2026 Mobility China' });
    expect(china).toHaveAccessibleDescription('Draft');
    expect(dialog).not.toHaveTextContent('uses another model or business unit');

    await user.click(china);

    expect(dialog).toHaveTextContent(
      "This will over-allocate 'FY2026 Mobility China' by $50,000.00.",
    );
    expect(dialog).toHaveTextContent(
      "'FY2026 Mobility China' uses another model or business unit:",
    );
    expect(dialog).toHaveTextContent(
      "These activities have a type the model 'Mobility 1.0' doesn't allow. They keep their type but need an allowed type before they can be submitted: Launch film (A-1042)",
    );
    expect(dialog).toHaveTextContent("Goals will be recalculated with the model 'Mobility 1.0'.");
    expect(dialog).toHaveTextContent(
      'Hidden there, values kept: Lubricants attributes (activities)',
    );
    expect(dialog).toHaveTextContent('Shown there: Mobility attributes (activities)');

    await user.click(within(dialog).getByRole('button', { name: 'Move campaign' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1002'));
    expect(router.state.location.state).toEqual({ highlight: 'C-1001' });
    expect(api.bodiesOf('POST /api/v1/campaigns/C-1001/move')).toEqual([
      { planId: 'P-1002', version: 3 },
    ]);
    expect(await findMessages()).toHaveTextContent(
      "Campaign 'Helix launch' moved to 'FY2026 Mobility China'",
    );
  });

  it('says when there is nowhere to move it (edge case)', async () => {
    renderDialog(HELIX, {
      'GET /api/v1/campaigns/C-1001/destinations': { body: destinationList([], CAMPAIGN_RULE) },
    });

    expect(
      await screen.findByText("There's no other plan in 2026 to move this campaign to."),
    ).toBeInTheDocument();
    expect(screen.getByText(CAMPAIGN_RULE)).toBeInTheDocument();
    expect(screen.queryByRole('radio')).not.toBeInTheDocument();
  });

  it('shows a refusal in the dialog and asks for the destinations again', async () => {
    const { user, api } = renderDialog(DIGITAL, {
      'GET /api/v1/initiatives/I-1001/destinations': INITIATIVE_DESTINATIONS,
      'POST /api/v1/initiatives/I-1001/move': {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Some fields need your attention.',
            fields: [
              {
                field: 'campaignId',
                message: "Already has an initiative called 'Digital & social' — rename it first",
              },
            ],
          },
        },
      },
    });

    await user.click(await screen.findByRole('radio', { name: 'Rimula fleet' }));
    await user.click(screen.getByRole('button', { name: 'Move initiative' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "Already has an initiative called 'Digital & social' — rename it first",
    );
    await waitFor(() =>
      expect(
        api.requests.filter(
          (request) => request.path === '/api/v1/initiatives/I-1001/destinations',
        ),
      ).toHaveLength(2),
    );
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  it('says who changed the record meanwhile (409)', async () => {
    const { user } = renderDialog(DIGITAL, {
      'GET /api/v1/initiatives/I-1001/destinations': INITIATIVE_DESTINATIONS,
      'POST /api/v1/initiatives/I-1001/move': {
        status: 409,
        body: {
          error: {
            code: 'version_conflict',
            message: 'Not saved — Daniel Okoro changed this initiative',
            fields: [],
            conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-12T09:00:00Z' },
          },
        },
      },
    });

    await user.click(await screen.findByRole('radio', { name: 'Rimula fleet' }));
    await user.click(screen.getByRole('button', { name: 'Move initiative' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      /^Not saved — Daniel Okoro changed this initiative (at|on) /,
    );
  });

  it('shows grey lines while loading, then the error with Try again', async () => {
    let calls = 0;
    const { user } = renderDialog(DIGITAL, {
      'GET /api/v1/initiatives/I-1001/destinations': () =>
        ++calls === 1 ? { status: 500 } : INITIATIVE_DESTINATIONS,
    });

    expect(screen.getByRole('status')).toHaveTextContent('Loading…');
    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('radio', { name: 'Rimula fleet' })).toHaveFocus();
  });

  it('says when the record is gone, and Cancel closes', async () => {
    const { user } = renderDialog(DIGITAL, {
      'GET /api/v1/initiatives/I-1001/destinations': {
        status: 404,
        body: {
          error: {
            code: 'not_found',
            message: 'This initiative no longer exists — someone may have deleted it.',
            fields: [],
          },
        },
      },
    });

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'This initiative no longer exists — someone may have deleted it.',
    );
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(await screen.findByText('Closed')).toBeInTheDocument();
  });
});
