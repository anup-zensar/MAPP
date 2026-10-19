import type { ReactNode } from 'react';
import {
  Card,
  ErrorState,
  InfoTip,
  PersonLabel,
  ShellAttributeValues,
  Skeleton,
} from '@/components';
import type { AttributeForm } from '@/components';
import { formatWhole } from '@/lib/decimal';
import { recordHref } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import type { Activity } from './activitiesApi';
import styles from './ActivitySections.module.css';
import { EMPTY, goalsFormula } from './activityText';
import type { ModelFacts } from './activityText';

/** The Details card of an activity's page (FR-020): every value, "—" when empty. */
export function ActivityDetails({ activity }: { activity: Activity }) {
  const { date, money } = useFormatters();
  const type = activity.activityType;
  const workflow = activity.approvalWorkflow;

  return (
    <Card title="Details">
      <dl className={styles.values}>
        <Pair label="Activity name">{activity.name}</Pair>
        <Pair label="External name">
          {activity.externalName || (
            <>
              {EMPTY} <span className={styles.muted}>(the activity name is used)</span>
            </>
          )}
        </Pair>
        <Pair label="Campaign">{activity.campaign.name}</Pair>
        <Pair label="Initiative">{activity.initiative.name}</Pair>
        <Pair label="Activity type">
          {type.name}
          {!type.active && ' (inactive)'}
          {!type.allowed && (
            <span className={styles.problem}>Not allowed by the plan&apos;s model</span>
          )}
        </Pair>
        <Pair label="Owner">
          {activity.owner ? <PersonLabel person={activity.owner} /> : EMPTY}
        </Pair>
        <Pair label="Start date">{activity.startDate ? date(activity.startDate) : EMPTY}</Pair>
        <Pair label="End date">
          {activity.ongoing
            ? 'Ongoing (no end date)'
            : activity.endDate
              ? date(activity.endDate)
              : EMPTY}
        </Pair>
        <Pair label="Planned cost">{money(activity.plannedCost)}</Pair>
        <Pair label="Approval workflow">
          {workflow ? `${workflow.name}${workflow.active ? '' : ' (inactive)'}` : EMPTY}
        </Pair>
        <Pair label="Description" wide>
          {activity.description ? (
            <span className={styles.description}>{activity.description}</span>
          ) : (
            EMPTY
          )}
        </Pair>
      </dl>
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
 * the plan's business unit with the reason, "—" when empty.
 */
export function ActivityAttributes({
  activity,
  form,
  businessUnits,
  error,
  onRetry,
}: ActivityAttributesProps) {
  return (
    <Card title="Shell attributes">
      {form && businessUnits ? (
        <ShellAttributeValues
          form={form}
          values={activity.attributes}
          businessUnit={businessUnits}
          planLink={recordHref('plan', activity.plan.id)}
        />
      ) : error ? (
        <ErrorState error={error} onRetry={onRetry} />
      ) : (
        <Skeleton lines={4} />
      )}
    </Card>
  );
}

/**
 * The Goals card (FR-018): INQ, MQL and revenue, MQL and revenue "Calculated" from the plan's
 * model (the formula in an ⓘ once the model is known) or "Typed"; "—" when empty.
 */
export function ActivityGoals({ activity, model }: { activity: Activity; model?: ModelFacts }) {
  const { money } = useFormatters();
  const { goals } = activity;
  const origin = (stage: string) =>
    goals.calculated ? (
      <span className={styles.muted}>
        Calculated
        {model && <InfoTip label={`${stage} calculation`} text={goalsFormula(model)} />}
      </span>
    ) : (
      <span className={styles.muted}>Typed</span>
    );

  return (
    <Card title="Goals">
      <dl className={styles.values}>
        <Pair label="INQ">{goals.inq === null ? EMPTY : formatWhole(goals.inq)}</Pair>
        <Pair label="MQL">
          {goals.mql === null ? (
            EMPTY
          ) : (
            <>
              {formatWhole(goals.mql)} {origin('MQL')}
            </>
          )}
        </Pair>
        <Pair label="Revenue">
          {goals.revenue === null ? (
            EMPTY
          ) : (
            <>
              {money(goals.revenue)} {origin('Revenue')}
            </>
          )}
        </Pair>
      </dl>
    </Card>
  );
}

/**
 * What the Finance module and GSAP will fill in later (FR-020): collapsed, every total "—".
 */
export function FromFinance() {
  return (
    <section aria-label="From finance (coming later)" className={styles.finance}>
      <details>
        <summary className={styles.summary}>From finance (coming later)</summary>
        <dl className={styles.values}>
          <Pair label="Total actuals">{EMPTY}</Pair>
          <Pair label="Total invoiced">{EMPTY}</Pair>
          <Pair label="Total purchase requests">{EMPTY}</Pair>
        </dl>
        <p className={styles.muted}>Filled when the Finance module and GSAP are connected</p>
      </details>
    </section>
  );
}

function Pair({ label, wide, children }: { label: string; wide?: boolean; children: ReactNode }) {
  return (
    <div className={wide ? `${styles.pair} ${styles.wide}` : styles.pair}>
      <dt>{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}
