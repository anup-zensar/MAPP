import { screen, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { BinRow } from './BinRow';
import type { BinRowProps } from './BinRow';
import { binItem, lineItemEntry } from '@/testing/fixtures/recycle-bin';

const PARENT = { type: 'initiative', typeLabel: 'initiative', id: 'I-1009', name: 'Trade shows' };

function renderRow(props: Partial<BinRowProps> = {}) {
  const onRestore = vi.fn();
  const onPurge = vi.fn();
  const rendered = renderWithProviders(
    <ul>
      <BinRow
        item={binItem()}
        containsText=""
        highlighted={false}
        highlightInside={false}
        onRestore={onRestore}
        onPurge={onPurge}
        {...props}
      />
    </ul>,
    { user: testUser() },
  );
  return { ...rendered, onRestore, onPurge };
}

describe('BinRow', () => {
  beforeEach(() => {
    // Monday 19 Oct 2026, noon UTC: "Trade shows" was deleted 2 days ago.
    vi.useFakeTimers({ toFake: ['Date'], now: new Date('2026-10-19T12:00:00Z') });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows the level, name, ID, place, contents and who deleted it when (FR-015)', async () => {
    const { user } = renderRow();

    const row = screen.getByRole('listitem');
    expect(within(row).getByText('INITIATIVE')).toBeInTheDocument();
    expect(within(row).getByRole('heading', { level: 2, name: 'Trade shows' })).toBeVisible();
    expect(within(row).getByText('I-1009')).toBeInTheDocument();
    expect(
      within(row).getByText('Was in FY2026 Lubricants Europe › Helix launch'),
    ).toBeInTheDocument();
    expect(within(row).getByText('1 activity · 1 line item')).toBeInTheDocument();
    expect(within(row).getByText('Deleted by Priya Raman')).toBeInTheDocument();

    const time = within(row).getByRole('button', { name: '2 days ago, 17 Oct 2026, 10:00' });
    await user.hover(time);
    expect(await screen.findByRole('tooltip')).toHaveTextContent('17 Oct 2026, 10:00');
  });

  it('says Nothing inside, and leaves out the place of a top-level record (FR-015)', () => {
    renderRow({ item: lineItemEntry({ path: [], parent: null }) });

    expect(screen.getByText('Nothing inside')).toBeInTheDocument();
    expect(screen.queryByText(/^Was in/)).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Show contents' })).not.toBeInTheDocument();
  });

  it('shows another record type as a chip with its name', () => {
    renderRow({
      item: binItem({ recordType: 'currency', recordTypeLabel: 'currency', contents: [] }),
    });

    expect(screen.getByText('Currency')).toBeInTheDocument();
  });

  it('restores and deletes permanently from the row (FR-015, FR-028)', async () => {
    const { user, onRestore, onPurge } = renderRow();

    await user.click(screen.getByRole('button', { name: "Restore initiative 'Trade shows'" }));
    expect(onRestore).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole('button', { name: "Actions for initiative 'Trade shows'" }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete permanently' }));
    expect(onPurge).toHaveBeenCalledTimes(1);
  });

  it('shows and hides its contents as a tree (FR-015)', async () => {
    const { user } = renderRow();

    const toggle = screen.getByRole('button', { name: 'Show contents' });
    expect(toggle).toHaveAttribute('aria-expanded', 'false');
    await user.click(toggle);

    const activity = screen.getByText("Activity 'Fleet expo stand' (A-1042)");
    expect(activity).toBeVisible();
    expect(within(activity).getByRole('listitem')).toHaveTextContent(
      "Line item 'Stand build' (L-1012)",
    );
    await user.click(screen.getByRole('button', { name: 'Hide contents' }));
    expect(activity).not.toBeVisible();
  });

  it.each([
    ['parentInRecycleBin', "Its initiative 'Trade shows' is also in the recycle bin"],
    [
      'parentDeletedPermanently',
      "Can't be restored — its initiative 'Trade shows' was permanently deleted",
    ],
  ] as const)('says why a %s row cannot simply come back (FR-017)', (restoreState, line) => {
    renderRow({ item: lineItemEntry({ restoreState, missingParent: PARENT }) });

    expect(screen.getByText(line)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: "Restore line item 'Print ads'" })).toBeVisible();
  });

  it('says which record inside matched the search (FR-016)', () => {
    renderRow({ containsText: "Contains A-1042 'Fleet expo stand'" });

    expect(screen.getByText("Contains A-1042 'Fleet expo stand'")).toBeInTheDocument();
  });

  it('marks and focuses the row a link points at, with its contents open (FR-008)', () => {
    renderRow({ highlighted: true, highlightInside: true });

    expect(screen.getAllByRole('listitem')[0]).toHaveAttribute('data-highlighted', 'true');
    expect(screen.getByRole('heading', { name: 'Trade shows' })).toHaveFocus();
    expect(screen.getByRole('button', { name: 'Hide contents' })).toBeInTheDocument();
  });
});
