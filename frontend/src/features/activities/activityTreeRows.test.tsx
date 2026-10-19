import { screen, waitFor, within } from '@testing-library/react';
import { useNavigate } from 'react-router';
import { describe, expect, it } from 'vitest';
import { ActionsMenu, TreeTable } from '@/components';
import { mockApi } from '@/test/mockApi';
import { findMessages, renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { activityActionsForInitiative, useActivityRows } from './activityTreeRows';
import { launchFilm, launchFilmRow } from './testData';

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
    'GET /api/plans/P-1001/activities': {
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
    expect(within(row).getByText('01 Mar 2026 – 31 May 2026')).toBeVisible();
    expect(within(row).getByText('Created')).toBeVisible();
    expect(within(row).getByRole('link', { name: '3 fields needed' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1042#ready',
    );
    expect(within(row).getByText('1 line item needs attention')).toBeVisible();
    expect(within(row).getByText('Line items exceed the planned cost')).toBeInTheDocument();
    expect(screen.queryByText('Print ads')).not.toBeInTheDocument();
  });

  it('opens the activity page from the row and offers View, Edit and Delete in its menu', async () => {
    const { user, router } = openRows();

    await user.click(await screen.findByRole('button', { name: 'Actions for Launch film' }));
    const menu = await screen.findByRole('menu');
    expect(
      within(menu)
        .getAllByRole('menuitem')
        .map((item) => item.textContent),
    ).toEqual(['View', 'Edit', 'Delete']);
    await user.click(within(menu).getByRole('menuitem', { name: 'Edit' }));
    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1042/edit'),
    );
  });

  it('deletes an activity from its row after asking, by keyboard, and offers Undo (FR-031)', async () => {
    const { user, router, api } = openRows({
      'GET /api/activities/A-1042': {
        body: launchFilm({ lineItems: { ...launchFilm().lineItems, count: 2 } }),
      },
      'DELETE /api/activities/A-1042': { body: DELETED },
      'POST /api/recycle-bin/BIN-1001/undo': { body: {} },
    });

    await user.click(await screen.findByRole('button', { name: 'Actions for Launch film' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const question = await screen.findByRole('alertdialog', {
      name: "Delete activity 'Launch film'?",
    });
    expect(within(question).getByText('2 line items go with it.')).toBeVisible();
    expect(within(question).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    expect(api.bodiesOf('DELETE /api/activities/A-1042')).toEqual([]);

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

  it('closes the question of a row on Cancel, deleting nothing', async () => {
    const { user, api } = openRows({ 'GET /api/activities/A-1042': { body: launchFilm() } });

    await user.click(await screen.findByRole('button', { name: 'Actions for Launch film' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const question = await screen.findByRole('alertdialog');
    await user.click(within(question).getByRole('button', { name: 'Cancel' }));

    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());
    expect(api.bodiesOf('DELETE /api/activities/A-1042')).toEqual([]);
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
