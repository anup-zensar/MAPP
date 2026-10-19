import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { detailsOf, PEOPLE, workflowOf } from './testData';
import { WorkflowFormPanel } from './WorkflowFormPanel';
import type { ApprovalWorkflow } from './workflowsApi';

const CREATE = 'POST /api/approval-workflows';
const UPDATE = 'PATCH /api/approval-workflows/WF-1001';

function renderPanel(workflow?: ApprovalWorkflow, answers: Record<string, MockHandler> = {}) {
  const api = mockApi({ 'GET /api/people': PEOPLE, ...answers });
  const onClose = vi.fn();
  const rendered = renderWithProviders(
    <WorkflowFormPanel workflow={workflow} onClose={onClose} />,
    {
      user: testUser(),
    },
  );
  return { ...rendered, api, onClose };
}

describe('WorkflowFormPanel (feature 008, US4)', () => {
  it('creates a workflow with its levels and names it in the message (FR-002)', async () => {
    const { user, api, onClose } = renderPanel(undefined, {
      [CREATE]: { status: 201, body: workflowOf({ id: 'WF-1003', name: 'Lubricants Asia' }) },
    });
    const panel = screen.getByRole('dialog', { name: 'New approval workflow' });

    await user.type(within(panel).getByRole('textbox', { name: /^Name/ }), 'Lubricants Asia');
    await user.click(await within(panel).findByRole('combobox', { name: /Approvers/ }));
    await user.click(screen.getByRole('option', { name: /Carlos Mendes/ }));
    await user.click(within(panel).getByRole('button', { name: 'Add level' }));
    await user.type(within(panel).getAllByRole('textbox', { name: /Level name/ })[1], 'Global');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(
      await screen.findByText("Approval workflow 'Lubricants Asia' created"),
    ).toBeInTheDocument();
    expect(api.bodiesOf(CREATE)).toEqual([
      {
        name: 'Lubricants Asia',
        description: '',
        levels: [
          { name: '', approverIds: ['U-1005'] },
          { name: 'Global', approverIds: [] },
        ],
      },
    ]);
    expect(onClose).toHaveBeenCalled();
  });

  it('shows every problem at once, in the summary and at each field (FR-003)', async () => {
    const { user } = renderPanel(undefined, {
      [CREATE]: {
        status: 400,
        body: {
          code: 'validation_failed',
          message: 'Fix the problems.',
          fields: [
            { field: 'name', message: 'Enter a name.' },
            { field: 'levels.1.approverIds', message: 'Level 1: choose at least one approver.' },
          ],
        },
      },
    });
    await screen.findByRole('combobox', { name: /Approvers/ });

    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByRole('link', { name: 'Enter a name.' })).toHaveAttribute(
      'href',
      '#name',
    );
    expect(
      screen.getByRole('link', { name: 'Level 1: choose at least one approver.' }),
    ).toHaveAttribute('href', '#levels.1.approverIds');
    expect(screen.getByRole('textbox', { name: /^Name/ })).toHaveAccessibleDescription(
      expect.stringContaining('Enter a name.'),
    );
    expect(screen.getByRole('combobox', { name: /Approvers/ })).toHaveAccessibleDescription(
      expect.stringContaining('Level 1: choose at least one approver.'),
    );
  });

  it('says that open rounds keep their levels, and sends only what changed (FR-006)', async () => {
    const { user, api } = renderPanel(workflowOf(), {
      [UPDATE]: { body: workflowOf({ name: 'Lubricants EU approvals' }) },
    });

    const panel = screen.getByRole('dialog', { name: 'Edit approval workflow' });
    expect(panel).toHaveTextContent(
      'Changes apply to activities submitted from now on. 2 activities waiting for approval keep the levels they were submitted with.',
    );
    await user.type(within(panel).getByRole('textbox', { name: /^Name/ }), 's');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(
      await screen.findByText("Approval workflow 'Lubricants EU approvals' saved"),
    ).toBeInTheDocument();
    expect(api.bodiesOf(UPDATE)).toEqual([{ name: 'Lubricants EU approvals', version: 4 }]);
  });

  it('says 1 activity keeps its levels', () => {
    renderPanel(workflowOf({ waitingCount: 1 }));

    expect(screen.getByRole('dialog')).toHaveTextContent(
      '1 activity waiting for approval keeps the levels it was submitted with.',
    );
  });

  it('sends the new levels when they changed', async () => {
    const { user, api } = renderPanel(workflowOf({ waitingCount: 0 }), {
      [UPDATE]: { body: workflowOf() },
    });
    await screen.findAllByRole('combobox', { name: /Approvers/ });

    await user.click(screen.getByRole('button', { name: 'Move level 2 up' }));
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await screen.findByText("Approval workflow 'Lubricants EU approval' saved");
    expect(api.bodiesOf(UPDATE)).toEqual([
      {
        version: 4,
        levels: [
          { name: 'Global brand', approverIds: ['U-1001'] },
          { name: '', approverIds: ['U-1005'] },
        ],
      },
    ]);
  });

  it('says who saved first, and takes their version for the next Save (FR-003)', async () => {
    let saves = 0;
    const { user, api } = renderPanel(workflowOf(), {
      [UPDATE]: () => {
        saves += 1;
        return saves === 1
          ? {
              status: 409,
              body: {
                code: 'version_conflict',
                message: 'Someone else saved first.',
                fields: [],
                conflict: { changedBy: 'Alex Morgan', changedAt: new Date().toISOString() },
              },
            }
          : { body: workflowOf() };
      },
      'GET /api/approval-workflows/WF-1001': { body: detailsOf(workflowOf({ version: 5 })) },
    });

    await user.type(screen.getByRole('textbox', { name: /^Description/ }), ' More.');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(
      await screen.findByText(
        /^Not saved — Alex Morgan changed this approval workflow at \d\d:\d\d$/,
      ),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Save' }));
    await screen.findByText("Approval workflow 'Lubricants EU approval' saved");
    expect(api.bodiesOf(UPDATE).map((body) => (body as { version: number }).version)).toEqual([
      4, 5,
    ]);
  });

  it('asks before typed changes are lost', async () => {
    const { user, onClose } = renderPanel();

    await user.type(screen.getByRole('textbox', { name: /^Name/ }), 'Draft');
    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(await screen.findByRole('alertdialog')).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
  });
});
