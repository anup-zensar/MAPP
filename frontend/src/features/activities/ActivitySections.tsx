import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { ShellAttributeValues } from '@/ui/ShellAttributeFields/ShellAttributeValues';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { Detail, DetailsList, MoneyPair, moneyText } from '@/features/line-items';
import { formatWhole } from '@/format/decimal';
import { localDate } from '@/format/intl';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import type { Activity } from './api';
import styles from './ActivitySections.module.css';
import { EMPTY, goalsFormula } from './activity-text';
import type { ModelFacts } from './activity-text';

/** The Details card: every value, "—" when empty, then who created and last changed it. */
export function ActivityDetails({ activity }: { activity: Activity }) {
  const { date } = useFormatters();
  const type = activity.activityType;
  const workflow = activity.approvalWorkflow;

  return (
    <Card title="Details">
      <div className={styles.stack}>
        <DetailsList>
          <Detail label="Activity name" helpKey="activity.name">
            {activity.name}
          </Detail>
          <Detail label="External name" helpKey="activity.externalName">
            {activity.externalName || (
              <>
                {EMPTY} <span className={styles.muted}>(the activity name is used)</span>
              </>
            )}
          </Detail>
          <Detail label="Campaign" helpKey="activity.initiative">
            {activity.campaign.name}
          </Detail>
          <Detail label="Initiative">{activity.initiative.name}</Detail>
          <Detail label="Activity type" helpKey="activity.activityType">
            {type.name}
            {!type.active && ' (inactive)'}
            {!type.allowed && (
              <span className={styles.problem}>Not allowed by the plan&apos;s model</span>
            )}
          </Detail>
          <Detail label="Owner" helpKey="activity.owner">
            {activity.owner ? <PersonLabel person={activity.owner} /> : null}
          </Detail>
          <Detail label="Start date" helpKey="activity.dates">
            {activity.startDate ? date(activity.startDate) : null}
          </Detail>
          <Detail label="End date">
            {activity.ongoing
              ? 'Ongoing (no end date)'
              : activity.endDate
                ? date(activity.endDate)
                : null}
          </Detail>
          <Detail label="Planned cost" helpKey="activity.plannedCost" numeric>
            {activity.plannedCost ? (
              <MoneyPair primary={activity.plannedCost} secondary={activity.display.plannedCost} />
            ) : null}
          </Detail>
          <Detail label="Approval workflow" helpKey="activity.approvalWorkflow">
            {workflow ? `${workflow.name}${workflow.active ? '' : ' (inactive)'}` : null}
          </Detail>
          <Detail label="Description" helpKey="activity.description">
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
  form: AttributeForm | undefined;
  businessUnits: string[] | undefined;
  error: unknown;
  onRetry: () => void;
}

/** The Shell attributes card; the form's own section names are its headings, never repeated. */
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

/** The Goals card: MQL and revenue "Calculated" from the plan's model (formula in an ⓘ) or "Typed". */
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
        <Detail label="INQ" helpKey="activity.inq" numeric>
          {goals.inq === null ? null : formatWhole(goals.inq)}
        </Detail>
        <Detail label="MQL" helpKey="activity.mql" numeric>
          {goals.mql === null ? null : (
            <>
              {formatWhole(goals.mql)} {origin('MQL')}
            </>
          )}
        </Detail>
        <Detail label="Revenue" helpKey="activity.revenue" numeric>
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
