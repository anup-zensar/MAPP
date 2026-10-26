import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { mockApi } from '@/testing/mockApi';
import { testUser } from '@/testing/fixtures/users';
import { ImportHistoryPanel } from './ImportHistoryPanel';
import { DEMO_IMPORT } from '@/testing/fixtures/financials';

const FILE_IMPORT = {
  ...DEMO_IMPORT,
  id: 'FIM-1002',
  source: 'file' as const,
  feedName: 'Simulated GSAP extract',
  fileName: 'gsap-2026-10-19.csv',
  importedAt: '2026-10-20T09:00:00Z',
  importedBy: { id: 'U-1001', name: 'Alex Morgan' },
  counts: { rows: 4, new: 2, updated: 1, unchanged: 1, refused: 0, notLinked: 1, relinked: 2 },
};

describe('ImportHistoryPanel (015 FR-008)', () => {
  it('lists every import newest first with its file, source, who, when and counts', async () => {
    mockApi({
      'GET /api/v1/financials/imports': { body: { items: [FILE_IMPORT, DEMO_IMPORT], total: 2 } },
    });
    renderWithProviders(<ImportHistoryPanel onClose={vi.fn()} />, { user: testUser() });

    const panel = await screen.findByRole('dialog', { name: 'Import history' });
    // The list takes the focus, so the keyboard can scroll a long history (axe scrollable-region).
    expect(
      await within(panel).findByRole('list', { name: 'Imports, newest first' }),
    ).toHaveAttribute('tabindex', '0');
    const items = await within(panel).findAllByRole('listitem');
    expect(items).toHaveLength(2);
    expect(items[0]).toHaveTextContent('gsap-2026-10-19.csv');
    expect(items[0]).toHaveTextContent('Simulated GSAP extract · FIM-1002');
    expect(items[0]).toHaveTextContent(/by Alex Morgan/);
    expect(items[0]).toHaveTextContent(
      '4 rows: 2 new, 1 updated, 1 unchanged, 0 refused · 1 not linked',
    );
    expect(items[0]).toHaveTextContent('2 earlier transactions are now linked');
    expect(items[1]).toHaveTextContent('Demo data (simulated GSAP)');
    expect(items[1]).toHaveTextContent('11 rows: 11 new');
  });

  it('says when nothing was ever imported', async () => {
    mockApi({ 'GET /api/v1/financials/imports': { body: { items: [], total: 0 } } });
    renderWithProviders(<ImportHistoryPanel onClose={vi.fn()} />, { user: testUser() });

    expect(await screen.findByText('No GSAP data loaded yet.')).toBeInTheDocument();
  });
});
