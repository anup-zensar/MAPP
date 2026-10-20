import type { ReactNode } from 'react';
import { PersonLabel, StatusChip } from '@/components';
import { ActivityStatusChip } from '@/features/activities/ActivityStatusChip';
import { formatWaitingTime, localDate } from '@/lib/format';
import { useFormatters } from '@/lib/useFormatters';
import type { Formatters } from '@/lib/useFormatters';
import type { ApprovalRow, ApprovalTab } from './approvalsApi';
import { levelOf, names } from './messages';
import { useIsPhone } from './useIsPhone';
import styles from './ApprovalsTable.module.css';

export interface ApprovalsTableProps {
  rows: ApprovalRow[];
  tab: ApprovalTab;
  /** Opens the preview of the row's activity. */
  onOpen: (activityId: string) => void;
}

/**
 * The rows of the approvals page (FR-030), in the order served: the activity (its name opens the
 * preview; "Start date passed" as a problem chip), plan › campaign, owner, planned cost, workflow,
 * the level it waits at and since when. **Decided by me** adds my decision and the current status,
 * and shows the level only while the round is open.
 *
 * The data table standard of design system §5.4: names never break inside a word, people never
 * wrap, money right-aligned under its header. Below 600 px every row is a stacked card instead
 * (inbox-type list), so nothing scrolls sideways on a phone.
 */
export function ApprovalsTable({ rows, tab, onOpen }: ApprovalsTableProps) {
  const phone = useIsPhone();
  const decided = tab === 'decided';

  if (phone) {
    return (
      <ul role="list" className={styles.cards}>
        {rows.map((row) => (
          <RowCard key={row.activity.id} row={row} decided={decided} onOpen={onOpen} />
        ))}
      </ul>
    );
  }

  return (
    <div className={styles.scroll}>
      <table className={styles.table}>
        <thead>
          <tr>
            <th scope="col" className={styles.nameCol}>
              Activity
            </th>
            <th scope="col" className={styles.pathCol}>
              Plan › campaign
            </th>
            <th scope="col">Owner</th>
            <th scope="col" className={styles.amount}>
              Planned cost
            </th>
            <th scope="col" className={styles.workflowCol}>
              Workflow
            </th>
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
  const chips = rowChips(row);

  return (
    <tr>
      <th scope="row">
        <OpenButton row={row} onOpen={onOpen} />
        <span className={styles.id}>{row.activity.id}</span>
        {chips && <span className={styles.chips}>{chips}</span>}
      </th>
      <td>{pathText(row)}</td>
      <td className={styles.nowrap}>{row.owner ? <PersonLabel person={row.owner} /> : '—'}</td>
      <td className={styles.amount}>{format.money(row.plannedCost)}</td>
      <td>{workflowText(row)}</td>
      <td className={styles.nowrap}>
        <LevelCell row={row} />
      </td>
      <td className={styles.nowrap}>
        <SubmittedCell row={row} format={format} />
      </td>
      {decided && <td className={styles.nowrap}>{decisionText(row, format)}</td>}
      {decided && (
        <td className={styles.nowrap}>
          <ActivityStatusChip status={row.activity.displayStatus} />
        </td>
      )}
    </tr>
  );
}

/** One row as a card on a phone: the name, its chips, then the facts as label and value. */
function RowCard({ row, decided, onOpen }: RowProps) {
  const format = useFormatters();
  const chips = rowChips(row);
  const open = row.round.outcome === 'open';

  return (
    <li className={styles.card}>
      <div className={styles.cardHead}>
        <OpenButton row={row} onOpen={onOpen} />
        <span className={styles.id}>{row.activity.id}</span>
      </div>
      {(decided || chips) && (
        <div className={styles.chips}>
          {decided && <ActivityStatusChip status={row.activity.displayStatus} />}
          {chips}
        </div>
      )}
      <dl className={styles.facts}>
        <Fact label="Plan › campaign">{pathText(row)}</Fact>
        <Fact label="Owner">{row.owner ? <PersonLabel person={row.owner} /> : '—'}</Fact>
        <Fact label="Planned cost">
          <span className={styles.money}>{format.money(row.plannedCost)}</span>
        </Fact>
        <Fact label="Workflow">{workflowText(row)}</Fact>
        {open && (
          <Fact label="Level">
            <LevelCell row={row} />
          </Fact>
        )}
        <Fact label="Submitted">
          <SubmittedCell row={row} format={format} />
        </Fact>
        {decided && <Fact label="My decision">{decisionText(row, format)}</Fact>}
      </dl>
    </li>
  );
}

function Fact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <>
      <dt>{label}</dt>
      <dd>{children}</dd>
    </>
  );
}

/** The activity's name, which opens its preview: a link-coloured button. */
function OpenButton({ row, onOpen }: { row: ApprovalRow; onOpen: (id: string) => void }) {
  return (
    <button type="button" className={styles.open} onClick={() => onOpen(row.activity.id)}>
      {row.activity.name}
    </button>
  );
}

/** "Level 1 of 2" over the level's approvers, each on one line; "—" once the round ended. */
function LevelCell({ row }: { row: ApprovalRow }) {
  if (row.round.outcome !== 'open') {
    return <>—</>;
  }
  return (
    <span className={styles.stack}>
      <span>{levelOf(row.round.currentLevel, row.round.levelCount)}</span>{' '}
      {row.round.approvers.length > 0 && (
        <span className={styles.secondary}>{names(row.round.approvers)}</span>
      )}
    </span>
  );
}

function SubmittedCell({ row, format }: { row: ApprovalRow; format: Formatters }) {
  return (
    <span className={styles.stack}>
      <span>{format.date(localDate(row.submittedAt))}</span>{' '}
      {row.waitingSince && (
        <span className={styles.secondary}>waiting {formatWaitingTime(row.waitingSince)}</span>
      )}
    </span>
  );
}

/** The problem chips of a row; null when it has none (at most two per row, §4.1). */
function rowChips(row: ApprovalRow): ReactNode {
  if (!row.startDatePassed) {
    return null;
  }
  return <StatusChip tone="problem">Start date passed</StatusChip>;
}

function pathText(row: ApprovalRow): string {
  return (
    row.path
      .slice(0, 2)
      .map((record) => record.name)
      .join(' › ') || '—'
  );
}

function workflowText(row: ApprovalRow): string {
  return `${row.workflow.name}${row.workflow.active ? '' : ' (inactive)'}`;
}

function decisionText(row: ApprovalRow, format: Formatters): string {
  return row.myDecision
    ? `${row.myDecision.action === 'approved' ? 'Approved' : 'Declined'} · ${format.date(localDate(row.myDecision.at))}`
    : '—';
}
