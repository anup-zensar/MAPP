import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { bulkResult, outcome } from '@/testing/fixtures/bulk';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { reply } from '@/testing/replies';
import { SubmitDialog } from './SubmitDialog';

const RECORDS = [
  { id: 'A-1001', level: 'activity' as const, name: 'Launch film', planId: 'P-1001' },
  { id: 'A-1002', level: 'activity' as const, name: 'Fleet card', planId: 'P-1001' },
];

function openDialog(done: number, skipped: number) {
  mockApi([
    reply('POST /api/v1/bulk-actions/preview', {
      body: bulkResult({
        action: 'submit',
        counts: { done, skipped, failed: 0 },
        outcomes: [outcome()],
      }),
    }),
  ]);
  renderWithProviders(<SubmitDialog records={RECORDS} onClose={vi.fn()} onDone={vi.fn()} />, {
    user: testUser(),
  });
  return screen.findByRole('dialog', { name: 'Submit for approval' });
}

describe('038 FR-043 (L-9): counts in the Submit dialog agree in number', () => {
  it('submits 1 activity, and 1 is not ready', async () => {
    const dialog = await openDialog(1, 1);

    expect(
      await within(dialog).findByRole('button', { name: 'Submit 1 activity' }),
    ).toBeInTheDocument();
    expect(within(dialog).getByText('1 will be submitted · 1 is not ready')).toBeInTheDocument();
  });

  it('submits 3 activities, and 2 are not ready', async () => {
    const dialog = await openDialog(3, 2);

    expect(
      await within(dialog).findByRole('button', { name: 'Submit 3 activities' }),
    ).toBeInTheDocument();
    expect(within(dialog).getByText('3 will be submitted · 2 are not ready')).toBeInTheDocument();
  });
});
