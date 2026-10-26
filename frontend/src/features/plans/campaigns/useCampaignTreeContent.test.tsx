import { screen, waitFor, within } from '@testing-library/react';
import { Outlet, useParams } from 'react-router';
import { describe, expect, it } from 'vitest';
import { SessionProvider } from '@/session/SessionProvider';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { TreeTable } from '@/ui/TreeTable/TreeTable';
import { campaignForm } from '@/testing/fixtures/attribute-forms';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { findMessages, renderRoutes } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { CELL_KEYS } from './campaignRows';
import {
  campaign,
  destination,
  destinationList,
  initiative,
  PEOPLE,
  planAnswer,
  planCampaigns,
  planDestination,
  usd,
} from './fixtures';
import { NewCampaignPanel } from './NewCampaignPanel';
import { useCampaignTreeContent } from './useCampaignTreeContent';

/** Stands in for feature 004's plan page: the tree with what this hook gives it. */
function PlanStandIn() {
  const { planId = '' } = useParams();
  const content = useCampaignTreeContent(planId);
  if (content.isLoading) {
    return <p>Loading rows</p>;
  }
  if (content.error) {
    return (
      <button type="button" onClick={content.refetch}>
        Try again
      </button>
    );
  }
  return (
    <>
      <div>{content.toolbarActions}</div>
      <ActionsMenu label="FY2026 Lubricants Europe" items={content.planRowActions} />
      <TreeTable
        label="Plan tree"
        columns={CELL_KEYS.map((key) => ({ key, header: key }))}
        nodes={content.nodes}
        expandedIds={new Set(content.nodes.map((node) => node.id))}
        onToggle={() => {}}
        footer={content.footer}
      />
      {content.nodes.length === 0 && <div data-testid="empty">{content.emptyAction}</div>}
      {content.dialogs}
      <Outlet />
    </>
  );
}

function renderTree(answers: Record<string, MockHandler>, route: unknown = '/plans/P-1001') {
  // Feature 006 adds the plan's activities under the initiatives: none here.
  const api = mockApi({
    'GET /api/v1/plans/P-1001/activities': { body: { items: [], total: 0 } },
    ...answers,
  });
  const rendered = renderRoutes(
    [
      {
        element: (
          <SessionProvider>
            <Outlet />
          </SessionProvider>
        ),
        children: [
          {
            path: '/plans/:planId',
            element: <PlanStandIn />,
            children: [
              { path: 'campaigns/new', element: <NewCampaignPanel /> },
              { path: '*', element: <p>A panel</p> },
            ],
          },
        ],
      },
    ],
    { route: route as string, user: testUser() },
  );
  return { api, ...rendered };
}

describe('useCampaignTreeContent (feature 005)', () => {
  it('gives the plan tree its campaign rows with their initiatives and the totals row (FR-019, FR-022)', async () => {
    renderTree({ 'GET /api/v1/plans/P-1001/campaigns': { body: planCampaigns() } });

    const tree = await screen.findByRole('treegrid', { name: 'Plan tree' });
    const rows = within(tree).getAllByRole('row');
    expect(rows[1]).toHaveTextContent('Helix launch');
    expect(rows[2]).toHaveTextContent('Digital & social');
    expect(within(tree).getByText('1 campaign · 1 initiative · 0 activities')).toBeVisible();
  });

  it('offers Add campaign in the toolbar and on the plan row (FR-001)', async () => {
    const { user, router } = renderTree({
      'GET /api/v1/plans/P-1001/campaigns': { body: planCampaigns() },
      'GET /api/v1/plans/P-1001': { body: planAnswer() },
      'GET /api/v1/shell-attribute-forms/campaign': { body: campaignForm() },
      'GET /api/v1/people': { body: PEOPLE },
    });

    await user.click(await screen.findByRole('link', { name: 'Add campaign' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001/campaigns/new'));

    await router.navigate('/plans/P-1001');
    await user.click(
      await screen.findByRole('button', { name: 'Actions for FY2026 Lubricants Europe' }),
    );
    await user.click(await screen.findByRole('menuitem', { name: 'Add campaign' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001/campaigns/new'));
  });

  it('offers Add campaign under an empty tree (FR-001)', async () => {
    renderTree({ 'GET /api/v1/plans/P-1001/campaigns': { body: planCampaigns([]) } });

    const empty = await screen.findByTestId('empty');
    expect(within(empty).getByRole('link', { name: 'Add campaign' })).toHaveAttribute(
      'href',
      '/plans/P-1001/campaigns/new',
    );
  });

  it('highlights the record named in the location state (FR-004)', async () => {
    renderTree(
      { 'GET /api/v1/plans/P-1001/campaigns': { body: planCampaigns() } },
      { pathname: '/plans/P-1001', state: { highlight: 'I-1001' } },
    );

    const tree = await screen.findByRole('treegrid', { name: 'Plan tree' });
    const highlighted = within(tree)
      .getAllByRole('row')
      .filter((row) => row.className.includes('highlighted'));
    expect(highlighted).toHaveLength(1);
    expect(highlighted[0]).toHaveTextContent('Digital & social');
  });

  it('shows a new campaign and the new totals right after it is created (FR-012)', async () => {
    const created = campaign({
      id: 'C-1005',
      name: 'Fleet growth',
      budget: usd('100000.00'),
      initiatives: [],
      counts: { initiatives: 0, activities: 0, approvedActivities: 0, lineItems: 0 },
    });
    let saved = false;
    const { user } = renderTree({
      'GET /api/v1/plans/P-1001/campaigns': () => ({
        body: saved
          ? planCampaigns([campaign(), created], { allocatedToCampaigns: usd('500000.00') })
          : planCampaigns(),
      }),
      'GET /api/v1/plans/P-1001': { body: planAnswer() },
      'GET /api/v1/shell-attribute-forms/campaign': { body: campaignForm() },
      'GET /api/v1/people': { body: PEOPLE },
      'POST /api/v1/plans/P-1001/campaigns': () => {
        saved = true;
        return { status: 201, body: created };
      },
    });

    await user.click(await screen.findByRole('link', { name: 'Add campaign' }));
    await user.type(await screen.findByLabelText('Campaign name'), 'Fleet growth');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    const tree = await screen.findByRole('treegrid', { name: 'Plan tree' });
    expect(await within(tree).findByRole('link', { name: 'Fleet growth' })).toBeInTheDocument();
    expect(within(tree).getByText('2 campaigns · 1 initiative · 0 activities')).toBeVisible();
    expect(within(tree).getByText('$500,000.00')).toBeVisible();
    const row = within(tree).getByRole('link', { name: 'Fleet growth' }).closest('tr');
    expect(row?.className).toContain('highlighted');
  });

  it('deletes from a row’s ⋯ menu and the row leaves the tree at once (FR-021, FR-028, FR-012)', async () => {
    let deleted = false;
    const { user } = renderTree({
      'GET /api/v1/plans/P-1001/campaigns': () => ({
        body: deleted ? planCampaigns([]) : planCampaigns(),
      }),
      'DELETE /api/v1/initiatives/I-1001': () => {
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
    });

    await user.click(await screen.findByRole('button', { name: 'Actions for Digital & social' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete initiative 'Digital & social'?",
    });
    await user.click(within(dialog).getByRole('button', { name: 'Delete initiative' }));

    expect(await screen.findByTestId('empty')).toBeInTheDocument();
    expect(await findMessages()).toHaveTextContent(
      "Initiative 'Digital & social' moved to the recycle bin",
    );
  });

  it('copies a campaign from its row; the copy appears highlighted (US6#4–5, FR-026, FR-027)', async () => {
    const copy = campaign({
      id: 'C-1009',
      name: 'Helix launch (copy)',
      initiatives: [initiative({ id: 'I-1009', campaignId: 'C-1009' })],
    });
    let copied = false;
    const { user, api } = renderTree({
      'GET /api/v1/plans/P-1001/campaigns': () => ({
        body: copied ? planCampaigns([campaign(), copy]) : planCampaigns(),
      }),
      'GET /api/v1/campaigns/C-1001/destinations': {
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
      'POST /api/v1/campaigns/C-1001/copy': () => {
        copied = true;
        return { status: 201, body: copy };
      },
    });

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
    const { user, api } = renderTree({
      'GET /api/v1/plans/P-1001/campaigns': () => ({
        body: done
          ? planCampaigns([campaign({ initiatives: [] }), { ...rimula, initiatives: [moved] }])
          : planCampaigns([campaign(), rimula]),
      }),
      'GET /api/v1/initiatives/I-1001/destinations': {
        body: destinationList(
          [destination()],
          'Initiatives move only between campaigns of the same plan',
        ),
      },
      'POST /api/v1/initiatives/I-1001/move': () => {
        done = true;
        return { body: moved };
      },
    });

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

  it('reports a failure to read the rows with Try again', async () => {
    let calls = 0;
    const { user } = renderTree({
      'GET /api/v1/plans/P-1001/campaigns': () =>
        ++calls === 1
          ? { status: 500, body: { error: { code: 'internal_error', message: 'Oops' } } }
          : { body: planCampaigns([campaign({ initiatives: [initiative()] })]) },
    });

    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('treegrid', { name: 'Plan tree' })).toBeInTheDocument();
  });
});
