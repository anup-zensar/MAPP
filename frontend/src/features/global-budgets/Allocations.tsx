import { Landmark, Plus } from 'lucide-react';
import { Link } from 'react-router';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { ProgressBar } from '@/ui/ProgressBar/ProgressBar';
import { TeamPills } from '@/ui/TeamPill/TeamPill';
import { ScrollBox } from '@/features/financials';
import { PlanStatusChip } from '@/features/plans';
import { useFormatters } from '@/format/useFormatters';
import type { BudgetAllocationRow, GlobalBudget } from './api';
import { budgetUsedText, otherMarketsLabel } from './global-budgets-text';
import styles from './GlobalBudgetPage.module.css';

interface AllocationsProps {
  budget: GlobalBudget;
  onAllocate: () => void;
  onChange: (row: BudgetAllocationRow) => void;
  onRemove: (row: BudgetAllocationRow) => void;
}

/** One row per allocated plan the person can see, then "Other markets (n plans)" (FR-032, FR-033). */
export function Allocations({ budget, onAllocate, onChange, onRemove }: AllocationsProps) {
  const { money } = useFormatters();
  const rows = budget.allocations;
  const other = budget.otherMarkets;

  return (
    <section className={styles.panel} aria-labelledby="allocations-heading">
      <div className={styles.panelHeader}>
        <h2 id="allocations-heading" className={styles.heading}>
          Allocations
        </h2>
        <Button size="sm" icon={<Plus />} onClick={onAllocate}>
          Allocate to a plan
        </Button>
      </div>
      {rows.length === 0 && !other ? (
        <div className={styles.empty}>
          <EmptyState
            icon={<Landmark />}
            title="No plan has an allocation yet"
            text="Allocate part of the global budget to a plan: it becomes that plan's budget."
          />
        </div>
      ) : (
        <ScrollBox label="Allocations" className={styles.scroll}>
          <table className={styles.table}>
            <caption className="visually-hidden">Allocations</caption>
            <thead>
              <tr>
                <th scope="col">Plan</th>
                <th scope="col">Teams</th>
                <th scope="col">Owner</th>
                <th scope="col">Status</th>
                <th scope="col" className={styles.numeric}>
                  Allocated
                </th>
                <th scope="col" className={styles.numeric}>
                  Planned cost
                </th>
                <th scope="col" className={styles.numeric}>
                  Left to plan
                </th>
                <th scope="col" className={styles.numeric}>
                  Budget used
                </th>
                <th scope="col">
                  <span className="visually-hidden">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.id}>
                  <th scope="row">
                    <Link to={`/plans/${encodeURIComponent(row.plan.id)}`} className={styles.name}>
                      {row.plan.name}
                    </Link>
                  </th>
                  <td className={styles.teams}>
                    <TeamPills teams={row.teams} />
                  </td>
                  <td className={styles.nowrap}>
                    {row.owner ? <PersonLabel person={row.owner} /> : '—'}
                  </td>
                  <td>
                    <PlanStatusChip status={row.plan.status} />
                  </td>
                  <td className={styles.numeric}>{money(row.amount)}</td>
                  <td className={styles.numeric}>{money(row.plannedCost)}</td>
                  <td
                    className={`${styles.numeric} ${row.leftToPlan.amount.startsWith('-') ? styles.problem : ''}`}
                  >
                    {money(row.leftToPlan)}
                  </td>
                  <td className={styles.numeric}>
                    <BudgetUsed percent={row.budgetUsedPercent} />
                  </td>
                  <td className={styles.actions}>
                    <ActionsMenu
                      label={row.plan.name}
                      items={[
                        { label: 'Change allocation', onSelect: () => onChange(row) },
                        {
                          label: 'Remove allocation',
                          tone: 'danger',
                          separatorBefore: true,
                          onSelect: () => onRemove(row),
                        },
                      ]}
                    />
                  </td>
                </tr>
              ))}
              {other && (
                <tr className={styles.other}>
                  <th scope="row">
                    {otherMarketsLabel(other.planCount)}
                    <span className={styles.secondary}>
                      Plans of markets you don&apos;t see, added up
                    </span>
                  </th>
                  <td />
                  <td />
                  <td />
                  <td className={styles.numeric}>{money(other.amount)}</td>
                  <td className={styles.numeric}>{money(other.plannedCost)}</td>
                  <td />
                  <td />
                  <td />
                </tr>
              )}
            </tbody>
          </table>
        </ScrollBox>
      )}
    </section>
  );
}

/** "62.50 %" with its bar; red above 100 %. */
function BudgetUsed({ percent }: { percent: string | null }) {
  const value = percent === null ? null : Number(percent);
  return (
    <div className={styles.used}>
      <span className={styles.usedText}>{budgetUsedText(percent)}</span>
      {value !== null && (
        <ProgressBar value={value} label="Budget used" tone={value > 100 ? 'problem' : undefined} />
      )}
    </div>
  );
}
