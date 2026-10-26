import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { ShellAttributeValues } from '@/ui/ShellAttributeFields/ShellAttributeValues';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { Detail, DetailsList } from '@/features/line-items/DetailsList';
import { moneyText } from '@/features/line-items/lineItemMath';
import { MoneyPair } from '@/features/line-items/MoneyPair';
import { formatWhole } from '@/lib/decimal';
import { localDate } from '@/lib/format';
import { recordHref } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import type { Activity } from './activitiesApi';
import styles from './ActivitySections.module.css';
import { EMPTY, goalsFormula } from './activityText';
import type { ModelFacts } from './activityText';

/**
 * The Details card of an activity's page (FR-020): every value in the one key–value layout, "—"
 * when empty, then who created and last changed it (design system §6.3).
 */
export function ActivityDetails({ activity }: { activity: Activity }) {
  const { date } = useFormatters();
  const type = activity.activityType;
  const workflow = activity.approvalWorkflow;

  return (
    <Card title="Details">
      <div className={styles.stack}>
        <DetailsList>
          <Detail label="Activity name">{activity.name}</Detail>
          <Detail label="External name">
            {activity.externalName || (
              <>
                {EMPTY} <span className={styles.muted}>(the activity name is used)</span>
              </>
            )}
          </Detail>
          <Detail label="Campaign">{activity.campaign.name}</Detail>
          <Detail label="Initiative">{activity.initiative.name}</Detail>
          <Detail label="Activity type">
            {type.name}
            {!type.active && ' (inactive)'}
            {!type.allowed && (
              <span className={styles.problem}>Not allowed by the plan&apos;s model</span>
            )}
          </Detail>
          <Detail label="Owner">
            {activity.owner ? <PersonLabel person={activity.owner} /> : null}
          </Detail>
          <Detail label="Start date">{activity.startDate ? date(activity.startDate) : null}</Detail>
          <Detail label="End date">
            {activity.ongoing
              ? 'Ongoing (no end date)'
              : activity.endDate
                ? date(activity.endDate)
                : null}
          </Detail>
          {/* In the plan currency, with the reader's display currency beside it (016 FR-013). */}
          <Detail label="Planned cost" numeric>
            {activity.plannedCost ? (
              <MoneyPair primary={activity.plannedCost} secondary={activity.display.plannedCost} />
            ) : null}
          </Detail>
          <Detail label="Approval workflow">
            {workflow ? `${workflow.name}${workflow.active ? '' : ' (inactive)'}` : null}
          </Detail>
          <Detail label="Description">
            {activity.description ? (
              <span className={styles.description}>{activity.description}</span>
            ) : null}
          </Detail>
        </DetailsList>
        <p className={styles.changed}>
          Created by {activity.createdBy.name} on {date(localDate(activity.createdAt))}, last
          changed by {activity.updatedBy.name} on {date(localDate(activity.updatedAt))}
        </p>
      </div>
    </Card>
  );
}

export interface ActivityAttributesProps {
  activity: Activity;
  /** The activity level's Shell attribute form (feature 002), once loaded. */
  form: AttributeForm | undefined;
  /** The plan's business units (option ids), once loaded. */
  businessUnits: string[] | undefined;
  /** Why the form or the business units did not load. */
  error: unknown;
  onRetry: () => void;
}

/**
 * The Shell attributes card (FR-014, FR-015): feature 002's read-only values, the groups shown for
 * the plan's business unit with the reason, "—" when empty. The form's own section names are the
 * card's headings ("Shell attributes", "Lubricants attributes"), so no heading repeats its parent
 * (UI audit AF-02, functional review P-08).
 */
export function ActivityAttributes({
  activity,
  form,
  businessUnits,
  error,
  onRetry,
}: ActivityAttributesProps) {
  return (
    <section aria-label="Shell attributes" className={styles.attributes}>
      {form && businessUnits ? (
        <ShellAttributeValues
          form={form}
          values={activity.attributes}
          businessUnit={businessUnits}
          planLink={recordHref('plan', activity.plan.id)}
          sectionHeading="h2"
        />
      ) : (
        <>
          <h2 className={styles.heading}>Shell attributes</h2>
          {error ? <ErrorState error={error} onRetry={onRetry} /> : <Skeleton lines={4} />}
        </>
      )}
    </section>
  );
}

/**
 * The Goals card (FR-018): INQ, MQL and revenue, MQL and revenue "Calculated" from the plan's
 * model (the formula in an ⓘ once the model is known) or "Typed"; "—" when empty.
 */
export function ActivityGoals({ activity, model }: { activity: Activity; model?: ModelFacts }) {
  const { goals } = activity;
  const origin = (stage: string) =>
    goals.calculated ? (
      <span className={styles.origin}>
        Calculated
        {model && <InfoTip label={`${stage} calculation`} text={goalsFormula(model)} />}
      </span>
    ) : (
      <span className={styles.origin}>Typed</span>
    );

  return (
    <Card title="Goals">
      <DetailsList>
        <Detail label="INQ" numeric>
          {goals.inq === null ? null : formatWhole(goals.inq)}
        </Detail>
        <Detail label="MQL" numeric>
          {goals.mql === null ? null : (
            <>
              {formatWhole(goals.mql)} {origin('MQL')}
            </>
          )}
        </Detail>
        <Detail label="Revenue" numeric>
          {activity.display.revenue === null ? null : (
            <>
              {moneyText(activity.display.revenue)} {origin('Revenue')}
            </>
          )}
        </Detail>
      </DetailsList>
    </Card>
  );
}
