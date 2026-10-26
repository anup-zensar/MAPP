import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { detailsOf, PEOPLE, workflowOf } from '@/testing/fixtures/approval-workflows';
import { reply } from '@/testing/replies';
import { WorkflowFormPanel } from './WorkflowFormPanel';
import type { ApprovalWorkflow } from './api';

const CREATE = 'POST /api/v1/approval-workflows';

/** A level as the form sends it (017): named people, any one approves, no rules. */
const sentLevel = (name: string, approverIds: string[]) => ({
  name,
  approverType: 'people',
  approverIds,
  mode: 'any',
  appliesWhen: null,
  autoApproveWhen: null,
  backupApproverId: '',
});
const UPDATE = 'PATCH /api/v1/approval-workflows/WF-1001';

function renderPanel(workflow?: ApprovalWorkflow, answers: HttpHandler[] = []) {
  const api = mockApi([reply('GET /api/v1/people', PEOPLE), ...answers]);
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
    const { user, api, onClose } = renderPanel(undefined, [
      reply('POST /api/v1/approval-workflows', {
        status: 201,
        body: workflowOf({ id: 'WF-1003', name: 'Lubricants Asia' }),
      }),
    ]);
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
        levels: [sentLevel('', ['U-1005']), sentLevel('Global', [])],
        reminderDays: 3,
        escalationDays: 5,
      },
    ]);
    expect(onClose).toHaveBeenCalled();
  });

  it('shows every problem at once, in the summary and at each field (FR-003)', async () => {
    const { user } = renderPanel(undefined, [
      reply('POST /api/v1/approval-workflows', {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Fix the problems.',
            fields: [
              { field: 'name', message: 'Enter a name.' },
              { field: 'levels.1.approverIds', message: 'Level 1: choose at least one approver.' },
            ],
          },
        },
      }),
    ]);
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
    const { user, api } = renderPanel(workflowOf(), [
      reply(
        'PATCH /api/v1/approval-workflows/{workflowId}',
        { body: workflowOf({ name: 'Lubricants EU approvals' }) },
        { workflowId: 'WF-1001' },
      ),
    ]);

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
    const { user, api } = renderPanel(workflowOf({ waitingCount: 0 }), [
      reply(
        'PATCH /api/v1/approval-workflows/{workflowId}',
        { body: workflowOf() },
        { workflowId: 'WF-1001' },
      ),
    ]);
    await screen.findAllByRole('combobox', { name: /Approvers/ });

    await user.click(screen.getByRole('button', { name: 'Move level 2 up' }));
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await screen.findByText("Approval workflow 'Lubricants EU approval' saved");
    expect(api.bodiesOf(UPDATE)).toEqual([
      {
        version: 4,
        levels: [sentLevel('Global brand', ['U-1001']), sentLevel('', ['U-1005'])],
      },
    ]);
  });

  it('says who saved first, and takes their version for the next Save (FR-003)', async () => {
    let saves = 0;
    const { user, api } = renderPanel(workflowOf(), [
      reply(
        'PATCH /api/v1/approval-workflows/{workflowId}',
        () => {
          saves += 1;
          return saves === 1
            ? {
                status: 409,
                body: {
                  error: {
                    code: 'version_conflict',
                    message: 'Someone else saved first.',
                    fields: [],
                    conflict: { changedBy: 'Alex Morgan', changedAt: new Date().toISOString() },
                  },
                },
              }
            : { body: workflowOf() };
        },
        { workflowId: 'WF-1001' },
      ),
      reply(
        'GET /api/v1/approval-workflows/{workflowId}',
        { body: detailsOf(workflowOf({ version: 5 })) },
        { workflowId: 'WF-1001' },
      ),
    ]);

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

  it('starts a new workflow with a reminder after 3 days and escalation after 5 (017 FR-007)', async () => {
    renderPanel();
    const section = screen.getByRole('region', { name: 'Reminders and escalation' });

    expect(section).toHaveTextContent('Days are calendar days.');
    expect(within(section).getByRole('checkbox', { name: 'Send a reminder' })).toBeChecked();
    expect(within(section).getByRole('textbox', { name: /Remind after/ })).toHaveValue('3');
    expect(
      within(section).getByRole('checkbox', { name: 'Escalate to the backup approver' }),
    ).toBeChecked();
    expect(within(section).getByRole('textbox', { name: /Escalate after/ })).toHaveValue('5');
    expect(screen.getByRole('dialog')).toHaveTextContent(
      'Levels are checked in order. A level can be needed only sometimes, approved automatically, or need every approver. Nobody approves an activity they own or submitted.',
    );
    await screen.findByRole('combobox', { name: /Approvers/ });
  });

  it('turns reminders on for a workflow without them and sends only that (017 FR-007)', async () => {
    const { user, api } = renderPanel(workflowOf({ reminderDays: null, escalationDays: null }), [
      reply(
        'PATCH /api/v1/approval-workflows/{workflowId}',
        { body: workflowOf() },
        { workflowId: 'WF-1001' },
      ),
    ]);
    const reminder = screen.getByRole('checkbox', { name: 'Send a reminder' });
    expect(reminder).not.toBeChecked();
    expect(screen.queryByRole('textbox', { name: /Remind after/ })).not.toBeInTheDocument();

    await user.click(reminder);
    const days = screen.getByRole('textbox', { name: /Remind after/ });
    await user.clear(days);
    await user.type(days, '2');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await screen.findByText("Approval workflow 'Lubricants EU approval' saved");
    expect(api.bodiesOf(UPDATE)).toEqual([{ version: 4, reminderDays: 2 }]);
  });

  it('says at once when the days are no whole number, without asking the API (017 FR-007)', async () => {
    const { user, api } = renderPanel(workflowOf({ reminderDays: 3, escalationDays: 5 }));

    const days = screen.getByRole('textbox', { name: /Escalate after/ });
    await user.clear(days);
    await user.type(days, '2.5');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(
      await screen.findByRole('link', {
        name: 'Escalate after: enter a whole number of days from 1 to 90.',
      }),
    ).toHaveAttribute('href', '#escalationDays');
    expect(api.bodiesOf(UPDATE)).toEqual([]);
  });

  it('links every rules problem of the API to its field (017 FR-008)', async () => {
    const { user } = renderPanel(undefined, [
      reply('POST /api/v1/approval-workflows', {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Fix the problems.',
            fields: [
              { field: 'levels.1.approverType', message: 'Level 1: choose who approves.' },
              {
                field: 'levels.1.backupApproverId',
                message: 'Level 1: choose a backup approver, or turn escalation off.',
              },
              {
                field: 'escalationDays',
                message: 'Escalate after must be more days than Remind after.',
              },
            ],
          },
        },
      }),
    ]);
    await screen.findByRole('combobox', { name: /Approvers/ });

    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(
      await screen.findByRole('link', { name: 'Level 1: choose who approves.' }),
    ).toHaveAttribute('href', '#levels.1.approverType');
    expect(screen.getByRole('group', { name: 'Who approves' })).toHaveAccessibleDescription(
      'Level 1: choose who approves.',
    );
    expect(
      screen.getByRole('link', {
        name: 'Level 1: choose a backup approver, or turn escalation off.',
      }),
    ).toHaveAttribute('href', '#levels.1.backupApproverId');
    expect(screen.getByRole('textbox', { name: /Escalate after/ })).toHaveAccessibleDescription(
      expect.stringContaining('Escalate after must be more days than Remind after.'),
    );
  });
});
