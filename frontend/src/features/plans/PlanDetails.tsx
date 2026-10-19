import type { ReactNode } from 'react';
import { useAttributeForm } from '@/api/settingsQueries';
import { Button, Card, ShellAttributeValues, Skeleton, TeamPills } from '@/components';
import { formatAmount, formatPercent } from '@/lib/decimal';
import { useFormatters } from '@/lib/useFormatters';
import styles from './PlanPage.module.css';
import type { Plan } from './plansApi';
import { SPREAD_LABELS } from './planText';

const NOTHING = '—';

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className={styles.field}>
      <dt>{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}

/**
 * Every field of the plan, read-only (FR-015): Basics, Budget and the Shell attributes shown for
 * its business unit; "—" for empty values and "(inactive)" for values that left Settings.
 */
export function PlanDetails({ plan }: { plan: Plan }) {
  const { money } = useFormatters();
  const attributeForm = useAttributeForm('plan');
  const { model } = plan;

  return (
    <Card title="Details" subtitle={`Amounts in ${plan.figures.currency}`}>
      <div className={styles.groups}>
        <section aria-labelledby="details-basics">
          <h3 id="details-basics" className={styles.groupTitle}>
            Basics
          </h3>
          <dl className={styles.fields}>
            <Field label="Plan name">{plan.name}</Field>
            <Field label="Year">{plan.year}</Field>
            <Field label="Model">
              {model.name}
              {!model.active && ' (inactive)'}
              <span className={styles.subtle}>
                INQ→MQL {formatPercent(model.inqToMqlRate)} · MQL→deal{' '}
                {formatPercent(model.mqlToDealRate)} · average deal size{' '}
                {formatAmount(model.averageDealSize.amount)} {model.averageDealSize.currency}
              </span>
            </Field>
            <Field label="Teams">
              {plan.teams.length > 0 ? <TeamPills teams={plan.teams} linked /> : NOTHING}
            </Field>
            <Field label="Owner">
              {plan.owner ? `${plan.owner.name}${plan.owner.active ? '' : ' (inactive)'}` : NOTHING}
            </Field>
            <Field label="Strategic / AOP plan">{plan.aop ? 'Yes' : 'No'}</Field>
            <Field label="Description">{plan.description || NOTHING}</Field>
          </dl>
        </section>
        <section aria-labelledby="details-budget">
          <h3 id="details-budget" className={styles.groupTitle}>
            Budget
          </h3>
          <dl className={styles.fields}>
            <Field label="Plan budget">{money(plan.figures.budget)}</Field>
            <Field label="Default finance budget">
              {plan.defaultFinanceBudget?.name ?? NOTHING}
            </Field>
            <Field label="Default line-item spread">{SPREAD_LABELS[plan.defaultSpread]}</Field>
            <Field label="BCPIR plan">{plan.bcpir ? 'Yes' : 'No'}</Field>
          </dl>
        </section>
        <section aria-labelledby="details-attributes">
          <h3 id="details-attributes" className={styles.groupTitle}>
            Shell attributes
          </h3>
          {attributeForm.isSuccess ? (
            <ShellAttributeValues
              form={attributeForm.data}
              values={plan.attributes}
              businessUnit={null}
              sectionHeading="none"
            />
          ) : attributeForm.isError ? (
            <p className={styles.subtle}>
              The Shell attributes couldn't be loaded.{' '}
              <Button size="sm" onClick={() => void attributeForm.refetch()}>
                Try again
              </Button>
            </p>
          ) : (
            <Skeleton lines={3} />
          )}
        </section>
      </div>
    </Card>
  );
}
