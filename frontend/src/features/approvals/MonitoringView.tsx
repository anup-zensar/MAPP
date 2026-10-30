import { CircleCheckBig } from 'lucide-react';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { subjectKindLabel } from '@/ui/subject-links';
import { formatWaitingTime, localDate } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import { AgeingBars } from './AgeingBars';
import { useApprovalMonitoring } from './api';
import type { ApprovalMonitoring } from './api';
import { ApprovalsTable } from './ApprovalsTable';
import { RunEscalationsButton } from './RunEscalationsButton';
import type { PreviewSubject } from './schemas';
import { useIsPhone } from './useIsPhone';
import styles from './MonitoringView.module.css';

export interface MonitoringViewProps {
  onOpen: (subject: PreviewSubject) => void;
}

export function MonitoringView({ onOpen }: MonitoringViewProps) {
  const monitoring = useApprovalMonitoring();

  if (monitoring.isError) {
    return (
      <Card>
        <ErrorState error={monitoring.error} onRetry={() => void monitoring.refetch()} />
      </Card>
    );
  }
  if (!monitoring.data) {
    return (
      <div className={styles.view}>
        <div className={styles.cards}>
          {[1, 2, 3, 4].map((key) => (
            <Skeleton key={key} height="104px" />
          ))}
        </div>
        <Card>
          <Skeleton lines={5} height="18px" />
        </Card>
      </div>
    );
  }
  return <MonitoringBody data={monitoring.data} onOpen={onOpen} />;
}

function MonitoringBody({ data, onOpen }: { data: ApprovalMonitoring } & MonitoringViewProps) {
  const { cards } = data;
  const nothingWaits = data.ageing.every((bucket) => bucket.count === 0);

  return (
    <div className={styles.view}>
      {data.canRunEscalations && (
        <div className={styles.toolbar}>
          <RunEscalationsButton />
        </div>
      )}
      <div className={styles.cards}>
        <SummaryCard
          label="Overdue"
          value={cards.overdue}
          tone={cards.overdue > 0 ? 'problem' : undefined}
          tip="Approvals that have waited at least the workflow's escalation days (or reminder days without escalation)."
        />
        <SummaryCard
          label="Escalated"
          value={cards.escalated}
          tip="Open approvals whose current level was escalated to a backup approver."
        />
        <SummaryCard
          label="Average time to decision"
          value={cards.averageDecisionDays === null ? '—' : `${cards.averageDecisionDays} days`}
          tip="From submission to final approval or decline, for activities decided this calendar month. Automatic approvals are not counted."
        />
        <SummaryCard
          label="Approved automatically"
          value={cards.autoApprovedThisMonth}
          tip="Activities approved by a rule when they were submitted, this calendar month."
        />
      </div>

      {nothingWaits ? (
        <Card>
          <EmptyState icon={<CircleCheckBig />} title="Nothing is waiting for approval." />
        </Card>
      ) : (
        <>
          <div className={styles.pair}>
            <Card title="How long approvals have waited">
              <AgeingBars buckets={data.ageing} />
            </Card>
            <Card title="Waiting by level">
              <WaitingByLevel data={data} />
            </Card>
          </div>
          <Card title="Overdue">
            {data.overdue.length === 0 ? (
              <p className={styles.empty}>Nothing is overdue.</p>
            ) : (
              <div className={styles.flush}>
                <ApprovalsTable rows={data.overdue} tab="open" onOpen={onOpen} />
              </div>
            )}
          </Card>
        </>
      )}
    </div>
  );
}

function WaitingByLevel({ data }: { data: ApprovalMonitoring }) {
  const format = useFormatters();
  const phone = useIsPhone();
  if (data.byLevel.length === 0) {
    return <p className={styles.empty}>Nothing is waiting for approval.</p>;
  }
  const oldest = (since: string) => (
    <span className={styles.stack}>
      <span>{format.date(localDate(since))}</span>{' '}
      <span className={styles.secondary}>{`waiting ${formatWaitingTime(since)}`}</span>
    </span>
  );
  if (phone) {
    return (
      <ul role="list" className={styles.levels}>
        {data.byLevel.map((row) => (
          <li key={`${row.workflow.id}-${row.level}`} className={styles.levelCard}>
            <p className={styles.levelName}>{`${row.workflow.name} · ${row.levelLabel}`}</p>
            <dl className={styles.facts}>
              <dt>Approves</dt>
              <dd>{subjectKindLabel(row.kind)}</dd>
              <dt>Waiting</dt>
              <dd>{row.waiting}</dd>
              <dt>Overdue</dt>
              <dd>{row.overdue}</dd>
              <dt>Oldest waiting</dt>
              <dd>{oldest(row.oldestWaitingSince)}</dd>
            </dl>
          </li>
        ))}
      </ul>
    );
  }
  return (
    <div
      className={styles.scroll}
      role="region"
      aria-label="Waiting by level, scrolls sideways"
      tabIndex={0}
    >
      <table className={styles.table}>
        <thead>
          <tr>
            <th scope="col" className={styles.workflowCol}>
              Workflow
            </th>
            <th scope="col">Approves</th>
            <th scope="col">Level</th>
            <th scope="col" className={styles.number}>
              Waiting
            </th>
            <th scope="col" className={styles.number}>
              Overdue
            </th>
            <th scope="col">Oldest waiting</th>
          </tr>
        </thead>
        <tbody>
          {data.byLevel.map((row) => (
            <tr key={`${row.workflow.id}-${row.level}`}>
              <th scope="row">{row.workflow.name}</th>
              <td className={styles.nowrap}>{subjectKindLabel(row.kind)}</td>
              <td className={styles.nowrap}>{row.levelLabel}</td>
              <td className={styles.number}>{row.waiting}</td>
              <td className={styles.number}>{row.overdue}</td>
              <td className={styles.nowrap}>{oldest(row.oldestWaitingSince)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
