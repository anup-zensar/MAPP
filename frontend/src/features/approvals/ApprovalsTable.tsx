import type { ReactNode } from 'react';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { SubjectStatusChip } from '@/ui/SubjectStatusChip/SubjectStatusChip';
import { formatWaitingTime, localDate } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import type { Formatters } from '@/format/useFormatters';
import type { ApprovalRow, ApprovalTab } from './api';
import { progressText } from './chain-messages';
import { levelOf, names } from './messages';
import { useIsPhone } from './useIsPhone';
import styles from './ApprovalsTable.module.css';

export interface ApprovalsTableProps {
  rows: ApprovalRow[];
  tab: ApprovalTab;
  onOpen: (activityId: string) => void;
}

export function ApprovalsTable({ rows, tab, onOpen }: ApprovalsTableProps) {
  const phone = useIsPhone();
  const decided = tab === 'decided';

  if (phone) {
    return (
      <ul role="list" className={styles.cards}>
        {rows.map((row) => (
          <RowCard
            key={`${row.subject.kind}-${row.subject.id}`}
            row={row}
            decided={decided}
            onOpen={onOpen}
          />
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
              Record
            </th>
            <th scope="col">Record type</th>
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
            <Row
              key={`${row.subject.kind}-${row.subject.id}`}
              row={row}
              decided={decided}
              onOpen={onOpen}
            />
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
        <span className={styles.id}>{row.subject.id}</span>
        <DelegateNote row={row} />
        {chips && <span className={styles.chips}>{chips}</span>}
      </th>
      <td className={styles.nowrap}>{row.subject.label}</td>
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
          <SubjectStatusChip status={row.subject.displayStatus} />
        </td>
      )}
    </tr>
  );
}

function RowCard({ row, decided, onOpen }: RowProps) {
  const format = useFormatters();
  const chips = rowChips(row);
  const open = row.round.outcome === 'open';

  return (
    <li className={styles.card}>
      <div className={styles.cardHead}>
        <OpenButton row={row} onOpen={onOpen} />
        <span className={styles.id}>{row.subject.id}</span>
        <DelegateNote row={row} />
      </div>
      {(decided || chips) && (
        <div className={styles.chips}>
          {decided && <SubjectStatusChip status={row.subject.displayStatus} />}
          {chips}
        </div>
      )}
      <dl className={styles.facts}>
        <Fact label="Record type">{row.subject.label}</Fact>
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

/** Every record opens in its preview panel: an activity's, or a purchase request's (032 FR-029). */
function OpenButton({ row, onOpen }: { row: ApprovalRow; onOpen: (id: string) => void }) {
  const { subject } = row;
  return (
    <button type="button" className={styles.open} onClick={() => onOpen(subject.id)}>
      {subject.name}
    </button>
  );
}

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
