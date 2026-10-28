import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { detailsOf, workflowOf } from '@/testing/fixtures/approval-workflows';
import {
  renderWorkflowsPage,
  workflowList,
  workflowRow,
} from '@/testing/fixtures/approval-workflows-page';
import { reply } from '@/testing/replies';

const PR_WORKFLOW = workflowOf({
  id: 'WF-1003',
  name: 'Purchase request approval',
  objectType: 'purchaseRequest',
  levels: [
    {
      number: 1,
      name: '',
      label: 'Level 1',
      approvers: [],
      approverType: 'recordField',
      recordField: { id: 'moaApprover', label: 'MoA approver' },
    },
  ],
  levelsText: 'Level 1 · MoA approver',
  usedByCount: 1,
  waitingCount: 0,
});

describe('ApprovalWorkflowsPage object types (feature 031, US2)', () => {
  it('shows what each workflow approves and filters on it, kept in the address (031 FR-006)', async () => {
    const { user, router } = renderWorkflowsPage('/settings/approval-workflows', [
      reply('GET /api/v1/approval-workflows', workflowList([workflowOf(), PR_WORKFLOW])),
    ]);

    const pr = await screen.findByRole('row', { name: /Purchase request approval/ });
    expect(pr).toHaveTextContent('Purchase requests');
    expect(pr).toHaveTextContent('Used by 1 purchase request');
    expect(workflowRow('Lubricants EU approval')).toHaveTextContent('Activities');
    expect(screen.getAllByRole('columnheader').map((header) => header.textContent)).toContain(
      'Approves',
    );

    await user.click(screen.getByRole('button', { name: 'Purchase requests 1' }));

    expect(screen.getByText('1 of 2 approval workflows')).toBeInTheDocument();
    expect(screen.queryByRole('row', { name: /Lubricants EU/ })).not.toBeInTheDocument();
    expect(router.state.location.search).toBe('?approves=purchaseRequest');
    await expectNoAxeViolations();
  });

  it('details name what it approves, who approves and the records that use it (031 FR-009, FR-011)', async () => {
    const details = detailsOf(PR_WORKFLOW);
    renderWorkflowsPage('/settings/approval-workflows/WF-1003', [
      reply('GET /api/v1/approval-workflows', workflowList([PR_WORKFLOW])),
      reply(
        'GET /api/v1/approval-workflows/{workflowId}',
        {
          body: {
            ...details,
            activities: [
              {
                activity: { type: 'purchaseRequest', id: 'PR-1001', name: 'Agency retainer' },
                path: [],
                displayStatus: 'inReview',
                waiting: true,
              },
            ],
          },
        },
        { workflowId: 'WF-1003' },
      ),
    ]);

    const panel = await screen.findByRole('dialog', { name: 'Purchase request approval' });
    expect(panel).toHaveTextContent('Approves purchase requests');
    expect(panel).toHaveTextContent('Person named on the record: MoA approver');
    expect(
      await screen.findByRole('region', { name: 'Purchase requests using it' }),
    ).toHaveTextContent('In review');
    expect(screen.getByRole('link', { name: 'Agency retainer' })).toHaveAttribute(
      'href',
      '/purchase-requests/PR-1001',
    );
  });
});
