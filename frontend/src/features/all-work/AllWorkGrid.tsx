import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { cx } from '@/ui/cx';
import styles from './AllWorkGrid.module.css';
import { COLUMN_TIPS, totalText } from './all-work-text';
import type { AllWorkColumn, AllWorkResult, AllWorkRow, Sort } from './all-work-types';
import { CellValue } from './CellValue';
import { isNumeric } from './cell-text';
import { GridRow } from './GridRow';
import { ScrollBox } from './ScrollBox';
import { PageSelectBox } from './SelectBox';
import type { GridSelection } from './SelectBox';
import type { VisibleRow } from './view-rows';

export interface AllWorkGridProps {
  result: AllWorkResult;
  rows: VisibleRow[];
  continued: string;
  search: string;
  sort: Sort | null;
  onSort: (column: string) => void;
  onToggle: (key: string) => void;
  onOpen: (row: AllWorkRow) => void;
  selection?: GridSelection;
}

/**
 * The Grid (010 FR-011 – FR-017): Name, then the chosen columns; the rows of one page, each opening its
 * record, ticked for bulk actions with `selection` (012); the totals row closes every page.
 */
export function AllWorkGrid({
  result,
  rows,
  continued,
  search,
  sort,
  onSort,
  onToggle,
  onOpen,
  selection,
}: AllWorkGridProps) {
  const { columns, totals } = result;
  const statusInName = !columns.some((column) => column.key === 'status');

  return (
    <div className={styles.grid}>
      {continued && <p className={styles.continued}>Continued: {continued}</p>}
      <ScrollBox label="All work grid">
        <table className={styles.table}>
          <caption className="visually-hidden">All work grid</caption>
          <thead>
            <tr>
              <th scope="col" className={styles.nameHead}>
                {selection && <PageSelectBox selection={selection} />}
                Name
              </th>
              {columns.map((column) => (
                <ColumnHeading key={column.key} column={column} sort={sort} onSort={onSort} />
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((visible) => (
              <GridRow
                key={visible.row.key}
                visible={visible}
                columns={columns}
                search={search}
                statusInName={statusInName}
                onToggle={onToggle}
                onOpen={onOpen}
                selection={selection}
              />
            ))}
          </tbody>
          <tfoot>
            <tr className={styles.totals}>
              <th scope="row" className={styles.nameCell}>
                {totalText(totals.activities)}
              </th>
              {columns.map((column, index) => (
                <td key={column.key} className={cx(isNumeric(column) && styles.end)}>
                  <CellValue cell={totals.cells[index]} />
                </td>
              ))}
            </tr>
          </tfoot>
        </table>
      </ScrollBox>
    </div>
  );
}

interface ColumnHeadingProps {
  column: AllWorkColumn;
  sort: Sort | null;
  onSort: (column: string) => void;
}

/** A sortable heading sorts ascending, descending, then back to the plan-tree order (FR-015). */
function ColumnHeading({ column, sort, onSort }: ColumnHeadingProps) {
  const direction = sort?.column === column.key ? sort.direction : null;
  const Icon = direction === 'asc' ? ArrowUp : direction === 'desc' ? ArrowDown : ArrowUpDown;
  const tip = COLUMN_TIPS[column.key];
  const label = column.sortable ? (
    <button
      type="button"
      className={cx(styles.sort, direction && styles.sorted)}
      onClick={() => onSort(column.key)}
    >
      {column.label}
      <Icon aria-hidden="true" />
    </button>
  ) : (
    column.label
  );
  return (
    <th
      scope="col"
      className={cx(isNumeric(column) && styles.end)}
      aria-sort={
        direction === 'asc' ? 'ascending' : direction === 'desc' ? 'descending' : undefined
      }
    >
      {tip ? (
        <span className={styles.headWithTip}>
          {label}
          <InfoTip label={column.label} text={tip} />
        </span>
      ) : (
        label
      )}
    </th>
  );
}
