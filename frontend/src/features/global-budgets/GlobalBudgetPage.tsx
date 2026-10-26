import { Landmark, Lock, Pencil, Plus } from 'lucide-react';
import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button, LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { ProgressBar } from '@/ui/ProgressBar/ProgressBar';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TeamPills } from '@/ui/TeamPill/TeamPill';
import { useToast } from '@/ui/Toast/useToast';
import { ScrollBox } from '@/features/financials/ScrollBox';
import { PlanStatusChip } from '@/features/plans/PlanFacts';
import { usePageTitle } from '@/ui/usePageTitle';
import { useFormatters } from '@/format/useFormatters';
import { useUndoableDelete } from '@/ui/useUndoableDelete';
import { AllocationDialog } from './AllocationDialog';
import type { AllocationTarget } from './AllocationDialog';
import { BudgetSummaryCards } from './BudgetFigures';
import { GlobalBudgetForm } from './GlobalBudgetForm';
import {
  GLOBAL_BUDGETS_KEY,
  useDeleteGlobalBudget,
  useGlobalBudget,
  useRemoveAllocation,
} from './globalBudgetsApi';
import type { AllocationOption, BudgetAllocationRow, GlobalBudget } from './globalBudgetsApi';
import {
  budgetUsedText,
  inUseText,
  otherMarketsLabel,
  removeEffect,
  removeQuestion,
  toasts,
  withoutAllocationHeading,
} from './globalBudgetsText';
import { BUDGET_GONE } from './saveFailure';
import styles from './GlobalBudgetPage.module.css';

type Dialog =
  | { kind: 'edit' }
  | { kind: 'allocate'; target: AllocationTarget }
  | { kind: 'remove'; row: BudgetAllocationRow }
  | { kind: 'delete' }
  | { kind: 'cantDelete'; reason: string };

/**
 * A global budget's page (014 US6, FR-032 – FR-035), at /budgets/<id>: its four figures (red
 * "Over-allocated by X", never blocking), one row per allocated plan with teams, owner, status,
 * allocated, planned cost, left to plan and budget used, the "Other markets" row for plans the
 * person can't see, and the plans of its year without an allocation. Allocate, change and remove
 * allocations; edit and delete the global budget.
 */
export function GlobalBudgetPage() {
  const { globalBudgetId = '' } = useParams();
  const query = useGlobalBudget(globalBudgetId);
  const budget = query.data;
  usePageTitle(budget?.name ?? 'Global budget');
  const [dialog, setDialog] = useState<Dialog | null>(null);
  const failure = query.error instanceof ApiError ? query.error : null;

  if (!budget) {
    return (
      <>
        <PageHeader
          breadcrumb={[{ label: 'Global budgets', to: '/budgets' }, { label: 'Global budget' }]}
          title="Global budget"
        />
        <Card>
          {failure?.kind === 'notFound' ? (
            <EmptyState
              icon={<Landmark />}
              title={BUDGET_GONE}
              action={<LinkButton to="/budgets">Go to Global budgets</LinkButton>}
            />
          ) : failure?.kind === 'forbidden' ? (
            <EmptyState icon={<Lock />} title={failure.message} />
          ) : query.isError ? (
            <ErrorState error={query.error} onRetry={() => void query.refetch()} />
          ) : (
            <Skeleton lines={6} height="20px" />
          )}
        </Card>
      </>
    );
  }

  const tryDelete = () =>
    setDialog(
      budget.access.delete.available
        ? { kind: 'delete' }
        : {
            kind: 'cantDelete',
            reason:
              budget.access.delete.reason || inUseText(budget.name, budget.allocations.length),
          },
    );

  return (
    <>
      <PageHeader
        breadcrumb={[{ label: 'Global budgets', to: '/budgets' }, { label: budget.name }]}
        title={budget.name}
        meta={<Facts budget={budget} />}
        description={budget.description || undefined}
        moreActions={[{ label: 'Delete', tone: 'danger', onSelect: tryDelete }]}
        actions={
          <Button variant="primary" icon={<Pencil />} onClick={() => setDialog({ kind: 'edit' })}>
            Edit
          </Button>
        }
      />
      <div className={styles.sections}>
        <BudgetSummaryCards figures={budget.figures} />
        <Allocations
          budget={budget}
          onAllocate={() => setDialog({ kind: 'allocate', target: { kind: 'new' } })}
          onChange={(row) => setDialog({ kind: 'allocate', target: { kind: 'change', row } })}
          onRemove={(row) => setDialog({ kind: 'remove', row })}
        />
        <WithoutAllocation
          budget={budget}
          onAllocate={(option) => setDialog({ kind: 'allocate', target: { kind: 'new', option } })}
        />
      </div>

      {dialog?.kind === 'edit' && (
        <GlobalBudgetForm budget={budget} onClose={() => setDialog(null)} />
      )}
      {dialog?.kind === 'allocate' && (
        <AllocationDialog budget={budget} target={dialog.target} onClose={() => setDialog(null)} />
      )}
      {dialog?.kind === 'remove' && (
        <RemoveDialog row={dialog.row} onClose={() => setDialog(null)} />
      )}
      {dialog?.kind === 'delete' && (
        <DeleteDialog
          budget={budget}
          onInUse={(reason) => setDialog({ kind: 'cantDelete', reason })}
          onClose={() => setDialog(null)}
        />
      )}
      {dialog?.kind === 'cantDelete' && (
        <ConfirmDialog
          open
          onOpenChange={(open) => !open && setDialog(null)}
          title={`Can't delete '${budget.name}'`}
          confirmLabel="Close"
          onConfirm={() => setDialog(null)}
        >
          {dialog.reason}
        </ConfirmDialog>
      )}
    </>
  );
}

/** The meta row: year, ID and owner. */
function Facts({ budget }: { budget: GlobalBudget }) {
  return (
    <>
      <span>{budget.year}</span>
      <span>{budget.id}</span>
      <span>{budget.owner ? `Owner: ${budget.owner.name}` : 'No owner'}</span>
    </>
  );
}

interface AllocationsProps {
  budget: GlobalBudget;
  onAllocate: () => void;
  onChange: (row: BudgetAllocationRow) => void;
  onRemove: (row: BudgetAllocationRow) => void;
}

/** One row per allocated plan the person can see, then "Other markets (n plans)" (FR-032, FR-033). */
function Allocations({ budget, onAllocate, onChange, onRemove }: AllocationsProps) {
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

/** "Plans of 2026 without an allocation", each with Allocate (FR-032). */
function WithoutAllocation({
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

/** "Remove the allocation to 'FY2026 Mobility China'?" — the plan keeps its budget (FR-034). */
function RemoveDialog({ row, onClose }: { row: BudgetAllocationRow; onClose: () => void }) {
  const { show } = useToast();
  const { money } = useFormatters();
  const remove = useRemoveAllocation();

  async function confirm() {
    if (remove.isPending) {
      return;
    }
    try {
      await remove.mutateAsync(row.id);
      show({ message: toasts.removed(row.plan.name) });
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      if (apiError.kind !== 'signInRequired' && apiError.kind !== 'sessionExpired') {
        show({ message: apiError.message });
      }
    } finally {
      onClose();
    }
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={removeQuestion(row.plan.name)}
      confirmLabel="Remove allocation"
      tone="danger"
      busy={remove.isPending}
      onConfirm={() => void confirm()}
    >
      {removeEffect(row.plan.name, money(row.amount))}
    </ConfirmDialog>
  );
}

/** "Delete global budget '<name>'?" — then back to the list, with Undo (FR-035). */
function DeleteDialog({
  budget,
  onInUse,
  onClose,
}: {
  budget: GlobalBudget;
  onInUse: (reason: string) => void;
  onClose: () => void;
}) {
  const navigate = useNavigate();
  const { show } = useToast();
  const remove = useDeleteGlobalBudget();
  const afterDelete = useUndoableDelete({ invalidate: [GLOBAL_BUDGETS_KEY] });

  async function confirm() {
    if (remove.isPending) {
      return;
    }
    try {
      afterDelete(await remove.mutateAsync(budget.id));
      onClose();
      void navigate('/budgets');
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      if (apiError.kind === 'inUse') {
        onInUse(apiError.message);
        return;
      }
      if (apiError.kind !== 'signInRequired' && apiError.kind !== 'sessionExpired') {
        show({ message: apiError.message });
      }
      onClose();
    }
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={`Delete global budget '${budget.name}'?`}
      confirmLabel="Delete global budget"
      tone="danger"
      busy={remove.isPending}
      onConfirm={() => void confirm()}
    >
      It moves to the recycle bin; you can restore it.
    </ConfirmDialog>
  );
}
