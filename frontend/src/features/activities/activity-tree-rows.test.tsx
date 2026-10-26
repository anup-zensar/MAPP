import { screen, waitFor, within } from '@testing-library/react';
import { useNavigate } from 'react-router';
import { describe, expect, it } from 'vitest';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { TreeTable } from '@/ui/TreeTable/TreeTable';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { activityRecord } from '@/testing/fixtures/line-items';
import { destinations, launchFilm, launchFilmRow } from '@/testing/fixtures/activities';
import { activityActionsForInitiative, useActivityRows } from './activity-tree-rows';

const DELETED = {
  recycleBinEntry: {
    id: 'BIN-1001',
    recordType: 'activity',
    recordTypeLabel: 'activity',
    recordId: 'A-1042',
    recordName: 'Launch film',
    path: [],
    parent: { type: 'initiative', id: 'I-1001', name: 'Digital & social' },
    deletedAt: '2026-03-12T09:00:00Z',
    deletedBy: { id: 'U-1002', name: 'Priya Raman' },
    contents: [],
  },
};

const INITIATIVE = { id: 'I-1001', name: 'Digital & social', planId: 'P-1001' };

function InitiativeMenu() {
  const navigate = useNavigate();
  return (
    <ActionsMenu
      label={INITIATIVE.name}
      items={activityActionsForInitiative(INITIATIVE, navigate)}
    />
  );
}

/** The activity rows under "Digital & social", in a tree with the plan tree's columns. */
function ActivityRows() {
  const rows = useActivityRows('P-1001');
  return (
    <TreeTable
      label="Plan tree"
      columns={[
        { key: 'name', header: 'Name' },
        { key: 'owner', header: 'Owner' },
        { key: 'dates', header: 'Dates' },
        { key: 'status', header: 'Status' },
        { key: 'plannedCost', header: 'Planned cost', align: 'end' },
      ]}
      nodes={rows.get('I-1001') ?? []}
      expandedIds={new Set()}
      onToggle={() => {}}
    />
  );
}

function openRows(answers: Parameters<typeof mockApi>[0] = {}) {
  const api = mockApi({
    'GET /api/v1/plans/P-1001/activities': {
      body: {
        items: [
          launchFilmRow({
            completeness: { missingFields: 3, lineItemsNeedingAttention: 1 },
            unassignedPlannedCost: { amount: '-500.00', currency: 'USD' },
          }),
          launchFilmRow({ id: 'A-1050', name: 'Print ads', initiativeId: 'I-1002' }),
        ],
        total: 2,
      },
    },
    // No GSAP data yet: a delete question says nothing of transactions (015 FR-029).
    'GET /api/v1/financials/records/A-1042': {
      status: 404,
      body: {
        error: { code: 'not_found', message: 'No financial figures for A-1042.', fields: [] },
      },
    },
    ...answers,
  });
  const rendered = renderWithProviders(<ActivityRows />, {
    route: '/plans/P-1001',
    user: testUser(),
  });
  return { ...rendered, api };
}

describe('activityTreeRows (feature 006)', () => {
  it('shows each activity as a row with its cells, under its own initiative (FR-033)', async () => {
    openRows();

    const row = (await screen.findByRole('link', { name: 'Launch film' })).closest(
      'tr',
    ) as HTMLElement;
    expect(within(row).getByText('ACTIVITY')).toBeVisible();
    expect(within(row).getByText('A-1042')).toBeVisible();
    expect(within(row).getByText('Priya Raman')).toBeVisible();
    expect(within(row).getByText('01 Mar – 31 May 2026')).toBeVisible();
    expect(within(row).getByText('Created')).toBeVisible();
    expect(within(row).getByRole('link', { name: '3 fields needed' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1042#ready',
    );
    expect(within(row).getByText('1 line item needs attention')).toBeVisible();
    expect(within(row).getByText('Line items exceed the planned cost')).toBeInTheDocument();
    expect(screen.queryByText('Print ads')).not.toBeInTheDocument();
  });

  it('shows the planned cost in the plan currency with the display currency under it (016 FR-013)', async () => {
    openRows({
      'GET /api/v1/plans/P-1001/activities': {
        body: {
          items: [
            launchFilmRow({
              plannedCost: { amount: '3950.00', currency: 'GBP' },
              unassignedPlannedCost: { amount: '0.00', currency: 'GBP' },
              display: {
                plannedCost: { amount: '5000.00', currency: 'USD' },
                unassignedPlannedCost: { amount: '0.00', currency: 'USD' },
                revenue: null,
              },
            }),
          ],
          total: 1,
        },
      },
    });

    const row = (await screen.findByRole('link', { name: 'Launch film' })).closest(
      'tr',
    ) as HTMLElement;
    expect(within(row).getByText(/3,950\.00/)).toHaveTextContent(/GBP|£/);
    expect(within(row).getByText(/≈/)).toHaveTextContent(/5,000\.00/);
  });

  it('shows only the display currency when the plan is planned in it', async () => {
    openRows();

    const row = (await screen.findByRole('link', { name: 'Launch film' })).closest(
      'tr',
    ) as HTMLElement;
    expect(within(row).queryByText(/≈/)).not.toBeInTheDocument();
  });

  it('opens the activity page from the row and offers View, Edit, Copy, Move, Split activity and Delete in its menu', async () => {
    const { user, router } = openRows();

    await user.click(await screen.findByRole('button', { name: 'Actions for Launch film' }));
    const menu = await screen.findByRole('menu');
    expect(
      within(menu)
        .getAllByRole('menuitem')
        .map((item) => item.textContent),
    ).toEqual(['View', 'Edit', 'Copy', 'Move', 'Split activity', 'Delete']);
    await user.click(within(menu).getByRole('menuitem', { name: 'Edit' }));
    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1042/edit'),
    );
  });

  it('deletes an activity from its row after asking, by keyboard, and offers Undo (FR-031)', async () => {
    const { user, router, api } = openRows({
      'GET /api/v1/activities/A-1042': {
        body: launchFilm({ lineItems: { ...launchFilm().lineItems, count: 2 } }),
      },
      'DELETE /api/v1/activities/A-1042': { body: DELETED },
      'POST /api/v1/recycle-bin/BIN-1001/undo': { body: {} },
      'GET /api/v1/financials/records/A-1042': {
        body: activityRecord({ record: { type: 'activity', id: 'A-1042', name: 'Launch film' } }),
      },
    });

    await user.click(await screen.findByRole('button', { name: 'Actions for Launch film' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const question = await screen.findByRole('alertdialog', {
      name: "Delete activity 'Launch film'?",
    });
    expect(within(question).getByText('2 line items go with it.')).toBeVisible();
    // The GSAP transactions linked to its line items stay (015 FR-029).
    expect(
      await within(question).findByText(
        '3 GSAP transactions are linked to it. They stay in MAPP and leave the figures until it is restored.',
      ),
    ).toBeVisible();
    expect(within(question).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    expect(api.bodiesOf('DELETE /api/v1/activities/A-1042')).toEqual([]);

    // Enter on the question's button is the dialog's, not the row's (which opens the page).
    await user.tab();
    expect(within(question).getByRole('button', { name: 'Delete activity' })).toHaveFocus();
    await user.keyboard('{Enter}');

    expect(
      await screen.findByText("Activity 'Launch film' moved to the recycle bin"),
    ).toBeVisible();
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/plans/P-1001');

    const messages = await findMessages();
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));
    expect(await screen.findByText("Activity 'Launch film' restored")).toBeVisible();
  });

  it('copies an activity from its row with the activity as stored now (FR-030)', async () => {
    const { user, router, api } = openRows({
      'GET /api/v1/activities/A-1042': {
        body: launchFilm({ lineItems: { ...launchFilm().lineItems, count: 2 } }),
      },
      'GET /api/v1/activities/A-1042/destinations': { body: destinations() },
      'POST /api/v1/activities/A-1042/copy': {
        status: 201,
        body: launchFilm({ id: 'A-1043', name: 'Launch film (copy)' }),
      },
    });

    await user.click(await screen.findByRole('button', { name: 'Actions for Launch film' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Copy' }));
    const dialog = await screen.findByRole('dialog', { name: "Copy activity 'Launch film'" });
    expect(dialog).toHaveTextContent('with copies of its 2 line items');
    await within(dialog).findByRole('group', { name: 'Copy to' });
    await user.click(within(dialog).getByRole('button', { name: 'Copy activity' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1043'),
    );
    expect(api.bodiesOf('POST /api/v1/activities/A-1042/copy')).toEqual([
      { initiativeId: 'I-1001' },
    ]);
    expect(await findMessages()).toHaveTextContent("Activity 'Launch film (copy)' created");
  });

  it('moves an activity from its row with the version stored now (FR-029)', async () => {
    const { user, api } = openRows({
      'GET /api/v1/activities/A-1042': { body: launchFilm({ version: 7 }) },
      'GET /api/v1/activities/A-1042/destinations': { body: destinations() },
      'POST /api/v1/activities/A-1042/move': {
        body: launchFilm({
          initiative: { type: 'initiative', id: 'I-1002', name: 'Print & outdoor' },
          version: 8,
        }),
      },
    });

    await user.click(await screen.findByRole('button', { name: 'Actions for Launch film' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Move' }));
    const dialog = await screen.findByRole('dialog', { name: "Move activity 'Launch film'" });
    await user.click(await within(dialog).findByRole('radio', { name: 'Print & outdoor' }));
    await user.click(within(dialog).getByRole('button', { name: 'Move activity' }));

    expect(await findMessages()).toHaveTextContent(
      "Activity 'Launch film' moved to 'Print & outdoor'",
    );
    expect(api.bodiesOf('POST /api/v1/activities/A-1042/move')).toEqual([
      { initiativeId: 'I-1002', version: 7 },
    ]);
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('closes the question of a row on Cancel, deleting nothing', async () => {
    const { user, api } = openRows({ 'GET /api/v1/activities/A-1042': { body: launchFilm() } });

    await user.click(await screen.findByRole('button', { name: 'Actions for Launch film' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const question = await screen.findByRole('alertdialog');
    await user.click(within(question).getByRole('button', { name: 'Cancel' }));

    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());
    expect(api.bodiesOf('DELETE /api/v1/activities/A-1042')).toEqual([]);
  });

  it('Add activity opens the new-activity page (FR-001)', async () => {
    const { user, router } = renderWithProviders(<InitiativeMenu />, { route: '/plans/P-1001' });

    await user.click(screen.getByRole('button', { name: 'Actions for Digital & social' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Add activity' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe(
        '/plans/P-1001/initiatives/I-1001/activities/new',
      ),
    );
  });
});
