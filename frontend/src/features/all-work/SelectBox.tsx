import styles from './SelectBox.module.css';
import type { AllWorkRow } from './all-work-types';

/** How the Grid ticks rows for bulk actions (012 FR-001, FR-002, FR-005). */
export interface GridSelection {
  canSelect: (row: AllWorkRow) => boolean;
  isSelected: (row: AllWorkRow) => boolean;
  toggle: (row: AllWorkRow, range: boolean) => void;
  page: 'all' | 'some' | 'none';
  togglePage: () => void;
}

const LEVEL_WORDS: Record<string, string> = {
  campaign: 'campaign',
  initiative: 'initiative',
  activity: 'activity',
};

export function RowSelectBox({ row, selection }: { row: AllWorkRow; selection: GridSelection }) {
  if (!selection.canSelect(row)) {
    return <span className={styles.selectSpace} />;
  }
  return (
    <input
      type="checkbox"
      className={styles.select}
      aria-label={`Select ${LEVEL_WORDS[row.level] ?? row.level} '${row.name}'`}
      checked={selection.isSelected(row)}
      onChange={(event) =>
        selection.toggle(
          row,
          'shiftKey' in event.nativeEvent && event.nativeEvent.shiftKey === true,
        )
      }
    />
  );
}

export function PageSelectBox({ selection }: { selection: GridSelection }) {
  return (
    <input
      type="checkbox"
      className={styles.select}
      aria-label="Select all rows on this page"
      checked={selection.page === 'all'}
      ref={(box) => {
        if (box) box.indeterminate = selection.page === 'some';
      }}
      onChange={selection.togglePage}
    />
  );
}
