import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { LogEntry } from './LogEntry';
import type { ActivityLogEntry } from './logText';
import { logEntry } from '@/testing/fixtures/activity-log';

function renderEntry(entry: ActivityLogEntry) {
  return renderWithProviders(
    <ol>
      <LogEntry entry={entry} />
    </ol>,
    { user: testUser() },
  );
}

describe('LogEntry', () => {
  it('shows the sentence, the record’s ID, its path at the time, the time and the action chip (FR-005)', () => {
    renderEntry(logEntry());

    const entry = screen.getByRole('listitem');
    expect(entry).toHaveTextContent("Priya Raman changed activity 'Film shoot'A-1042");
    expect(within(entry).getByText('PR')).toBeInTheDocument();
    expect(
      within(entry).getByText('in FY2026 Lubricants Europe › Helix launch'),
    ).toBeInTheDocument();
    expect(within(entry).getByRole('button', { name: '19 Oct 2026, 09:15' })).toHaveTextContent(
      '09:15',
    );
    expect(within(entry).getByText('Changed')).toBeInTheDocument();
  });

  it('shows the full date and time in a tooltip on the time (FR-005)', async () => {
    const { user } = renderEntry(logEntry());

    await user.hover(screen.getByRole('button', { name: '19 Oct 2026, 09:15' }));

    expect(await screen.findByRole('tooltip')).toHaveTextContent('19 Oct 2026, 09:15');
  });

  it('names an emptied recycle bin with its item count, without a record (FR-006)', () => {
    renderEntry(
      logEntry({
        action: 'emptiedRecycleBin',
        verb: 'emptied',
        path: [],
        recordNow: { state: 'none', name: '', path: [], binEntryId: '' },
        items: [
          {
            recordType: 'lineItem',
            recordTypeLabel: 'line item',
            recordId: 'L-1004',
            recordName: 'Print',
            details: '',
          },
        ],
      }),
    );

    expect(screen.getByRole('listitem')).toHaveTextContent(
      'Priya Raman emptied the recycle bin (1 item)',
    );
    expect(screen.queryByText('A-1042')).not.toBeInTheDocument();
    expect(screen.getByText('Permanently deleted')).toBeInTheDocument();
  });

  it('shows and hides the details: changes, comment and what went with a delete (FR-007)', async () => {
    const { user } = renderEntry(
      logEntry({
        action: 'statusChanged',
        verb: 'declined',
        changes: [
          {
            field: 'status',
            label: 'Status',
            old: 'submitted',
            new: 'declined',
            oldText: 'Submitted',
            newText: 'Declined',
          },
        ],
        comment: 'Budget too high',
      }),
    );

    const toggle = screen.getByRole('button', { name: 'Show details' });
    expect(toggle).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByText('Status: Submitted → Declined')).not.toBeVisible();

    await user.click(toggle);

    expect(screen.getByRole('button', { name: 'Hide details' })).toHaveAttribute(
      'aria-expanded',
      'true',
    );
    expect(screen.getByText('Status: Submitted → Declined')).toBeVisible();
    expect(screen.getByText('Comment: Budget too high')).toBeVisible();

    await user.click(screen.getByRole('button', { name: 'Hide details' }));
    expect(screen.getByText('Comment: Budget too high')).not.toBeVisible();
  });

  it('has no details button when there is nothing more to show (FR-007)', () => {
    renderEntry(logEntry());

    expect(screen.queryByRole('button', { name: 'Show details' })).not.toBeInTheDocument();
  });

  it('links the record to its page, with its current name when renamed (FR-008)', () => {
    renderEntry(
      logEntry({
        recordNow: {
          state: 'exists',
          name: 'Film shoot (final)',
          path: [{ type: 'plan', id: 'P-1001', name: 'FY2026' }],
          binEntryId: '',
        },
      }),
    );

    expect(screen.getByRole('link', { name: 'Film shoot' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1042',
    );
    expect(screen.getByRole('listitem')).toHaveTextContent(
      "'Film shoot' — now 'Film shoot (final)'",
    );
  });

  it('links a record in the recycle bin to its highlighted row (FR-008)', () => {
    renderEntry(
      logEntry({
        action: 'deleted',
        verb: 'deleted',
        recordNow: { state: 'inRecycleBin', name: 'Film shoot', path: [], binEntryId: 'BIN-1003' },
      }),
    );

    expect(screen.getByRole('link', { name: 'Film shoot' })).toHaveAttribute(
      'href',
      '/recycle-bin?record=A-1042',
    );
    expect(screen.getByRole('listitem')).toHaveTextContent("'Film shoot' (in the recycle bin)");
  });

  it('shows a permanently deleted record as plain text (FR-008)', () => {
    renderEntry(
      logEntry({
        recordNow: { state: 'deletedPermanently', name: '', path: [], binEntryId: '' },
      }),
    );

    expect(screen.queryByRole('link')).not.toBeInTheDocument();
    expect(screen.getByRole('listitem')).toHaveTextContent("'Film shoot' (permanently deleted)");
  });

  it('offers no way to edit or remove an entry (FR-010)', () => {
    renderEntry(logEntry({ comment: 'A reason' }));

    const buttons = screen.getAllByRole('button').map((button) => button.textContent);
    expect(buttons).toEqual(['Show details', '09:15']);
  });
});
