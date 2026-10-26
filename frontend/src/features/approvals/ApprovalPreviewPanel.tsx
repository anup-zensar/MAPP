import { ExternalLink } from 'lucide-react';
import { useId } from 'react';
import { ApiError } from '@/api/errors';
import { Button, LinkButton } from '@/ui/Button/Button';
import { CompletenessChip } from '@/ui/CompletenessChip/CompletenessChip';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { ActivityStatusChip } from '@/features/activities/ActivityStatusChip';
import { activityStatusWord } from '@/features/activities/activityStatus';
import { formatDate } from '@/format/intl';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import { ApprovalChain } from './ApprovalChain';
import { ApprovalHistory } from './ApprovalHistory';
import { useApprovalPreview } from './approvalsApi';
import type { ApprovalPreview } from './approvalsApi';
import { DecisionButton } from './DecisionButton';
import { OwnerDecisionMenu } from './DecisionActions';
import { decisionRole } from './decisionRole';
import { ACTIVITY_GONE, roundOutcomeText } from './messages';
import { useIsPhone } from './useIsPhone';
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
          <PreviewFooter preview={data} planId={planId} onDecided={onDecided} />
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

interface PreviewFooterProps {
  preview: ApprovalPreview;
  planId: string | undefined;
  onDecided: () => void;
}

/**
 * The panel's buttons, one primary chosen by the person's role (design system §2.5, §6.3): an
 * approver sees **Decline · Approve** with Approve last and primary; the owner never sees Approve
 * as a button (functional review A-01) — where the rules let them decide, it waits in ⋯. The
 * footer lines its buttons up from the right, so they are listed primary first.
 */
function PreviewFooter({ preview, planId, onDecided }: PreviewFooterProps) {
  const { approval, activity } = preview;
  const role = decisionRole(approval);
  return (
    <>
      {role === 'decide' && approval.actions.approve.available && (
        <DecisionButton
          mode="approve"
          approval={approval}
          variant="primary"
          onDecided={onDecided}
        />
      )}
      {role === 'decide' && approval.actions.decline.available && (
        <DecisionButton mode="decline" approval={approval} onDecided={onDecided} />
      )}
      <LinkButton
        to={`${recordHref('activity', activity.id, planId)}#approval`}
        icon={<ExternalLink />}
      >
        Open activity
      </LinkButton>
      {role === 'owner' && <OwnerDecisionMenu approval={approval} onDecided={onDecided} />}
    </>
  );
}

function PreviewBody({ preview }: { preview: ApprovalPreview }) {
  const format = useFormatters();
  const id = useId();
  const { activity, lineItems, unassignedPlannedCost, approval } = preview;
  const round = approval.rounds[0];
  // Why this person cannot decide, when the API says so (014, 017); empty hides it.
  const reason =
    activity.status === 'submitted' && !approval.actions.approve.available
      ? approval.actions.approve.reason
      : '';
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
      {reason && <p className={styles.muted}>{reason}</p>}

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
          <LineItems items={lineItems} />
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

type PreviewLineItem = ApprovalPreview['lineItems'][number];

/**
 * The activity's line items: a table that scrolls sideways inside the panel when it is wider, and
 * below 600 px one stacked card per line item, so nothing scrolls sideways on a phone (design
 * system §5.4).
 */
function LineItems({ items }: { items: PreviewLineItem[] }) {
  const format = useFormatters();
  const phone = useIsPhone();
  const converted = `Planned cost (${format.displayCurrency})`;

  if (phone) {
    return (
      <ul role="list" className={styles.items}>
        {items.map((item) => (
          <li key={item.id} className={styles.item}>
            <p className={styles.itemName}>
              {item.name}
              <span className={styles.id}>{item.id}</span>
            </p>
            <dl className={styles.itemFacts}>
              <dt>Planned cost</dt>
              <dd className={styles.amountText}>{format.money(item.plannedCost)}</dd>
              <dt>{converted}</dt>
              <dd className={styles.amountText}>{format.money(item.plannedCostDisplay)}</dd>
              <dt>Spend type</dt>
              <dd>{item.spendType || '—'}</dd>
              <dt>GL code</dt>
              <dd>{item.glCode || '—'}</dd>
              <dt>Finance budget item</dt>
              <dd>{item.financeBudgetItem || '—'}</dd>
            </dl>
          </li>
        ))}
      </ul>
    );
  }

  return (
    // Focusable, so the keyboard can scroll it sideways when it is wider than the panel.
    <div
      className={styles.scroll}
      role="region"
      aria-label="Line items, scrolls sideways"
      tabIndex={0}
    >
      <table className={styles.table}>
        <thead>
          <tr>
            <th scope="col">Line item</th>
            <th scope="col" className={styles.amount}>
              Planned cost
            </th>
            <th scope="col" className={styles.amount}>
              {converted}
            </th>
            <th scope="col">Spend type</th>
            <th scope="col">GL code</th>
            <th scope="col">Finance budget item</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.id}>
              <th scope="row">
                {item.name}
                <span className={styles.id}>{item.id}</span>
              </th>
              <td className={styles.amount}>{format.money(item.plannedCost)}</td>
              <td className={styles.amount}>{format.money(item.plannedCostDisplay)}</td>
              <td>{item.spendType || '—'}</td>
              <td>{item.glCode || '—'}</td>
              <td>{item.financeBudgetItem || '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
