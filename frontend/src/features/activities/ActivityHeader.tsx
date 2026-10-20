import { CircleAlert } from 'lucide-react';
import type { ReactNode } from 'react';
import { useNavigate } from 'react-router';
import { Breadcrumb, CompletenessChip, InfoTip, LevelTag, PersonLabel } from '@/components';
import { localDate } from '@/lib/format';
import { recordHref } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import type { Activity } from './activitiesApi';
import { activityBannerText } from './activityStatus';
import { ActivityStatusChip } from './ActivityStatusChip';
import styles from './ActivityHeader.module.css';
import { activityEditHref, datesText, EMPTY, lineItemsAttentionText } from './activityText';

export interface ActivityHeaderProps {
  activity: Activity;
  /** Edit, the status action and More. */
  actions: ReactNode;
  /** Opens the Ready to submit panel and moves the focus there. */
  onOpenReady: () => void;
}

/**
 * The top of an activity's page (FR-019 – FR-021, 006 plan "Activity page"): where it sits, its
 * level, name, status and completeness, ID, owner, dates, planned cost and unassigned planned
 * cost, who created and last changed it, the actions, and the banner of an activity under approval.
 */
export function ActivityHeader({ activity, actions, onOpenReady }: ActivityHeaderProps) {
  const navigate = useNavigate();
  const format = useFormatters();
  const { date, money } = format;
  const { plan, campaign, initiative } = activity;
  const banner = activityBannerText(activity.displayStatus);
  // Amounts in the reader's display currency (D-220).
  const { plannedCost, unassignedPlannedCost: unassigned } = activity.display;
  const overspent = unassigned?.amount.startsWith('-') ?? false;

  return (
    <>
      <Breadcrumb
        items={[
          { label: 'Plans', to: '/plans' },
          { label: plan.name, to: recordHref('plan', plan.id) },
          { label: campaign.name, to: recordHref('campaign', campaign.id, plan.id) },
          { label: initiative.name, to: recordHref('initiative', initiative.id, plan.id) },
          { label: activity.name },
        ]}
      />
      <header className={styles.header}>
        <div className={styles.titleRow}>
          <div className={styles.text}>
            <LevelTag level="activity" />
            <h1 className={styles.title}>{activity.name}</h1>
            <div className={styles.chips}>
              <ActivityStatusChip status={activity.displayStatus} />
              <CompletenessChip
                missing={activity.readiness.activity.map((problem) => ({
                  id: problem.field,
                  label: problem.message,
                }))}
                onSelectField={(field) =>
                  void navigate(activityEditHref(plan.id, activity.id, field))
                }
                itemsBelow={activity.completeness.lineItemsNeedingAttention}
                itemsBelowText={lineItemsAttentionText}
                onItemsBelowSelect={onOpenReady}
              />
              <span className={styles.id}>{activity.id}</span>
            </div>
          </div>
          <div className={styles.actions}>{actions}</div>
        </div>

        <dl className={styles.facts}>
          <div className={styles.fact}>
            <dt>Owner</dt>
            <dd>{activity.owner ? <PersonLabel person={activity.owner} /> : EMPTY}</dd>
          </div>
          <div className={styles.fact}>
            <dt>Dates</dt>
            <dd>{datesText(activity.startDate, activity.endDate, activity.ongoing, format)}</dd>
          </div>
          <div className={styles.fact}>
            <dt>Planned cost</dt>
            <dd className={styles.money}>{money(plannedCost)}</dd>
          </div>
          <div className={styles.fact}>
            <dt className={styles.withTip}>
              Unassigned planned cost
              <InfoTip
                label="Unassigned planned cost"
                text="Planned cost minus the line items' planned costs in USD."
              />
            </dt>
            <dd className={overspent ? styles.problem : styles.money}>
              {money(unassigned)}
              {overspent && unassigned && (
                <span className={styles.problemNote}>
                  <CircleAlert aria-hidden="true" />
                  Line items exceed the activity&apos;s planned cost by{' '}
                  {money({ ...unassigned, amount: unassigned.amount.slice(1) })}
                </span>
              )}
            </dd>
          </div>
        </dl>
        <p className={styles.changed}>
          Created by {activity.createdBy.name} on {date(localDate(activity.createdAt))}, last
          changed by {activity.updatedBy.name} on {date(localDate(activity.updatedAt))}
        </p>
      </header>
      {banner && <p className={styles.banner}>{banner}</p>}
    </>
  );
}
