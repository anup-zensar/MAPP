import { ExternalLink } from 'lucide-react';
import { useId } from 'react';
import { ApiError } from '@/api/errors';
import {
  Button,
  CompletenessChip,
  ErrorState,
  LinkButton,
  PersonLabel,
  SidePanel,
  Skeleton,
} from '@/components';
import { ActivityStatusChip } from '@/features/activities/ActivityStatusChip';
import { activityStatusWord } from '@/features/activities/activityStatus';
import { formatDate } from '@/lib/format';
import { recordHref } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import { ApprovalChain } from './ApprovalChain';
import { ApprovalHistory } from './ApprovalHistory';
import { useApprovalPreview } from './approvalsApi';
import type { ApprovalPreview } from './approvalsApi';
import { DecisionButton } from './DecisionButton';
import { ACTIVITY_GONE, roundOutcomeText } from './messages';
import styles from './ApprovalPreviewPanel.module.css';

export interface ApprovalPreviewPanelProps {
  activityId: string;
  onClose: () => void;
  /** After Approve or Decline here: the page moves on to the next row. */
  onDecided: () => void;
}

/**
 * The preview of an activity on the approvals page (FR-032), so the approver decides without
 * leaving the page: its facts and completeness, line items, unassigned planned cost, the approval
 * chain and the latest history; **Approve**, **Decline** (while Submitted) and **Open activity**.
 */
export function ApprovalPreviewPanel({
  activityId,
  onClose,
  onDecided,
}: ApprovalPreviewPanelProps) {
  const preview = useApprovalPreview(activityId);
  const gone = preview.error instanceof ApiError && preview.error.kind === 'notFound';
  const data = preview.data;
  const round = data?.approval.rounds[0];
  const planId = data?.activity.path.find((record) => record.type === 'plan')?.id;

  const description = data
    ? `${data.activity.id} · ${
        data.activity.status === 'submitted' && round?.outcome === 'open'
          ? roundOutcomeText(round)
          : activityStatusWord(data.activity.displayStatus)
      }`
    : activityId;

  return (
    <SidePanel
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      title={data?.activity.name ?? 'Activity'}
      description={description}
      footer={
        data && !gone ? (
          <>
            {data.approval.actions.approve.available && (
              <DecisionButton
                mode="approve"
                approval={data.approval}
                ownerName={data.activity.owner?.name ?? null}
                onDecided={onDecided}
              />
            )}
            {data.approval.actions.decline.available && (
              <DecisionButton
                mode="decline"
                approval={data.approval}
                ownerName={data.activity.owner?.name ?? null}
                onDecided={onDecided}
              />
            )}
            <LinkButton
              to={`${recordHref('activity', data.activity.id, planId)}#approval`}
              icon={<ExternalLink />}
            >
              Open activity
            </LinkButton>
          </>
        ) : (
          <Button onClick={onClose}>Close</Button>
        )
      }
    >
      {data ? (
        <PreviewBody preview={data} />
      ) : gone ? (
        <p className={styles.gone}>{ACTIVITY_GONE}</p>
      ) : preview.isError ? (
        <ErrorState error={preview.error} onRetry={() => void preview.refetch()} />
      ) : (
        <Skeleton lines={8} height="18px" />
      )}
    </SidePanel>
  );
}

function PreviewBody({ preview }: { preview: ApprovalPreview }) {
  const format = useFormatters();
  const id = useId();
  const { activity, lineItems, unassignedPlannedCost, approval } = preview;
  const round = approval.rounds[0];
  const negative = unassignedPlannedCost?.amount.startsWith('-') ?? false;
  const dates = activity.ongoing
    ? activity.startDate
      ? `Ongoing from ${formatDate(activity.startDate, format.dateFormat)}`
      : '—'
    : format.dateRange(activity.startDate, activity.endDate);

  return (
    <div className={styles.body}>
      <div className={styles.chips}>
        <ActivityStatusChip status={activity.displayStatus} />
        <CompletenessChip missing={activity.missing} />
      </div>

      <section className={styles.section} aria-labelledby={`${id}-details`}>
        <h3 id={`${id}-details`} className={styles.heading}>
          Details
        </h3>
        <dl className={styles.facts}>
          <dt>Plan › campaign › initiative</dt>
          <dd>{activity.path.map((record) => record.name).join(' › ') || '—'}</dd>
          <dt>Owner</dt>
          <dd>{activity.owner ? <PersonLabel person={activity.owner} /> : '—'}</dd>
          <dt>Dates</dt>
          <dd>{dates}</dd>
          <dt>Planned cost</dt>
          <dd>{format.money(activity.plannedCost)}</dd>
          <dt>Activity type</dt>
          <dd>{activity.typeName || '—'}</dd>
          <dt>Description</dt>
          <dd className={styles.description}>{activity.description || '—'}</dd>
        </dl>
      </section>

      <section className={styles.section} aria-labelledby={`${id}-line-items`}>
        <h3 id={`${id}-line-items`} className={styles.heading}>
          Line items
        </h3>
        {lineItems.length === 0 ? (
          <p className={styles.muted}>No line items yet.</p>
        ) : (
          <div className={styles.scroll}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th scope="col">Line item</th>
                  <th scope="col" className={styles.amount}>
                    Planned cost
                  </th>
                  <th scope="col" className={styles.amount}>
                    Planned cost (USD)
                  </th>
                  <th scope="col">Spend type</th>
                  <th scope="col">GL code</th>
                  <th scope="col">Finance budget item</th>
                </tr>
              </thead>
              <tbody>
                {lineItems.map((item) => (
                  <tr key={item.id}>
                    <th scope="row">
                      {item.name}
                      <span className={styles.id}>{item.id}</span>
                    </th>
                    <td className={styles.amount}>{format.money(item.plannedCost)}</td>
                    <td className={styles.amount}>{format.money(item.plannedCostUsd)}</td>
                    <td>{item.spendType || '—'}</td>
                    <td>{item.glCode || '—'}</td>
                    <td>{item.financeBudgetItem || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        {unassignedPlannedCost && (
          <p className={styles.unassigned}>
            <span>Unassigned planned cost</span>
            <span className={negative ? styles.negative : styles.amountText}>
              {format.money(unassignedPlannedCost)}
            </span>
          </p>
        )}
      </section>

      {round && <ApprovalChain round={round} />}
      <ApprovalHistory rounds={approval.rounds} compact />
    </div>
  );
}
