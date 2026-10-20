import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ALEX } from '@/features/approvals/testData';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { CHINA_WORKFLOW, detailsOf, workflowOf } from './testData';
import { WorkflowDetailsPanel } from './WorkflowDetailsPanel';
import { WORKFLOW_GONE } from './workflowTexts';

const DETAILS = 'GET /api/approval-workflows/WF-1001';

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

  it('for an inactive, unused workflow: says so and offers Set active', async () => {
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
    await user.click(await screen.findByRole('menuitem', { name: 'Set active' }));
    expect(actions.setActive).toHaveBeenCalledWith(CHINA_WORKFLOW);
  });

  it('offers Edit as the primary action, Close, and Set inactive and Delete in ⋯ (design system §5.3)', async () => {
    const { user, actions, onClose } = renderPanel({ body: detailsOf() });

    const panel = await screen.findByRole('dialog', { name: 'Lubricants EU approval' });
    expect(within(panel).queryByRole('button', { name: 'Delete' })).not.toBeInTheDocument();
    await user.click(within(panel).getByRole('button', { name: 'Edit' }));
    const more = within(panel).getByRole('button', { name: 'Actions for Lubricants EU approval' });
    await user.click(more);
    await user.click(await screen.findByRole('menuitem', { name: 'Set inactive' }));
    await user.click(more);
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const closeButtons = within(panel).getAllByRole('button', { name: 'Close' });
    await user.click(closeButtons[closeButtons.length - 1]);

    expect(actions.edit).toHaveBeenCalled();
    expect(actions.setInactive).toHaveBeenCalled();
    expect(actions.remove).toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });

  it('says when the workflow no longer exists', async () => {
    renderPanel({
      status: 404,
      body: { code: 'not_found', message: WORKFLOW_GONE, fields: [] },
    });

    expect(await screen.findByText(WORKFLOW_GONE)).toBeInTheDocument();
  });
});
