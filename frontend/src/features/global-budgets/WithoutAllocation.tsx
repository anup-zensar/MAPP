import { Link } from 'react-router';
import { Button } from '@/ui/Button/Button';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { TeamPills } from '@/ui/TeamPill/TeamPill';
import { ScrollBox } from '@/features/financials';
import { PlanStatusChip } from '@/features/plans';
import { useFormatters } from '@/format/useFormatters';
import type { AllocationOption, GlobalBudget } from './api';
import { withoutAllocationHeading } from './global-budgets-text';
import styles from './GlobalBudgetPage.module.css';

/** "Plans of 2026 without an allocation", each with Allocate (FR-032). */
export function WithoutAllocation({
  budget,
  onAllocate,
}: {
  budget: GlobalBudget;
  onAllocate: (option: AllocationOption) => void;
}) {
  const { money } = useFormatters();
  const heading = withoutAllocationHeading(budget.year);
  const plans = budget.plansWithoutAllocation;

  return (
    <section className={styles.panel} aria-labelledby="without-heading">
      <div className={styles.panelHeader}>
        <h2 id="without-heading" className={styles.heading}>
          {heading}
        </h2>
      </div>
      {plans.length === 0 ? (
        <p
          className={styles.emptyLine}
        >{`Every plan of ${budget.year} you can see has an allocation.`}</p>
      ) : (
        <ScrollBox label={heading} className={styles.scroll}>
          <table className={styles.table}>
            <caption className="visually-hidden">{heading}</caption>
            <thead>
              <tr>
                <th scope="col">Plan</th>
                <th scope="col">Teams</th>
                <th scope="col">Owner</th>
                <th scope="col">Status</th>
                <th scope="col" className={styles.numeric}>
                  Budget now
                </th>
                <th scope="col">
                  <span className="visually-hidden">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {plans.map((option) => (
                <tr key={option.plan.id}>
                  <th scope="row">
                    <Link
                      to={`/plans/${encodeURIComponent(option.plan.id)}`}
                      className={styles.name}
                    >
                      {option.plan.name}
                    </Link>
                  </th>
                  <td className={styles.teams}>
                    <TeamPills teams={option.teams} />
                  </td>
                  <td className={styles.nowrap}>
                    {option.owner ? <PersonLabel person={option.owner} /> : '—'}
                  </td>
                  <td>
                    <PlanStatusChip status={option.plan.status} />
                  </td>
                  <td className={styles.numeric}>{money(option.budget)}</td>
                  <td className={styles.actions}>
                    <Button
                      size="sm"
                      onClick={() => onAllocate(option)}
                      aria-label={`Allocate to ${option.plan.name}`}
                    >
                      Allocate
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </ScrollBox>
      )}
    </section>
  );
}
