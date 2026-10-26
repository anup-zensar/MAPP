import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ALEX, DANIEL } from '@/features/approvals/testData';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { CHINA_WORKFLOW, detailsOf, workflowOf } from './testData';
import { WorkflowDetailsPanel } from './WorkflowDetailsPanel';
import { WORKFLOW_GONE } from './workflowTexts';

const DETAILS = 'GET /api/v1/approval-workflows/WF-1001';

function renderPanel(answer: MockHandler) {
  mockApi({ [DETAILS]: answer });
  const actions = { edit: vi.fn(), setInactive: vi.fn(), setActive: vi.fn(), remove: vi.fn() };
  const onClose = vi.fn();
  const rendered = renderWithProviders(
    <WorkflowDetailsPanel workflowId="WF-1001" actions={actions} onClose={onClose} />,
    { user: testUser() },
  );
  return { ...rendered, actions, onClose };
}

describe('WorkflowDetailsPanel (feature 008, US4)', () => {
  it('shows the use, the levels and the activities that use it, with links (FR-005)', async () => {
    renderPanel({ body: detailsOf() });

    const panel = await screen.findByRole('dialog', { name: 'Lubricants EU approval' });
    expect(panel).toHaveAccessibleDescription('Used by 3 activities (2 waiting for approval)');
    expect(within(panel).getByText('Active')).toBeInTheDocument();
    expect(panel).toHaveTextContent('Every Lubricants Europe activity.');
    const levels = within(panel).getByRole('region', { name: 'Levels' });
    expect(levels).toHaveTextContent('Level 1');
    expect(levels).toHaveTextContent('Carlos Mendes');
    expect(levels).toHaveTextContent('Global brand');
    const using = within(panel).getByRole('region', { name: 'Activities using it' });
    expect(within(using).getByRole('link', { name: 'Launch film' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1042',
    );
    expect(using).toHaveTextContent('A-1042 · FY2026 Lubricants Europe · Spring launch');
    expect(using).toHaveTextContent('Waiting for approval');
    expect(panel).toHaveTextContent(
      'Created by Alex Morgan on 05 Jan 2026, last changed by Alex Morgan on 02 Mar 2026',
    );
  });

  it('warns about a level without active approver, marked "(inactive)" (edge case)', async () => {
    const workflow = workflowOf({
      levels: [
        workflowOf().levels[0],
        { ...workflowOf().levels[1], approvers: [{ ...ALEX, active: false }] },
      ],
      warnings: ['Level 2 has no active approver'],
    });
    renderPanel({ body: detailsOf(workflow) });

    const panel = await screen.findByRole('dialog', { name: 'Lubricants EU approval' });
    expect(panel).toHaveTextContent('Level 2 has no active approver');
    expect(panel).toHaveTextContent(
      'Anyone can still decide on their behalf. Edit the workflow to choose an active approver.',
    );
    expect(panel).toHaveTextContent('Alex Morgan (inactive)');
  });

  it('for an inactive, unused workflow: says so and offers Reactivate', async () => {
    const { user, actions } = renderPanel({
      body: { workflow: CHINA_WORKFLOW, activities: [] },
    });

    const panel = await screen.findByRole('dialog', { name: 'China BC approval workflow' });
    expect(panel).toHaveAccessibleDescription('Not used yet');
    expect(panel).toHaveTextContent(
      'Inactive — not offered on activities. Activities that use it keep it.',
    );
    expect(panel).toHaveTextContent('No activities use it yet.');
    await user.click(
      within(panel).getByRole('button', { name: 'Actions for China BC approval workflow' }),
    );
    await user.click(await screen.findByRole('menuitem', { name: 'Reactivate' }));
    expect(actions.setActive).toHaveBeenCalledWith(CHINA_WORKFLOW);
  });

  it('offers Edit, then Set inactive and Delete in the panel footer menu', async () => {
    const { user, actions } = renderPanel({ body: detailsOf() });

    const panel = await screen.findByRole('dialog', { name: 'Lubricants EU approval' });
    expect(within(panel).queryByRole('button', { name: 'Delete' })).not.toBeInTheDocument();
    await user.click(within(panel).getByRole('button', { name: 'Edit' }));
    const more = within(panel).getByRole('button', { name: 'Actions for Lubricants EU approval' });
    await user.click(more);
    expect(
      within(screen.getByRole('menu'))
        .getAllByRole('menuitem')
        .map((item) => item.firstChild?.textContent),
    ).toEqual(['Set inactive', 'Delete']);
    await user.click(screen.getByRole('menuitem', { name: 'Set inactive' }));
    await user.click(more);
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));

    expect(actions.edit).toHaveBeenCalled();
    expect(actions.setInactive).toHaveBeenCalled();
    expect(actions.remove).toHaveBeenCalled();
  });

  it('says when the workflow no longer exists', async () => {
    renderPanel({
      status: 404,
      body: { error: { code: 'not_found', message: WORKFLOW_GONE, fields: [] } },
    });

    expect(await screen.findByText(WORKFLOW_GONE)).toBeInTheDocument();
  });

  it('writes each level rule and the timings in words (017 FR-009)', async () => {
    const workflow = workflowOf({
      levels: [
        {
          number: 1,
          name: '',
          label: 'Level 1',
          approvers: [],
          approverType: 'manager',
          role: null,
          mode: 'any',
          appliesWhen: null,
          autoApproveWhen: {
            plannedCostAtLeast: null,
            plannedCostBelow: { amount: '10000.00', currency: 'USD' },
            businessUnits: [],
            activityTypes: [],
            text: 'planned cost is below $10,000.00',
          },
          backupApprover: ALEX,
        },
        {
          number: 2,
          name: 'Global brand',
          label: 'Global brand',
          approvers: [],
          approverType: 'role',
          role: { id: 'RL-1005', name: 'Approver', builtIn: 'approver' },
          mode: 'all',
          appliesWhen: {
            plannedCostAtLeast: { amount: '100000.00', currency: 'USD' },
            plannedCostBelow: null,
            businessUnits: [],
            activityTypes: [],
            text: 'planned cost is at least $100,000.00',
          },
          autoApproveWhen: null,
          backupApprover: { ...DANIEL, active: false },
        },
      ],
      reminderDays: 3,
      escalationDays: 5,
    });
    renderPanel({ body: detailsOf(workflow) });

    const levels = await screen.findByRole('region', { name: 'Levels' });
    const items = within(levels)
      .getAllByRole('listitem')
      .filter((item) => item.closest('ol'));
    const first = items.find((item) => item.textContent?.startsWith('L1')) as HTMLElement;
    expect(first).toHaveTextContent("The owner's line manager");
    expect(first).toHaveTextContent('Approved automatically when planned cost is below $10,000.00');
    expect(first).toHaveTextContent('Backup approver: Alex Morgan');
    const second = items.find((item) => item.textContent?.startsWith('L2')) as HTMLElement;
    expect(second).toHaveTextContent('Role: Approver');
    expect(second).toHaveTextContent('All must approve');
    expect(second).toHaveTextContent('Only needed when planned cost is at least $100,000.00');
    expect(second).toHaveTextContent('Backup approver: Daniel Okoro (inactive)');
    expect(levels).toHaveTextContent('Reminder after 3 days · escalation after 5 days');
  });

  it('says when a workflow sends no reminders (017 FR-009)', async () => {
    renderPanel({ body: detailsOf(workflowOf({ reminderDays: null, escalationDays: null })) });

    expect(await screen.findByRole('region', { name: 'Levels' })).toHaveTextContent(
      'No reminders or escalation',
    );
  });
});
