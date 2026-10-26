import type { ReactNode } from 'react';
import { Link } from 'react-router';
import { useAttributeForm } from '@/api/settingsQueries';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { ShellAttributeValues } from '@/ui/ShellAttributeFields/ShellAttributeValues';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TeamPills } from '@/ui/TeamPill/TeamPill';
import { formatAmount, formatPercent } from '@/lib/decimal';
import { localDate } from '@/lib/format';
import { useFormatters } from '@/lib/useFormatters';
import { AmountPair } from './AmountPair';
import { PLAN_TYPE_LABELS } from './planCurrency';
import styles from './PlanDetails.module.css';
import type { Plan } from './plansApi';
import { SPREAD_LABELS } from './planText';

const NOTHING = '—';

/**
 * One label and value of the details (design system §5.4 DescriptionList): a 200 px label column
 * from 600 px, the label above its value below, so every value of the page starts at one x.
 */
export function Field({ label, children }: { label: ReactNode; children: ReactNode }) {
  return (
    <div className={styles.field}>
      <dt>{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}

export interface PlanDetailsProps {
  plan: Plan;
  /** Rows added to Basics by later features (016: plan type). */
  basicsExtra?: ReactNode;
  /** Rows added to Budget by later features (016: plan currency; 014: allocation). */
  budgetExtra?: ReactNode;
  /** The plan budget as shown (016: in the plan currency with the display currency beside). */
  budgetValue?: ReactNode;
}

/**
 * Every field of the plan, read-only (FR-015): Basics, Budget and the Shell attributes shown for
 * its business unit; "—" for empty values and "(inactive)" for values that left Settings; then
 * who created and last changed the plan (design system §6.3).
 */
export function PlanDetails({ plan, basicsExtra, budgetExtra, budgetValue }: PlanDetailsProps) {
  const { date } = useFormatters();
  const attributeForm = useAttributeForm('plan');
  const { model } = plan;

  return (
    <Card title="Details" subtitle={`Amounts in ${plan.currency ?? plan.figures.currency}`}>
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
              {plan.owner ? <PersonLabel person={plan.owner} /> : NOTHING}
            </Field>
            {plan.planType && <Field label="Plan type">{PLAN_TYPE_LABELS[plan.planType]}</Field>}
            {basicsExtra}
            <Field label="Strategic / AOP plan">{plan.aop ? 'Yes' : 'No'}</Field>
            <Field label="Description">{plan.description || NOTHING}</Field>
          </dl>
        </section>
        <section aria-labelledby="details-budget">
          <h3 id="details-budget" className={styles.groupTitle}>
            Budget
          </h3>
          <dl className={styles.fields}>
            {plan.currency && (
              <Field label="Plan currency">
                {plan.currency}
                {plan.currencyLock && <span className={styles.subtle}>{plan.currencyLock}</span>}
              </Field>
            )}
            {budgetExtra}
            <Field label="Plan budget">
              {budgetValue ?? (
                <AmountPair
                  primary={plan.budget ?? plan.figures.budget}
                  secondary={plan.figures.budget}
                  inline
                />
              )}
            </Field>
            {plan.allocation && (
              <Field label="Allocated from">
                {plan.allocation.canOpen ? (
                  <Link
                    to={`/global-budgets/${encodeURIComponent(plan.allocation.globalBudget.id)}`}
                  >
                    {plan.allocation.globalBudget.name}
                  </Link>
                ) : (
                  plan.allocation.globalBudget.name
                )}
                {plan.allocation.owner && (
                  <span className={styles.subtle}>Owner: {plan.allocation.owner.name}</span>
                )}
              </Field>
            )}
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
        <p className={styles.changed}>
          Created by {plan.createdBy.name} on {date(localDate(plan.createdAt))}, last changed by{' '}
          {plan.updatedBy.name} on {date(localDate(plan.updatedAt))}
        </p>
      </div>
    </Card>
  );
}
