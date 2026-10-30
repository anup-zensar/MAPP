import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderTree } from '@/testing/fixtures/campaign-tree';
import {
  campaign,
  destination,
  destinationList,
  initiative,
  planCampaigns,
  planDestination,
} from '@/testing/fixtures/campaigns';
import { findMessages } from '@/testing/render';
import { reply } from '@/testing/replies';

describe('useCampaignTreeContent: row actions (feature 005, US5, US6)', () => {
  it('deletes from a row’s ⋯ menu and the row leaves the tree at once (FR-021, FR-028, FR-012)', async () => {
    let deleted = false;
    const { user } = renderTree([
      reply(
        'GET /api/v1/plans/{planId}/campaigns',
        () => ({
          body: deleted ? planCampaigns([]) : planCampaigns(),
        }),
        { planId: 'P-1001' },
      ),
      reply(
        'DELETE /api/v1/initiatives/{initiativeId}',
        () => {
          deleted = true;
          return {
            body: {
              recycleBinEntry: {
                id: 'BIN-1002',
                recordType: 'initiative',
                recordTypeLabel: 'initiative',
                recordId: 'I-1001',
                recordName: 'Digital & social',
                path: [],
                parent: null,
                deletedAt: '2026-03-12T09:00:00Z',
                deletedBy: { id: 'U-1002', name: 'Priya Raman' },
                contents: [],
              },
            },
          };
        },
        { initiativeId: 'I-1001' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Actions for Digital & social' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete initiative 'Digital & social'?",
    });
    await user.click(within(dialog).getByRole('button', { name: 'Delete initiative' }));

    expect(await screen.findByTestId('empty')).toBeInTheDocument();
    expect(await findMessages()).toHaveTextContent("Initiative 'Digital & social' deleted");
  });

  it('copies a campaign from its row; the copy appears highlighted (US6#4–5, FR-026, FR-027)', async () => {
    const copy = campaign({
      id: 'C-1009',
      name: 'Helix launch (copy)',
      initiatives: [initiative({ id: 'I-1009', campaignId: 'C-1009' })],
    });
    let copied = false;
    const { user, api } = renderTree([
      reply(
        'GET /api/v1/plans/{planId}/campaigns',
        () => ({
          body: copied ? planCampaigns([campaign(), copy]) : planCampaigns(),
        }),
        { planId: 'P-1001' },
      ),
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
      reply(
        'POST /api/v1/campaigns/{campaignId}/copy',
        () => {
          copied = true;
          return { status: 201, body: copy };
        },
        { campaignId: 'C-1001' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Actions for Helix launch' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Copy' }));
    const dialog = await screen.findByRole('dialog', { name: "Copy campaign 'Helix launch'" });
    await within(dialog).findByRole('textbox', { name: /Name of the copy/ });
    await user.click(within(dialog).getByRole('button', { name: 'Copy campaign' }));

    const tree = await screen.findByRole('treegrid', { name: 'Plan tree' });
    const link = await within(tree).findByRole('link', { name: 'Helix launch (copy)' });
    await waitFor(() => expect(link.closest('tr')?.className).toContain('highlighted'));
    expect(api.bodiesOf('POST /api/v1/campaigns/C-1001/copy')).toEqual([
      { planId: 'P-1001', name: 'Helix launch (copy)' },
    ]);
    expect(await findMessages()).toHaveTextContent("Campaign 'Helix launch (copy)' created");
  });

  it('moves an initiative from its row to another campaign (US6#1, FR-023, FR-025)', async () => {
    const rimula = campaign({
      id: 'C-1002',
      name: 'Rimula fleet',
      initiatives: [],
      counts: { initiatives: 0, activities: 0, approvedActivities: 0, lineItems: 0 },
    });
    const moved = initiative({ campaignId: 'C-1002', version: 4 });
    let done = false;
    const { user, api } = renderTree([
      reply(
        'GET /api/v1/plans/{planId}/campaigns',
        () => ({
          body: done
            ? planCampaigns([campaign({ initiatives: [] }), { ...rimula, initiatives: [moved] }])
            : planCampaigns([campaign(), rimula]),
        }),
        { planId: 'P-1001' },
      ),
      reply(
        'GET /api/v1/initiatives/{initiativeId}/destinations',
        {
          body: destinationList(
            [destination()],
            'Initiatives move only between campaigns of the same plan',
          ),
        },
        { initiativeId: 'I-1001' },
      ),
      reply(
        'POST /api/v1/initiatives/{initiativeId}/move',
        () => {
          done = true;
          return { body: moved };
        },
        { initiativeId: 'I-1001' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Actions for Digital & social' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Move to…' }));
    const dialog = await screen.findByRole('dialog', {
      name: "Move initiative 'Digital & social'",
    });
    await user.click(await within(dialog).findByRole('radio', { name: 'Rimula fleet' }));
    await user.click(within(dialog).getByRole('button', { name: 'Move initiative' }));

    expect(await findMessages()).toHaveTextContent(
      "Initiative 'Digital & social' moved to 'Rimula fleet'",
    );
    expect(api.bodiesOf('POST /api/v1/initiatives/I-1001/move')).toEqual([
      { campaignId: 'C-1002', version: 3 },
    ]);
    const tree = screen.getByRole('treegrid', { name: 'Plan tree' });
    await waitFor(() => {
      // The header row, the campaigns with the initiative now under Rimula fleet, the totals row.
      const rows = within(tree).getAllByRole('row');
      expect(rows.map((row) => row.querySelector('a')?.textContent ?? '')).toEqual([
        '',
        'Helix launch',
        'Rimula fleet',
        'Digital & social',
        '',
      ]);
    });
  });
});
