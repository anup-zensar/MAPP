import type { ReactNode } from 'react';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { ActivityStatusChip } from '@/ui/ActivityStatusChip/ActivityStatusChip';
import { formatWaitingTime, localDate } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import type { Formatters } from '@/format/useFormatters';
import type { ApprovalRow, ApprovalTab } from './api';
import { levelOf, names, progressText } from './messages';
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
 * preview; its problems in one chip), plan › campaign, owner, planned cost, workflow,
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
        <DelegateNote row={row} />
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
      {decided && (
        <td className={styles.nowrap}>
          <DecisionCell row={row} format={format} />
        </td>
      )}
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
        <DelegateNote row={row} />
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
        {decided && (
          <Fact label="My decision">
            <DecisionCell row={row} format={format} />
          </Fact>
        )}
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

/**
 * "Level 1 of 2" over the level's approvers, and "1 of 3 approved" on a level where all must
 * approve (017 FR-031); "—" once the round ended.
 */
function LevelCell({ row }: { row: ApprovalRow }) {
  const { round } = row;
  if (round.outcome !== 'open') {
    return <>—</>;
  }
  const all = round.mode === 'all' && (round.requiredCount ?? 0) > 1;
  return (
    <span className={styles.stack}>
      <span>{levelOf(round.currentLevel, round.levelCount)}</span>{' '}
      {round.approvers.length > 0 && (
        <span className={styles.secondary}>{names(round.approvers)}</span>
      )}
      {all && (
        <>
          {' '}
          <span className={styles.secondary}>
            {progressText(round.approvedCount ?? 0, round.requiredCount ?? 0)}
          </span>
        </>
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

/**
 * The chips of a row (008, 017 FR-031), at most two (design system §4.1): one problem chip with the
 * row's problems most pressing first — "No approver", "Overdue", "Start date passed" — and
 * "Escalated to <name>" (waiting); null when it has none.
 */
function rowChips(row: ApprovalRow): ReactNode {
  const open = row.round.outcome === 'open';
  const problems = [
    open && row.noApprover && 'No approver',
    open && row.overdue && 'Overdue',
    row.startDatePassed && 'Start date passed',
  ].filter((word): word is string => Boolean(word));
  const chips = [
    problems.length > 0 && (
      <StatusChip key="problem" tone="problem">
        {problems.join(' · ')}
      </StatusChip>
    ),
    open && row.escalatedTo && (
      <StatusChip key="escalated" tone="review">
        {`Escalated to ${row.escalatedTo.name}`}
      </StatusChip>
    ),
  ].filter(Boolean);
  return chips.length > 0 ? chips : null;
}

/**
 * "as delegate of <names>" when the person decides this row for someone else (017 FR-031): a note
 * under the activity's ID, not a chip, as it says who decides rather than how the approval stands.
 */
function DelegateNote({ row }: { row: ApprovalRow }) {
  if (row.round.outcome !== 'open' || !row.asDelegateOf || row.asDelegateOf.length === 0) {
    return null;
  }
  return <span className={styles.delegate}>{`as delegate of ${names(row.asDelegateOf)}`}</span>;
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

/** My decision over its date ("Approved" over "12 Mar 2026"), so the Decided table fits its card. */
function DecisionCell({ row, format }: { row: ApprovalRow; format: Formatters }) {
  if (!row.myDecision) {
    return <>—</>;
  }
  return (
    <span className={styles.stack}>
      <span>{row.myDecision.action === 'approved' ? 'Approved' : 'Declined'}</span>{' '}
      <span className={styles.secondary}>{format.date(localDate(row.myDecision.at))}</span>
    </span>
  );
}
