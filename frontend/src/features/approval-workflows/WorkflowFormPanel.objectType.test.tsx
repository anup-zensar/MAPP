import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { expectNoAxeViolations } from '@/testing/axe';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { PEOPLE, workflowOf } from '@/testing/fixtures/approval-workflows';
import { reply } from '@/testing/replies';
import { WorkflowFormPanel } from './WorkflowFormPanel';
import type { ApprovalWorkflow } from './api';

const CREATE = 'POST /api/v1/approval-workflows';
const UPDATE = 'PATCH /api/v1/approval-workflows/WF-1001';
const FIXED =
  "It approves activities and is in use, so this can't change. Create a new workflow instead.";

function renderPanel(workflow?: ApprovalWorkflow, answers: HttpHandler[] = []) {
  const api = mockApi([reply('GET /api/v1/people', PEOPLE), ...answers]);
  const rendered = renderWithProviders(
    <WorkflowFormPanel workflow={workflow} onClose={vi.fn()} />,
    {
      user: testUser(),
    },
  );
  return { ...rendered, api };
}

const approves = () => screen.getByRole('combobox', { name: /^Approves/ });

describe('WorkflowFormPanel object type (feature 031, US2)', () => {
  it('a new workflow approves activities unless another type is chosen (031 FR-004)', async () => {
    const { user, api } = renderPanel(undefined, [
      reply('POST /api/v1/approval-workflows', {
        status: 201,
        body: workflowOf({ id: 'WF-1003', name: 'PR approval', objectType: 'purchaseRequest' }),
      }),
    ]);
    const panel = screen.getByRole('dialog', { name: 'New approval workflow' });

    expect(approves()).toHaveValue('activity');
    expect(panel).toHaveTextContent('Nobody approves an activity they own or submitted.');
    await user.selectOptions(approves(), 'Purchase requests');
    expect(panel).not.toHaveTextContent('Nobody approves an activity they own or submitted.');
    await user.click(within(panel).getByRole('textbox', { name: /^Name/ }));
    await user.paste('PR approval');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Approval workflow 'PR approval' created")).toBeInTheDocument();
    expect(api.bodiesOf(CREATE)).toEqual([
      expect.objectContaining({ name: 'PR approval', objectType: 'purchaseRequest' }),
    ]);
  });

  it('an unused workflow can change what it approves, and sends only that (031 FR-005)', async () => {
    const unused = workflowOf({ usedByCount: 0, waitingCount: 0, objectTypeFixed: false });
    const { user, api } = renderPanel(unused, [
      reply(
        'PATCH /api/v1/approval-workflows/{workflowId}',
        { body: { ...unused, objectType: 'modifiedPurchaseRequest' } },
        { workflowId: 'WF-1001' },
      ),
    ]);

    await user.selectOptions(approves(), 'Modified purchase requests');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText(/saved$/)).toBeInTheDocument();
    expect(api.bodiesOf(UPDATE)).toEqual([{ objectType: 'modifiedPurchaseRequest', version: 4 }]);
  });

  it('a used workflow shows what it approves fixed, and says why (031 FR-005)', async () => {
    renderPanel(workflowOf());

    expect(approves()).toBeDisabled();
    expect(approves()).toHaveAccessibleDescription(expect.stringContaining(FIXED));
    await expectNoAxeViolations(screen.getByRole('dialog'));
  });
});
