import type { ReactNode } from 'react';
import { Link } from 'react-router';
import { FieldHelpTip, useAttributeForm } from '@/features/settings';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { ShellAttributeValues } from '@/ui/ShellAttributeFields/ShellAttributeValues';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TeamPills } from '@/ui/TeamPill/TeamPill';
import { formatAmount, formatPercent } from '@/format/decimal';
import { localDate } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import { AmountPair } from './AmountPair';
import { PLAN_TYPE_LABELS } from './plan-currency';
import styles from './PlanDetails.module.css';
import type { Plan } from './api';
import { SPREAD_LABELS } from './plan-text';

const NOTHING = '—';

interface FieldProps {
  label: string;
  /** The Field help entry explaining the value, shown as an ⓘ after the label. */
  helpKey?: string;
  children: ReactNode;
}

function Field({ label, helpKey, children }: FieldProps) {
  return (
    <div className={styles.field}>
      <dt>
        {label}
        {helpKey && <FieldHelpTip helpKey={helpKey} label={label} />}
      </dt>
      <dd>{children}</dd>
    </div>
  );
}

export interface PlanDetailsProps {
  plan: Plan;
  basicsExtra?: ReactNode;
  budgetExtra?: ReactNode;
  budgetValue?: ReactNode;
}

/** Every field of the plan, read-only: Basics, Budget, the Shell attributes and who changed it. */
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
            <Field label="Plan name" helpKey="plan.name">
              {plan.name}
            </Field>
            <Field label="Year" helpKey="plan.year">
              {plan.year}
            </Field>
            <Field label="Model" helpKey="plan.model">
              {model.name}
              {!model.active && ' (inactive)'}
              <span className={styles.subtle}>
                INQ→MQL {formatPercent(model.inqToMqlRate)} · MQL→deal{' '}
                {formatPercent(model.mqlToDealRate)} · average deal size{' '}
                {formatAmount(model.averageDealSize.amount)} {model.averageDealSize.currency}
              </span>
            </Field>
            <Field label="Teams" helpKey="plan.teams">
              {plan.teams.length > 0 ? <TeamPills teams={plan.teams} linked /> : NOTHING}
            </Field>
            <Field label="Owner" helpKey="plan.owner">
              {plan.owner ? <PersonLabel person={plan.owner} /> : NOTHING}
            </Field>
            {plan.planType && (
              <Field label="Plan type" helpKey="plan.planType">
                {PLAN_TYPE_LABELS[plan.planType]}
              </Field>
            )}
            {basicsExtra}
            <Field label="Strategic / AOP plan" helpKey="plan.aop">
              {plan.aop ? 'Yes' : 'No'}
            </Field>
            <Field label="Description" helpKey="plan.description">
              {plan.description || NOTHING}
            </Field>
          </dl>
        </section>
        <section aria-labelledby="details-budget">
          <h3 id="details-budget" className={styles.groupTitle}>
            Budget
          </h3>
          <dl className={styles.fields}>
            {plan.currency && (
              <Field label="Plan currency" helpKey="plan.currency">
                {plan.currency}
                {plan.currencyLock && <span className={styles.subtle}>{plan.currencyLock}</span>}
              </Field>
            )}
            {budgetExtra}
            <Field label="Plan budget" helpKey="plan.budget">
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
            <Field label="Default finance budget" helpKey="plan.defaultFinanceBudget">
              {plan.defaultFinanceBudget?.name ?? NOTHING}
            </Field>
            <Field label="Default line-item spread" helpKey="plan.defaultSpread">
              {SPREAD_LABELS[plan.defaultSpread]}
            </Field>
            <Field label="BCPIR plan" helpKey="plan.bcpir">
              {plan.bcpir ? 'Yes' : 'No'}
            </Field>
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
