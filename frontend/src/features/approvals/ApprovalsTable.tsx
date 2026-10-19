import { PersonLabel, StatusChip } from '@/components';
import { ActivityStatusChip } from '@/features/activities/ActivityStatusChip';
import { formatWaitingTime, localDate } from '@/lib/format';
import { useFormatters } from '@/lib/useFormatters';
import type { ApprovalRow, ApprovalTab } from './approvalsApi';
import { levelOf, names } from './messages';
import styles from './ApprovalsTable.module.css';

export interface ApprovalsTableProps {
  rows: ApprovalRow[];
  tab: ApprovalTab;
  /** Opens the preview of the row's activity. */
  onOpen: (activityId: string) => void;
}

/**
 * The rows of the approvals page (FR-030), in the order served: the activity (its name opens the
 * preview; "Start date passed" in red), plan › campaign, owner, planned cost, workflow, the level it
 * waits at and since when. **Decided by me** adds my decision and the current status, and shows
 * the level only while the round is open. Scrolls sideways inside its card on narrow screens.
 */
export function ApprovalsTable({ rows, tab, onOpen }: ApprovalsTableProps) {
  const decided = tab === 'decided';

  return (
    <div className={styles.scroll}>
      <table className={styles.table}>
        <thead>
          <tr>
            <th scope="col">Activity</th>
            <th scope="col">Plan › campaign</th>
            <th scope="col">Owner</th>
            <th scope="col" className={styles.amount}>
              Planned cost
            </th>
            <th scope="col">Workflow</th>
            <th scope="col">Level</th>
            <th scope="col">Submitted</th>
            {decided && <th scope="col">My decision</th>}
            {decided && <th scope="col">Status</th>}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <Row key={row.activity.id} row={row} decided={decided} onOpen={onOpen} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

interface RowProps {
  row: ApprovalRow;
  decided: boolean;
  onOpen: (activityId: string) => void;
}

function Row({ row, decided, onOpen }: RowProps) {
  const format = useFormatters();
  const open = row.round.outcome === 'open';

  return (
    <tr>
      <th scope="row">
        <button type="button" className={styles.open} onClick={() => onOpen(row.activity.id)}>
          {row.activity.name}
        </button>
        <span className={styles.id}>{row.activity.id}</span>
        {row.startDatePassed && (
          <span className={styles.chip}>
            <StatusChip tone="problem">Start date passed</StatusChip>
          </span>
        )}
      </th>
      <td>
        {row.path
          .slice(0, 2)
          .map((record) => record.name)
          .join(' › ') || '—'}
      </td>
      <td>{row.owner ? <PersonLabel person={row.owner} /> : '—'}</td>
      <td className={styles.amount}>{format.money(row.plannedCost)}</td>
      <td>{`${row.workflow.name}${row.workflow.active ? '' : ' (inactive)'}`}</td>
      <td>
        {open
          ? `${levelOf(row.round.currentLevel, row.round.levelCount)} · ${names(row.round.approvers)}`
          : '—'}
      </td>
      <td>
        <span className={styles.nowrap}>{format.date(localDate(row.submittedAt))}</span>
        {row.waitingSince && (
          <span className={styles.secondary}>waiting {formatWaitingTime(row.waitingSince)}</span>
        )}
      </td>
      {decided && (
        <td>
          {row.myDecision
            ? `${row.myDecision.action === 'approved' ? 'Approved' : 'Declined'} · ${format.date(localDate(row.myDecision.at))}`
            : '—'}
        </td>
      )}
      {decided && (
        <td>
          <ActivityStatusChip status={row.activity.displayStatus} />
        </td>
      )}
    </tr>
  );
}
