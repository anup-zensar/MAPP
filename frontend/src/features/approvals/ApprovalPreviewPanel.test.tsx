import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { ApprovalPreviewPanel } from './ApprovalPreviewPanel';
import { ACTIVITY_GONE } from './messages';
import { approvalOf, previewOf } from './testData';

const PREVIEW = 'GET /api/approvals/activities/A-1042';
const APPROVE = 'POST /api/activities/A-1042/approval/approve';
const DECLINE = 'POST /api/activities/A-1042/approval/decline';

function renderPanel(answers: Record<string, MockHandler>) {
  const api = mockApi(answers);
  const onClose = vi.fn();
  const onDecided = vi.fn();
  const rendered = renderWithProviders(
    <ApprovalPreviewPanel activityId="A-1042" onClose={onClose} onDecided={onDecided} />,
    { user: testUser() },
  );
  return { ...rendered, api, onClose, onDecided };
}

describe('ApprovalPreviewPanel (feature 008, US3)', () => {
  it('shows the facts, line items, unassigned planned cost and the chain (FR-032)', async () => {
    renderPanel({ [PREVIEW]: { body: previewOf() } });

    const panel = await screen.findByRole('dialog', { name: 'Launch film' });
    expect(panel).toHaveAccessibleDescription('A-1042 · Waiting at level 1 of 2');
    const details = within(panel).getByRole('region', { name: 'Details' });
    expect(details).toHaveTextContent('FY2026 Lubricants Europe › Spring launch › Brand film');
    expect(details).toHaveTextContent('Priya Raman');
    expect(details).toHaveTextContent('01 Apr – 30 Jun 2026');
    expect(details).toHaveTextContent('$50,000.00');
    expect(details).toHaveTextContent('Video production');
    expect(details).toHaveTextContent('Description—');

    const lineItems = within(panel).getByRole('region', { name: 'Line items' });
    const row = within(lineItems).getByRole('row', { name: /Agency fee/ });
    expect(row).toHaveTextContent('40,000.00 EUR');
    expect(row).toHaveTextContent('$43,200.00');
    expect(row).toHaveTextContent('—');
    expect(lineItems).toHaveTextContent('Unassigned planned cost$6,800.00');
    expect(panel).toHaveTextContent('Carlos Mendes');
    expect(within(panel).getByRole('region', { name: 'Approval history' })).toHaveTextContent(
      'Submitted',
    );
    expect(within(panel).getByRole('link', { name: 'Open activity' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1042#approval',
    );
  });

  it('shows a negative unassigned amount in red, an ongoing activity, and no line items', async () => {
    const preview = previewOf();
    renderPanel({
      [PREVIEW]: {
        body: {
          ...preview,
          activity: { ...preview.activity, ongoing: true, endDate: null },
          lineItems: [],
          unassignedPlannedCost: { amount: '-1200.00', currency: 'USD' },
        },
      },
    });

    const panel = await screen.findByRole('dialog', { name: 'Launch film' });
    expect(panel).toHaveTextContent('Ongoing from 01 Apr 2026');
    expect(panel).toHaveTextContent('No line items yet.');
    expect(within(panel).getByText('-$1,200.00').className).toMatch(/negative/);
  });

  it('approves, says so and lets the page move on (FR-032)', async () => {
    const { user, api, onDecided } = renderPanel({
      [PREVIEW]: { body: previewOf() },
      [APPROVE]: { body: approvalOf({ status: 'approved' }) },
    });

    const panel = await screen.findByRole('dialog', { name: 'Launch film' });
    await user.click(within(panel).getByRole('button', { name: 'Approve' }));
    const dialog = await screen.findByRole('alertdialog');
    expect(dialog).not.toHaveTextContent('You are not an approver');
    await user.click(within(dialog).getByRole('button', { name: 'Approve' }));

    await waitFor(() => expect(onDecided).toHaveBeenCalled());
    expect(api.bodiesOf(APPROVE)).toEqual([{ roundId: 'AR-1001', level: 1 }]);
  });

  it('needs a comment to decline (FR-016)', async () => {
    const { user, api } = renderPanel({ [PREVIEW]: { body: previewOf() } });

    const panel = await screen.findByRole('dialog', { name: 'Launch film' });
    await user.click(within(panel).getByRole('button', { name: 'Decline' }));
    const dialog = await screen.findByRole('alertdialog', { name: "Decline 'Launch film'?" });
    expect(dialog).toHaveTextContent(
      'The activity goes back to Priya Raman, who can fix it and re-submit.',
    );
    await user.click(within(dialog).getByRole('button', { name: 'Decline activity' }));

    expect(dialog).toHaveTextContent('Add a comment explaining why the activity is declined.');
    expect(api.bodiesOf(DECLINE)).toEqual([]);
  });

  it('explains a decision someone made first, with its time (FR-018)', async () => {
    const { user } = renderPanel({
      [PREVIEW]: { body: previewOf() },
      [APPROVE]: {
        status: 409,
        body: {
          code: 'version_conflict',
          message: 'Level 1 was already declined by Carlos Mendes',
          fields: [],
          conflict: { changedBy: 'Carlos Mendes', changedAt: new Date().toISOString() },
        },
      },
    });

    const panel = await screen.findByRole('dialog', { name: 'Launch film' });
    await user.click(within(panel).getByRole('button', { name: 'Approve' }));
    await user.click(
      within(await screen.findByRole('alertdialog')).getByRole('button', { name: 'Approve' }),
    );

    expect(
      await screen.findByText(/^Level 1 was already declined by Carlos Mendes at \d\d:\d\d\.$/),
    ).toBeInTheDocument();
  });

  it('offers no decision once the activity is no longer Submitted', async () => {
    const preview = previewOf();
    const none = { available: false, reason: '' };
    renderPanel({
      [PREVIEW]: {
        body: {
          ...preview,
          activity: { ...preview.activity, status: 'approved', displayStatus: 'approved' },
          approval: approvalOf({
            status: 'approved',
            actions: { submit: none, resubmit: none, approve: none, decline: none, recall: none },
          }),
        },
      },
    });

    const panel = await screen.findByRole('dialog', { name: 'Launch film' });
    expect(panel).toHaveAccessibleDescription('A-1042 · Approved');
    expect(within(panel).queryByRole('button', { name: 'Approve' })).not.toBeInTheDocument();
    expect(within(panel).getByRole('link', { name: 'Open activity' })).toBeInTheDocument();
  });

  it('says when the activity no longer exists, with Close', async () => {
    const { user, onClose } = renderPanel({
      [PREVIEW]: {
        status: 404,
        body: { code: 'not_found', message: ACTIVITY_GONE, fields: [] },
      },
    });

    expect(await screen.findByText(ACTIVITY_GONE)).toBeInTheDocument();
    const panel = screen.getByRole('dialog');
    await user.click(within(panel).getAllByRole('button', { name: 'Close' })[1]);

    expect(onClose).toHaveBeenCalled();
  });
});
