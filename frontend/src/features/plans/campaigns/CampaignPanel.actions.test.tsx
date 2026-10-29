import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DETAILS, findDetails, renderPanel } from '@/testing/fixtures/campaign-panel';
import { campaign, destinationList, planDestination } from '@/testing/fixtures/campaigns';
import { findMessages } from '@/testing/render';
import { reply } from '@/testing/replies';

describe('CampaignPanel — Delete, Move to… and Copy (feature 005, US5, US6)', () => {
  it('deletes from More, closes the panel and offers Undo (US5, FR-028, FR-029)', async () => {
    const { user, router, api } = renderPanel(DETAILS, [
      reply(
        'DELETE /api/v1/campaigns/{campaignId}',
        {
          body: {
            recycleBinEntry: {
              id: 'BIN-1001',
              recordType: 'campaign',
              recordTypeLabel: 'campaign',
              recordId: 'C-1001',
              recordName: 'Helix launch',
              path: [],
              parent: null,
              deletedAt: '2026-03-12T09:00:00Z',
              deletedBy: { id: 'U-1002', name: 'Priya Raman' },
              contents: [],
            },
          },
        },
        { campaignId: 'C-1001' },
      ),
    ]);
    const panel = await findDetails();

    await user.click(within(panel).getByRole('button', { name: 'Actions for Helix launch' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete campaign 'Helix launch'?",
    });
    expect(dialog).toHaveTextContent('With it go 1 initiative.');
    await user.click(within(dialog).getByRole('button', { name: 'Delete campaign' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001'));
    expect(api.bodiesOf('DELETE /api/v1/campaigns/C-1001')).toHaveLength(1);
    expect(await findMessages()).toHaveTextContent(
      "Campaign 'Helix launch' deleted",
    );
  });

  it('moves from More and opens the plan it moved to, with the campaign highlighted (US6, FR-025)', async () => {
    const { user, router, api } = renderPanel(DETAILS, [
      reply(
        'GET /api/v1/campaigns/{campaignId}/destinations',
        {
          body: destinationList(
            [planDestination()],
            'Campaigns move only between plans of the same year',
          ),
        },
        { campaignId: 'C-1001' },
      ),
      reply(
        'POST /api/v1/campaigns/{campaignId}/move',
        { body: campaign({ planId: 'P-1002', version: 4 }) },
        { campaignId: 'C-1001' },
      ),
    ]);
    const panel = await findDetails();

    await user.click(within(panel).getByRole('button', { name: 'Actions for Helix launch' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Move to…' }));
    const dialog = await screen.findByRole('dialog', { name: "Move campaign 'Helix launch'" });
    await user.click(await within(dialog).findByRole('radio', { name: 'FY2026 Mobility China' }));
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

  it('copies from More (US6, FR-026)', async () => {
    const { user } = renderPanel(DETAILS, [
      reply(
        'GET /api/v1/campaigns/{campaignId}/destinations',
        {
          body: destinationList(
            [
              planDestination({
                id: 'P-1001',
                name: 'FY2026 Lubricants Europe',
                current: true,
                suggestedName: 'Helix launch (copy)',
              }),
            ],
            'A copy can go into this plan or another plan of the same year',
          ),
        },
        { campaignId: 'C-1001' },
      ),
    ]);
    const panel = await findDetails();

    await user.click(within(panel).getByRole('button', { name: 'Actions for Helix launch' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Copy' }));

    const dialog = await screen.findByRole('dialog', { name: "Copy campaign 'Helix launch'" });
    expect(await within(dialog).findByRole('textbox', { name: /Name of the copy/ })).toHaveValue(
      'Helix launch (copy)',
    );
    expect(dialog).toHaveTextContent('It copies 1 initiative.');
  });
});
