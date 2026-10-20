import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react';
import { InfoTip } from '@/components';
import { cx } from '@/lib/cx';
import styles from './AllWorkGrid.module.css';
import { COLUMN_TIPS, totalText } from './allWorkText';
import type { AllWorkColumn, AllWorkResult, AllWorkRow, Sort } from './allWorkTypes';
import { CellValue } from './CellValue';
import { isNumeric } from './cellText';
import { GridRow } from './GridRow';
import { ScrollBox } from './ScrollBox';
import type { VisibleRow } from './viewState';

export interface AllWorkGridProps {
  result: AllWorkResult;
  /** The rows of the page on screen (D-115). */
  rows: VisibleRow[];
  /** The path above a page that starts inside a branch ("Continued: …"). */
  continued: string;
  search: string;
  /** The column the children of every parent are sorted by (FR-015); null: plan-tree order. */
  sort: Sort | null;
  /** A click on a sortable heading. */
  onSort: (column: string) => void;
  onToggle: (key: string) => void;
  onOpen: (row: AllWorkRow) => void;
}

/**
 * The Grid (US1, FR-011 – FR-017, plan "Grid"): Name, then the chosen columns; the rows of one
 * page, each opening its record; and the totals row, which closes every page. Read-only (FR-002).
 * The table scrolls sideways inside its card; the names stay on the left.
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

/**
 * A column heading; a sortable one is a button that sorts ascending, then descending, then back to
 * the plan-tree order (FR-015). aria-sort tells screen readers which way. A column whose sum needs
 * explaining has an ⓘ after its words (Unassigned planned cost; functional review X-02).
 */
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
