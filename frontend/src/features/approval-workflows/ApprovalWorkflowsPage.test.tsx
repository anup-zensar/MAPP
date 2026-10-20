import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages, renderWithProviders } from '@/test/render';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { testUser } from '@/test/users';
import { ApprovalWorkflowsPage } from './ApprovalWorkflowsPage';
import { CHINA_WORKFLOW, detailsOf, PEOPLE, workflowOf } from './testData';

const LIST = 'GET /api/approval-workflows';
const ITEMS = [CHINA_WORKFLOW, workflowOf()];
const list = (items = ITEMS) => ({ body: { items, total: items.length } });

const BIN_ENTRY = {
  id: 'RB-1001',
  recordType: 'approvalWorkflow',
  recordTypeLabel: 'approval workflow',
  recordId: 'WF-1002',
  recordName: 'China BC approval workflow',
  path: [],
  parent: null,
  contents: [],
  deletedAt: '2026-03-12T09:00:00Z',
  deletedBy: { id: 'U-1002', name: 'Priya Raman' },
};

function renderPage(
  route = '/settings/approval-workflows',
  answers: Record<string, MockHandler> = {},
) {
  const api = mockApi({ [LIST]: list(), 'GET /api/people': PEOPLE, ...answers });
  const rendered = renderWithProviders(<ApprovalWorkflowsPage />, {
    route,
    path: '/settings/approval-workflows/:workflowId?',
    user: testUser(),
  });
  return { ...rendered, api };
}

const rowOf = (name: string) => screen.getByRole('row', { name: new RegExp(name) });

describe('ApprovalWorkflowsPage (feature 008, US4)', () => {
  it('lists the workflows with their levels, use and status (FR-001, FR-005)', async () => {
    renderPage('/settings/approval-workflows?status=all');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Approval workflows' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Approval workflows · NorthStar');
    const eu = await screen.findByRole('row', { name: /Lubricants EU approval/ });
    expect(eu).toHaveTextContent('Level 1 · Carlos Mendes → Global brand · Alex Morgan');
    expect(eu).toHaveTextContent('Used by 3 activities (2 waiting for approval)');
    expect(eu).toHaveTextContent('Active');
    const china = rowOf('China BC approval workflow');
    expect(china).toHaveTextContent('Not used yet');
    expect(china).toHaveTextContent('Inactive');
    expect(screen.getByText('2 of 2 approval workflows')).toBeInTheDocument();
    expect(within(eu).getByRole('link', { name: 'Lubricants EU approval' })).toHaveAttribute(
      'href',
      '/settings/approval-workflows/WF-1001',
    );
  });

  it('shows the active workflows first, searches names and filters by status', async () => {
    const { user } = renderPage();
    await screen.findByRole('row', { name: /Lubricants EU approval/ });
    const search = () => screen.getByRole('searchbox', { name: 'Search approval workflows' });

    expect(screen.getByText('1 of 2 approval workflows')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Active 1' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(screen.queryByRole('row', { name: /China BC/ })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /^All/ }));
    await user.type(search(), 'china');
    expect(screen.getByText('1 of 2 approval workflows')).toBeInTheDocument();
    expect(screen.queryByRole('row', { name: /Lubricants EU/ })).not.toBeInTheDocument();

    await user.clear(search());
    await user.click(screen.getByRole('button', { name: /^Inactive/ }));
    await user.type(search(), 'lubricants');
    expect(screen.getByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));
    expect(screen.getByText('2 of 2 approval workflows')).toBeInTheDocument();
  });

  it('shows a warning icon for a workflow with warnings', async () => {
    renderPage('/settings/approval-workflows', {
      [LIST]: list([workflowOf({ warnings: ['Level 2 has no active approver'] })]),
    });

    expect(
      await screen.findByRole('button', { name: 'Warnings: Level 2 has no active approver' }),
    ).toBeInTheDocument();
  });

  it('invites to create the first workflow when there is none', async () => {
    const { user } = renderPage('/settings/approval-workflows', { [LIST]: list([]) });

    expect(
      await screen.findByText('No approval workflows yet — create your first approval workflow'),
    ).toBeInTheDocument();
    await user.click(screen.getAllByRole('button', { name: 'New approval workflow' })[1]);
    expect(screen.getByRole('dialog', { name: 'New approval workflow' })).toBeInTheDocument();
  });

  it('sets a workflow inactive after saying what changes (FR-007)', async () => {
    const { user, api } = renderPage('/settings/approval-workflows', {
      'POST /api/approval-workflows/WF-1001/deactivate': { body: workflowOf({ active: false }) },
    });
    await screen.findByRole('row', { name: /Lubricants EU approval/ });

    await user.click(screen.getByRole('button', { name: 'Actions for Lubricants EU approval' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Set inactive' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Set approval workflow 'Lubricants EU approval' inactive?",
    });
    expect(dialog).toHaveTextContent('It will no longer be offered on activities.');
    expect(dialog).toHaveTextContent(
      "The 3 activities that use it keep it, shown as '(inactive)'.",
    );
    expect(dialog).toHaveTextContent(
      '2 activities waiting for approval finish with the levels they were submitted with.',
    );
    await user.click(within(dialog).getByRole('button', { name: 'Set inactive' }));

    expect(
      await screen.findByText("Approval workflow 'Lubricants EU approval' set inactive"),
    ).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/approval-workflows/WF-1001/deactivate')).toEqual([
      { version: 4 },
    ]);
  });

  it('sets an unused workflow inactive at once, with Undo (design system §6.7)', async () => {
    const unused = workflowOf({ usedByCount: 0, waitingCount: 0 });
    const { user, api } = renderPage('/settings/approval-workflows', {
      [LIST]: list([unused]),
      'POST /api/approval-workflows/WF-1001/deactivate': {
        body: { ...unused, active: false, version: 5 },
      },
      'POST /api/approval-workflows/WF-1001/activate': {
        body: { ...unused, active: true, version: 6 },
      },
    });
    await screen.findByRole('row', { name: /Lubricants EU approval/ });

    await user.click(screen.getByRole('button', { name: 'Actions for Lubricants EU approval' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Set inactive' }));

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    const messages = await findMessages();
    expect(
      await within(messages).findByText("Approval workflow 'Lubricants EU approval' set inactive"),
    ).toBeInTheDocument();
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));
    expect(
      await screen.findByText("Approval workflow 'Lubricants EU approval' reactivated"),
    ).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/approval-workflows/WF-1001/deactivate')).toEqual([
      { version: 4 },
    ]);
    expect(api.bodiesOf('POST /api/approval-workflows/WF-1001/activate')).toEqual([{ version: 5 }]);
  });

  it('reactivates an inactive workflow at once (FR-007)', async () => {
    const { user, api } = renderPage('/settings/approval-workflows?status=inactive', {
      'POST /api/approval-workflows/WF-1002/activate': { body: CHINA_WORKFLOW },
    });
    await user.click(await screen.findByRole('button', { name: /^All/ }));
    await screen.findByRole('row', { name: /China BC/ });

    await user.click(
      screen.getByRole('button', { name: 'Actions for China BC approval workflow' }),
    );
    await user.click(await screen.findByRole('menuitem', { name: 'Reactivate' }));

    expect(
      await screen.findByText("Approval workflow 'China BC approval workflow' reactivated"),
    ).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/approval-workflows/WF-1002/activate')).toEqual([{ version: 2 }]);
  });

  it('explains that a workflow in use is set inactive instead of deleted (FR-008)', async () => {
    const { user, api } = renderPage();
    await screen.findByRole('row', { name: /Lubricants EU approval/ });

    await user.click(screen.getByRole('button', { name: 'Actions for Lubricants EU approval' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));

    const dialog = await screen.findByRole('alertdialog', {
      name: "Can't delete 'Lubricants EU approval'",
    });
    expect(dialog).toHaveTextContent('Used by 3 activities — set it inactive instead.');
    expect(within(dialog).getByRole('button', { name: 'Set inactive' })).toBeInTheDocument();
    expect(api.requests.some((request) => request.method === 'DELETE')).toBe(false);
  });

  it('deletes an unused workflow with Undo (FR-008)', async () => {
    const { user } = renderPage('/settings/approval-workflows?status=all', {
      'DELETE /api/approval-workflows/WF-1002': { body: { recycleBinEntry: BIN_ENTRY } },
      'POST /api/recycle-bin/RB-1001/undo': {
        body: {
          recordType: 'approvalWorkflow',
          recordTypeLabel: 'approval workflow',
          recordId: 'WF-1002',
          recordName: 'China BC approval workflow',
          details: '',
        },
      },
    });
    await user.click(await screen.findByRole('button', { name: /^All/ }));
    await screen.findByRole('row', { name: /China BC/ });

    await user.click(
      screen.getByRole('button', { name: 'Actions for China BC approval workflow' }),
    );
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete approval workflow 'China BC approval workflow'?",
    });
    expect(dialog).toHaveTextContent('It moves to the recycle bin; you can restore it.');
    await user.click(within(dialog).getByRole('button', { name: 'Delete approval workflow' }));

    expect(
      await screen.findByText(
        "Approval workflow 'China BC approval workflow' moved to the recycle bin",
      ),
    ).toBeInTheDocument();
    const messages = await findMessages();
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));
    expect(
      await screen.findByText("Approval workflow 'China BC approval workflow' restored"),
    ).toBeInTheDocument();
  });

  it('shows the used dialog when the API answers in_use (FR-008)', async () => {
    const { user } = renderPage('/settings/approval-workflows?status=all', {
      'DELETE /api/approval-workflows/WF-1002': {
        status: 409,
        body: {
          code: 'in_use',
          message: 'Used by 1 activity — set it inactive instead.',
          fields: [],
        },
      },
    });
    await user.click(await screen.findByRole('button', { name: /^All/ }));
    await screen.findByRole('row', { name: /China BC/ });

    await user.click(
      screen.getByRole('button', { name: 'Actions for China BC approval workflow' }),
    );
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    await user.click(
      within(await screen.findByRole('alertdialog')).getByRole('button', {
        name: 'Delete approval workflow',
      }),
    );

    expect(
      await screen.findByRole('alertdialog', { name: "Can't delete 'China BC approval workflow'" }),
    ).toBeInTheDocument();
  });

  it('opens the details at their address and goes back to the list on close (FR-005)', async () => {
    const { user, router } = renderPage('/settings/approval-workflows/WF-1001', {
      'GET /api/approval-workflows/WF-1001': { body: detailsOf() },
    });

    const panel = await screen.findByRole('dialog', { name: 'Lubricants EU approval' });
    await user.click(within(panel).getAllByRole('button', { name: 'Close' })[0]);

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/settings/approval-workflows'),
    );
  });

  it('edits from the menu in the side panel', async () => {
    const { user } = renderPage();
    await screen.findByRole('row', { name: /Lubricants EU approval/ });

    await user.click(screen.getByRole('button', { name: 'Actions for Lubricants EU approval' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Edit' }));

    const panel = await screen.findByRole('dialog', { name: 'Edit approval workflow' });
    expect(within(panel).getByRole('textbox', { name: /^Name/ })).toHaveValue(
      'Lubricants EU approval',
    );
  });

  it('sets an unused workflow inactive at once, with Undo (design system §6.7)', async () => {
    const unused = workflowOf({ usedByCount: 0, waitingCount: 0 });
    const { user, api } = renderPage('/settings/approval-workflows', {
      [LIST]: list([unused]),
      'POST /api/approval-workflows/WF-1001/deactivate': {
        body: workflowOf({ active: false, usedByCount: 0, waitingCount: 0, version: 5 }),
      },
      'POST /api/approval-workflows/WF-1001/activate': { body: unused },
    });
    await screen.findByRole('row', { name: /Lubricants EU approval/ });

    await user.click(screen.getByRole('button', { name: 'Actions for Lubricants EU approval' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Set inactive' }));

    expect(
      await screen.findByText("Approval workflow 'Lubricants EU approval' set inactive"),
    ).toBeInTheDocument();
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    const messages = await findMessages();
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));
    await waitFor(() =>
      expect(api.bodiesOf('POST /api/approval-workflows/WF-1001/activate')).toEqual([
        { version: 5 },
      ]),
    );
  });
});
