import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import { CHINA_WORKFLOW, workflowOf } from '@/testing/fixtures/approval-workflows';
import { renderWorkflowsPage, workflowList } from '@/testing/fixtures/approval-workflows-page';
import { reply } from '@/testing/replies';

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

describe('ApprovalWorkflowsPage set inactive, reactivate and delete (feature 008, US4)', () => {
  it('sets a workflow inactive after saying what changes (FR-007)', async () => {
    const { user, api } = renderWorkflowsPage('/settings/approval-workflows', [
      reply(
        'POST /api/v1/approval-workflows/{workflowId}/deactivate',
        { body: workflowOf({ active: false }) },
        { workflowId: 'WF-1001' },
      ),
    ]);
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
    expect(api.bodiesOf('POST /api/v1/approval-workflows/WF-1001/deactivate')).toEqual([
      { version: 4 },
    ]);
  });

  it('sets an unused workflow inactive at once, with Undo (design system §6.7)', async () => {
    const unused = workflowOf({ usedByCount: 0, waitingCount: 0 });
    const { user, api } = renderWorkflowsPage('/settings/approval-workflows', [
      reply('GET /api/v1/approval-workflows', workflowList([unused])),
      reply(
        'POST /api/v1/approval-workflows/{workflowId}/deactivate',
        {
          body: { ...unused, active: false, version: 5 },
        },
        { workflowId: 'WF-1001' },
      ),
      reply(
        'POST /api/v1/approval-workflows/{workflowId}/activate',
        {
          body: { ...unused, active: true, version: 6 },
        },
        { workflowId: 'WF-1001' },
      ),
    ]);
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
    expect(api.bodiesOf('POST /api/v1/approval-workflows/WF-1001/deactivate')).toEqual([
      { version: 4 },
    ]);
    expect(api.bodiesOf('POST /api/v1/approval-workflows/WF-1001/activate')).toEqual([
      { version: 5 },
    ]);
  });

  it('reactivates an inactive workflow at once (FR-007)', async () => {
    const { user, api } = renderWorkflowsPage('/settings/approval-workflows?status=inactive', [
      reply(
        'POST /api/v1/approval-workflows/{workflowId}/activate',
        { body: CHINA_WORKFLOW },
        { workflowId: 'WF-1002' },
      ),
    ]);
    await user.click(await screen.findByRole('button', { name: /^All/ }));
    await screen.findByRole('row', { name: /China BC/ });

    await user.click(
      screen.getByRole('button', { name: 'Actions for China BC approval workflow' }),
    );
    await user.click(await screen.findByRole('menuitem', { name: 'Reactivate' }));

    expect(
      await screen.findByText("Approval workflow 'China BC approval workflow' reactivated"),
    ).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/approval-workflows/WF-1002/activate')).toEqual([
      { version: 2 },
    ]);
  });

  it('explains that a workflow in use is set inactive instead of deleted (FR-008)', async () => {
    const { user, api } = renderWorkflowsPage();
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
    const { user } = renderWorkflowsPage('/settings/approval-workflows?status=all', [
      reply(
        'DELETE /api/v1/approval-workflows/{workflowId}',
        { body: { recycleBinEntry: BIN_ENTRY } },
        { workflowId: 'WF-1002' },
      ),
      reply(
        'POST /api/v1/recycle-bin/{entryId}/undo',
        {
          body: {
            recordType: 'approvalWorkflow',
            recordTypeLabel: 'approval workflow',
            recordId: 'WF-1002',
            recordName: 'China BC approval workflow',
            details: '',
          },
        },
        { entryId: 'RB-1001' },
      ),
    ]);
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
    const { user } = renderWorkflowsPage('/settings/approval-workflows?status=all', [
      reply(
        'DELETE /api/v1/approval-workflows/{workflowId}',
        {
          status: 409,
          body: {
            error: {
              code: 'in_use',
              message: 'Used by 1 activity — set it inactive instead.',
              fields: [],
            },
          },
        },
        { workflowId: 'WF-1002' },
      ),
    ]);
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

  it('sets an unused workflow inactive at once, with Undo (design system §6.7)', async () => {
    const unused = workflowOf({ usedByCount: 0, waitingCount: 0 });
    const { user, api } = renderWorkflowsPage('/settings/approval-workflows', [
      reply('GET /api/v1/approval-workflows', workflowList([unused])),
      reply(
        'POST /api/v1/approval-workflows/{workflowId}/deactivate',
        {
          body: workflowOf({ active: false, usedByCount: 0, waitingCount: 0, version: 5 }),
        },
        { workflowId: 'WF-1001' },
      ),
      reply(
        'POST /api/v1/approval-workflows/{workflowId}/activate',
        { body: unused },
        { workflowId: 'WF-1001' },
      ),
    ]);
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
      expect(api.bodiesOf('POST /api/v1/approval-workflows/WF-1001/activate')).toEqual([
        { version: 5 },
      ]),
    );
  });
});
