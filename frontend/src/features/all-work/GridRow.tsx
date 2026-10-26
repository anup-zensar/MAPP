import type { MouseEvent } from 'react';
import { cx } from '@/ui/cx';
import styles from './AllWorkGrid.module.css';
import type { AllWorkColumn, AllWorkRow } from './all-work-types';
import { CellValue } from './CellValue';
import { isControl, isNumeric, isSearched } from './cell-text';
import { RowName } from './RowName';
import type { VisibleRow } from './view-state';

export interface GridRowProps {
  visible: VisibleRow;
  columns: AllWorkColumn[];
  search: string;
  /** The Status column is hidden, so the chip shows after the name (D-118). */
  statusInName: boolean;
  onToggle: (key: string) => void;
  onOpen: (row: AllWorkRow) => void;
}

/** One row of the Grid: a click anywhere but on a control opens the record (FR-016). */
export function GridRow({
  visible,
  columns,
  search,
  statusInName,
  onToggle,
  onOpen,
}: GridRowProps) {
  const { row } = visible;

  function openFromRow(event: MouseEvent<HTMLTableRowElement>) {
    if (!isControl(event.target)) {
      onOpen(row);
    }
  }

  return (
    <tr
      data-row-key={row.key}
      className={cx(
        styles.row,
        row.role === 'context' && styles.context,
        row.level === 'unassigned' && styles.unassignedRow,
      )}
      onClick={openFromRow}
    >
      <th scope="row" className={cx(styles.nameCell, styles[row.level])}>
        <RowName
          visible={visible}
          search={search}
          showStatus={statusInName}
          onToggle={onToggle}
          onOpen={onOpen}
        />
      </th>
      {columns.map((column, index) => (
        <td key={column.key} className={cx(isNumeric(column) && styles.end)}>
          <CellValue cell={row.cells[index]} search={isSearched(column.key) ? search : ''} />
        </td>
      ))}
    </tr>
  );
}
