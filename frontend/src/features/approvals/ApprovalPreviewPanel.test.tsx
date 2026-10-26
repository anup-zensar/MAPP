import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import type { MockHandler } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { ApprovalPreviewPanel } from './ApprovalPreviewPanel';
import { ACTIVITY_GONE } from './messages';
import { approvalOf, previewOf, roundOf, SUBMITTED } from '@/testing/fixtures/approvals';

const PREVIEW = 'GET /api/v1/approvals/activities/A-1042';
const APPROVE = 'POST /api/v1/activities/A-1042/approval/approve';
const DECLINE = 'POST /api/v1/activities/A-1042/approval/decline';

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
    // The second amount is in the reader's display currency, as the API converted it (D-213).
    expect(
      within(lineItems).getByRole('columnheader', { name: 'Planned cost (USD)' }),
    ).toBeInTheDocument();
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

  it('shows each line item as a stacked card on a phone, so nothing scrolls sideways (design system §5.4)', async () => {
    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: query === '(max-width: 599px)',
      media: query,
      addEventListener: () => {},
      removeEventListener: () => {},
    }));
    renderPanel({ [PREVIEW]: { body: previewOf() } });

    const panel = await screen.findByRole('dialog', { name: 'Launch film' });
    const lineItems = within(panel).getByRole('region', { name: 'Line items' });
    expect(within(lineItems).queryByRole('table')).not.toBeInTheDocument();
    const card = within(lineItems)
      .getAllByRole('listitem')
      .find((item) => item.textContent?.includes('Agency fee')) as HTMLElement;
    const facts = within(card)
      .getAllByRole('term')
      .map((term) => `${term.textContent}: ${term.nextElementSibling?.textContent}`);
    expect(facts).toEqual([
      'Planned cost: 40,000.00 EUR',
      'Planned cost (USD): $43,200.00',
      expect.stringMatching(/^Spend type: /),
      expect.stringMatching(/^GL code: /),
      expect.stringMatching(/^Finance budget item: /),
    ]);
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
          error: {
            code: 'version_conflict',
            message: 'Level 1 was already declined by Carlos Mendes',
            fields: [],
            conflict: { changedBy: 'Carlos Mendes', changedAt: new Date().toISOString() },
          },
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

  it('shows an approval restarted by a change, without a decision (FR-022)', async () => {
    const preview = previewOf();
    const none = { available: false, reason: '' };
    const restarted = roundOf({
      outcome: 'restarted',
      endedAt: '2026-03-12T10:15:00Z',
      levels: [{ ...roundOf().levels[0], state: 'notDecided' }, roundOf().levels[1]],
      history: [
        {
          ...SUBMITTED,
          id: 'AH-1002',
          action: 'restarted',
          comment: '',
          at: '2026-03-12T10:15:00Z',
          changes: [
            {
              label: 'Planned cost',
              oldText: '$50,000.00',
              newText: '$65,000.00',
              text: 'Planned cost changed from $50,000.00 to $65,000.00',
            },
          ],
        },
        SUBMITTED,
      ],
    });
    renderPanel({
      [PREVIEW]: {
        body: {
          ...preview,
          activity: { ...preview.activity, status: 'created', displayStatus: 'created' },
          approval: approvalOf({
            status: 'created',
            displayStatus: 'created',
            rounds: [restarted],
            actions: {
              submit: { available: true, reason: '' },
              resubmit: none,
              approve: none,
              decline: none,
              recall: none,
            },
          }),
        },
      },
    });

    const panel = await screen.findByRole('dialog', { name: 'Launch film' });
    expect(panel).toHaveAccessibleDescription('A-1042 · Created');
    expect(panel).toHaveTextContent('Not decided — approval restarted');
    const history = within(panel).getByRole('region', { name: 'Approval history' });
    expect(history).toHaveTextContent(
      'Approval restarted · Round 1 · Level 1 · 12 Mar 2026, 10:15',
    );
    expect(history).toHaveTextContent('Planned cost changed from $50,000.00 to $65,000.00');
    expect(within(panel).queryByRole('button', { name: 'Approve' })).not.toBeInTheDocument();
    expect(within(panel).queryByRole('button', { name: 'Decline' })).not.toBeInTheDocument();
  });

  it('explains an approval restarted after the preview was opened (FR-018, FR-022)', async () => {
    const { user, onDecided } = renderPanel({
      [PREVIEW]: { body: previewOf() },
      [DECLINE]: {
        status: 409,
        body: {
          error: {
            code: 'version_conflict',
            message: 'Approval was restarted by Priya Raman',
            fields: [],
            conflict: { changedBy: 'Priya Raman', changedAt: '2026-03-11T10:15:00Z' },
          },
        },
      },
    });

    const panel = await screen.findByRole('dialog', { name: 'Launch film' });
    await user.click(within(panel).getByRole('button', { name: 'Decline' }));
    const dialog = await screen.findByRole('alertdialog');
    await user.type(
      within(dialog).getByRole('textbox', { name: /Why are you declining it\?/ }),
      'Too expensive',
    );
    await user.click(within(dialog).getByRole('button', { name: 'Decline activity' }));

    expect(
      await screen.findByText(
        /^Approval was restarted by Priya Raman on 11 Mar 2026 at \d\d:\d\d\.$/,
      ),
    ).toBeInTheDocument();
    expect(onDecided).not.toHaveBeenCalled();
  });

  it('says when the activity no longer exists, with Close', async () => {
    const { user, onClose } = renderPanel({
      [PREVIEW]: {
        status: 404,
        body: { error: { code: 'not_found', message: ACTIVITY_GONE, fields: [] } },
      },
    });

    expect(await screen.findByText(ACTIVITY_GONE)).toBeInTheDocument();
    const panel = screen.getByRole('dialog');
    await user.click(within(panel).getAllByRole('button', { name: 'Close' })[1]);

    expect(onClose).toHaveBeenCalled();
  });
});
