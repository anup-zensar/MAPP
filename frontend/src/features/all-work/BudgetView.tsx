import type { MouseEvent } from 'react';
import { cx } from '@/lib/cx';
import { useFormatters } from '@/lib/useFormatters';
import gridStyles from './AllWorkGrid.module.css';
import { totalText } from './allWorkText';
import type { AllWorkResult, AllWorkRow, Money } from './allWorkTypes';
import styles from './BudgetView.module.css';
import { isControl, NOTHING } from './cellText';
import { RowName } from './RowName';
import type { VisibleRow } from './viewState';

export interface BudgetViewProps {
  /** The answer for view=budget: its periods and each row's amounts. */
  result: AllWorkResult;
  /** The rows of the page on screen (D-115). */
  rows: VisibleRow[];
  continued: string;
  search: string;
  onToggle: (key: string) => void;
  onOpen: (row: AllWorkRow) => void;
}

/**
 * Budget & cost (US4, FR-022 – FR-024): the Grid's rows with Plan budget, the months or quarters
 * of the year (then Other years and No dates when they hold an amount) and Total. Every amount
 * comes from the API, already spread and added up (D-114). Read-only; a row opens its record.
 */
export function BudgetView({ result, rows, continued, search, onToggle, onOpen }: BudgetViewProps) {
  const { periods, totals } = result;

  return (
    <div className={gridStyles.grid}>
      <p className={styles.note}>To change how a cost is spread, open its line item</p>
      {continued && <p className={gridStyles.continued}>Continued: {continued}</p>}
      <div className={gridStyles.scroll}>
        <table className={gridStyles.table}>
          <caption className="visually-hidden">Budget & cost</caption>
          <thead>
            <tr>
              <th scope="col" className={gridStyles.nameHead}>
                Name
              </th>
              <th scope="col" className={gridStyles.end}>
                Plan budget
              </th>
              {periods.map((period) => (
                <th key={period.key} scope="col" className={gridStyles.end}>
                  {period.label}
                </th>
              ))}
              <th scope="col" className={gridStyles.end}>
                Total
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map((visible) => (
              <BudgetRow
                key={visible.row.key}
                visible={visible}
                periodCount={periods.length}
                search={search}
                onToggle={onToggle}
                onOpen={onOpen}
              />
            ))}
          </tbody>
          <tfoot>
            <tr className={gridStyles.totals}>
              <th scope="row" className={gridStyles.nameCell}>
                {totalText(totals.activities)}
              </th>
              <td className={gridStyles.end}>
                <Amount value={totals.planBudget} />
              </td>
              {periods.map((period, index) => (
                <td key={period.key} className={gridStyles.end}>
                  <Amount value={totals.periods?.[index]} />
                </td>
              ))}
              <td className={gridStyles.end}>
                <Amount value={totals.total} />
              </td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>
  );
}

interface BudgetRowProps {
  visible: VisibleRow;
  periodCount: number;
  search: string;
  onToggle: (key: string) => void;
  onOpen: (row: AllWorkRow) => void;
}

function BudgetRow({ visible, periodCount, search, onToggle, onOpen }: BudgetRowProps) {
  const { money } = useFormatters();
  const { row } = visible;
  const isContainer =
    row.level === 'plan' || row.level === 'campaign' || row.level === 'initiative';
  // A negative balance is not spread; the activity says so instead (FR-023).
  const exceeded =
    row.level === 'activity' && row.balance?.amount.startsWith('-') ? row.balance : undefined;

  function openFromRow(event: MouseEvent<HTMLTableRowElement>) {
    if (!isControl(event.target)) {
      onOpen(row);
    }
  }

  return (
    <tr
      data-row-key={row.key}
      className={cx(
        gridStyles.row,
        row.role === 'context' && gridStyles.context,
        row.level === 'unassigned' && gridStyles.unassignedRow,
      )}
      onClick={openFromRow}
    >
      <th scope="row" className={cx(gridStyles.nameCell, gridStyles[row.level])}>
        <RowName visible={visible} search={search} showStatus onToggle={onToggle} onOpen={onOpen}>
          {exceeded && (
            <span className={styles.exceeded}>
              Line items exceed the planned cost by{' '}
              {money({ amount: exceeded.amount.slice(1), currency: exceeded.currency })}
            </span>
          )}
        </RowName>
      </th>
      <td className={gridStyles.end}>{isContainer && <Amount value={row.budget ?? null} />}</td>
      {Array.from({ length: periodCount }, (_, index) => (
        <td key={index} className={gridStyles.end}>
          <Amount value={row.periods?.[index]} />
        </td>
      ))}
      <td className={gridStyles.end}>
        <Amount value={row.total} />
      </td>
    </tr>
  );
}

/** An amount of the view, "—" when there is none (FR-024), red with a minus sign when negative. */
function Amount({ value }: { value: Money | null | undefined }) {
  const { money } = useFormatters();
  if (!value) {
    return <span className={styles.empty}>{NOTHING}</span>;
  }
  return (
    <span className={cx(styles.amount, value.amount.startsWith('-') && styles.problem)}>
      {money(value)}
    </span>
  );
}
