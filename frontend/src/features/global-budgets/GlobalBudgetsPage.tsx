import { ChevronRight, Landmark, Lock, Plus } from 'lucide-react';
import { useId, useState } from 'react';
import { Link, Navigate, useSearchParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { Button, EmptyState, ErrorState, PageHeader, Skeleton } from '@/components';
import { plural } from '@/lib/text';
import { usePageTitle } from '@/lib/usePageTitle';
import { BudgetFigures } from './BudgetFigures';
import { GlobalBudgetForm } from './GlobalBudgetForm';
import { useGlobalBudgets } from './globalBudgetsApi';
import type { GlobalBudgetSummary } from './globalBudgetsApi';
import { emptyText, PAGE_DESCRIPTION } from './globalBudgetsText';
import styles from './GlobalBudgetsPage.module.css';

/** "?year=all" lists every year; no year means the current one. */
function readYear(params: URLSearchParams, currentYear: number): number | null {
  const value = params.get('year');
  if (value === 'all') {
    return null;
  }
  const year = Number(value);
  return value && Number.isInteger(year) && year >= 2000 && year <= 2100 ? year : currentYear;
}

/**
 * Global budgets (014 US6, FR-036), at /budgets: the global budgets of a year (the current year
 * unless another is chosen), each a card with its owner, its allocations and its four figures,
 * linking to its page; New global budget opens the form.
 */
export function GlobalBudgetsPage() {
  usePageTitle('Global budgets');
  const [params, setParams] = useSearchParams();
  const currentYear = new Date().getFullYear();
  const year = readYear(params, currentYear);
  const budgets = useGlobalBudgets(year);
  const [creating, setCreating] = useState(false);
  // The new global budget opens once its form has closed (its unsaved-changes guard is gone).
  const [createdId, setCreatedId] = useState<string | null>(null);
  const yearId = useId();
  const forbidden = budgets.error instanceof ApiError && budgets.error.kind === 'forbidden';
  const years = withChosen(budgets.data?.years ?? [], [currentYear, year]);

  return (
    <>
      <PageHeader
        title="Global budgets"
        description={PAGE_DESCRIPTION}
        actions={
          <Button variant="primary" icon={<Plus />} onClick={() => setCreating(true)}>
            New global budget
          </Button>
        }
      />
      <section className={styles.panel} aria-label="Global budgets">
        <div className={styles.toolbar}>
          <label htmlFor={yearId} className={styles.field}>
            Year
          </label>
          <select
            id={yearId}
            className={styles.select}
            value={year === null ? 'all' : String(year)}
            onChange={(event) =>
              setParams(
                event.target.value === String(currentYear) ? {} : { year: event.target.value },
              )
            }
          >
            <option value="all">All years</option>
            {years.map((choice) => (
              <option key={choice} value={choice}>
                {choice}
              </option>
            ))}
          </select>
          {budgets.data && (
            <p className={styles.count} aria-live="polite">
              {plural(budgets.data.total, 'global budget')}
            </p>
          )}
        </div>
        <div className={styles.body}>
          {budgets.data ? (
            budgets.data.items.length === 0 ? (
              <EmptyState
                icon={<Landmark />}
                title={emptyText(year)}
                text="A global budget shares money for a year out to market plans."
              />
            ) : (
              <ul className={styles.list}>
                {budgets.data.items.map((budget) => (
                  <BudgetCard key={budget.id} budget={budget} />
                ))}
              </ul>
            )
          ) : forbidden ? (
            <EmptyState icon={<Lock />} title={budgets.error?.message ?? ''} />
          ) : budgets.isError ? (
            <ErrorState error={budgets.error} onRetry={() => void budgets.refetch()} />
          ) : (
            <Skeleton lines={4} height="20px" />
          )}
        </div>
      </section>
      {creating && (
        <GlobalBudgetForm
          onClose={() => setCreating(false)}
          onCreated={(created) => setCreatedId(created.id)}
        />
      )}
      {createdId && !creating && <Navigate to={`/budgets/${encodeURIComponent(createdId)}`} />}
    </>
  );
}

/** One global budget: its name (the link to its page), year, owner, allocations and figures. */
function BudgetCard({ budget }: { budget: GlobalBudgetSummary }) {
  const facts = [
    String(budget.year),
    budget.owner ? `Owner: ${budget.owner.name}` : 'No owner',
    budget.allocationCount === 0
      ? 'Not allocated yet'
      : `Allocated to ${plural(budget.allocationCount, 'plan')}`,
  ];

  return (
    <li className={styles.card}>
      <div className={styles.cardHeader}>
        <h2 className={styles.name}>
          <Link to={`/budgets/${encodeURIComponent(budget.id)}`}>{budget.name}</Link>
        </h2>
        <ChevronRight aria-hidden="true" className={styles.chevron} />
      </div>
      <p className={styles.facts}>{facts.join(' · ')}</p>
      <BudgetFigures figures={budget.figures} />
    </li>
  );
}

/** The years to offer: the API's, the current year and the chosen one, newest first. */
function withChosen(years: number[], more: (number | null)[]): number[] {
  const all = new Set(years);
  for (const year of more) {
    if (year !== null) {
      all.add(year);
    }
  }
  return [...all].sort((a, b) => b - a);
}
