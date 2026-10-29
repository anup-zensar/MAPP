import type { BulkLevel, SelectedRecord, Selection } from '@/features/bulk';
import type { AllWorkRow } from './all-work-types';
import type { GridSelection } from './SelectBox';

function levelOf(row: AllWorkRow): BulkLevel | null {
  switch (row.level) {
    case 'campaign':
    case 'initiative':
    case 'activity':
      return row.level;
    default:
      return null;
  }
}

function recordOf(row: AllWorkRow): SelectedRecord | null {
  const level = levelOf(row);
  return level ? { id: row.recordId, level, name: row.name, planId: row.planId } : null;
}

export function recordsOf(rows: AllWorkRow[]): SelectedRecord[] {
  return rows.flatMap((row) => recordOf(row) ?? []);
}

export function matchingRecords(rows: AllWorkRow[]): SelectedRecord[] {
  return recordsOf(rows.filter((row) => row.role === 'match'));
}

/** The header checkbox ticks the page's matching rows, never those shown for context (D-134). */
export function gridSelection(selection: Selection, pageRows: AllWorkRow[]): GridSelection {
  const onPage = recordsOf(pageRows);
  const matchingOnPage = matchingRecords(pageRows);
  const ticked = matchingOnPage.filter((r) => selection.isSelected(r.id)).length;
  const page = ticked === 0 ? 'none' : ticked === matchingOnPage.length ? 'all' : 'some';
  return {
    canSelect: (row) => levelOf(row) !== null,
    isSelected: (row) => selection.isSelected(row.recordId),
    toggle: (row, range) => {
      const record = recordOf(row);
      if (record) selection.toggle(record, range ? onPage : undefined);
    },
    page,
    togglePage: () =>
      page === 'all' ? selection.unselect(matchingOnPage) : selection.select(matchingOnPage),
  };
}
