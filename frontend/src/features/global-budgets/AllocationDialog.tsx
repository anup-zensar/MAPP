import { CircleAlert } from 'lucide-react';
import { useState } from 'react';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { problemOf } from '@/ui/FormPanel/problem-of';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { formatAmount, parseMoneyInput } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import { useAllocate, useAllocationOptions, useUpdateAllocation } from './api';
import type { AllocationOption, BudgetAllocationRow, GlobalBudget } from './api';
import { toasts } from './global-budgets-text';
import { saveFailureOf } from './save-failure';
import styles from './AllocationDialog.module.css';

const AMOUNT_PROBLEM = 'Enter an amount of 0 or more.';
const PLAN_PROBLEM = 'Choose a plan.';

export type AllocationTarget =
  { kind: 'new'; option?: AllocationOption } | { kind: 'change'; row: BudgetAllocationRow };

export interface AllocationDialogProps {
  budget: GlobalBudget;
  target: AllocationTarget;
  onClose: () => void;
}

/**
 * Allocate and Change allocation (014 FR-030): a plan of the budget's year and an amount in the
 * plan's currency. Saving sets the plan's budget to the amount in the same save; the plan's
 * version guards against a plan changed meanwhile.
 */
export function AllocationDialog({ budget, target, onClose }: AllocationDialogProps) {
  const { show } = useToast();
  const { money, dateFormat } = useFormatters();
  const allocate = useAllocate();
  const change = useUpdateAllocation();
  const choosing = target.kind === 'new' && !target.option;
  const options = useAllocationOptions(budget.id, choosing);
  const [planId, setPlanId] = useState(target.kind === 'new' ? (target.option?.plan.id ?? '') : '');
  const [amount, setAmount] = useState(
    target.kind === 'change' ? formatAmount(target.row.amount.amount) : '',
  );
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<string | null>(null);
  const busy = allocate.isPending || change.isPending;

  const option =
    target.kind === 'new'
      ? (target.option ?? options.data?.items.find((item) => item.plan.id === planId))
      : undefined;
  const planName = target.kind === 'change' ? target.row.plan.name : option?.plan.name;
  const currency =
    target.kind === 'change' ? target.row.amount.currency : (option?.budget?.currency ?? 'USD');

  async function submit() {
    if (busy) {
      return;
    }
    const found: FieldProblem[] = [];
    const parsed = parseMoneyInput(amount);
    if (target.kind === 'new' && !option) {
      found.push({ field: 'planId', message: PLAN_PROBLEM });
    }
    if (!parsed || parsed.amount.startsWith('-')) {
      found.push({ field: 'amount', message: AMOUNT_PROBLEM });
    }
    setProblems(found);
    setFailure(null);
    if (found.length > 0 || !parsed) {
      return;
    }
    const allocated = { amount: parsed.amount, currency };
    try {
      if (target.kind === 'change') {
        await change.mutateAsync({
          allocationId: target.row.id,
          amount: allocated,
          version: target.row.version,
          planVersion: target.row.plan.version,
        });
      } else if (option) {
        await allocate.mutateAsync({
          globalBudgetId: budget.id,
          planId: option.plan.id,
          amount: allocated,
          planVersion: option.plan.version,
        });
      }
      show({ message: toasts.allocated(planName ?? '') });
      onClose();
    } catch (error) {
      const view = saveFailureOf(error, 'plan', dateFormat);
      if (view.sessionLost) {
        onClose();
        return;
      }
      setProblems(view.problems);
      setFailure(
        view.failure
          ? [view.failure.message, view.failure.nextStep].filter(Boolean).join(' ')
          : null,
      );
    }
  }

  return (
    <FormDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={
        target.kind === 'change'
          ? `Change the allocation to '${target.row.plan.name}'`
          : planName && target.option
            ? `Allocate to '${planName}'`
            : `Allocate from '${budget.name}'`
      }
      description="The plan's budget becomes this amount when you save. Its owner sees it but can't change it."
      submitLabel={target.kind === 'change' ? 'Save allocation' : 'Allocate'}
      busy={busy}
      onSubmit={() => void submit()}
    >
      <div className={styles.body}>
        {failure && (
          <p className={styles.failure} role="alert">
            <CircleAlert aria-hidden="true" />
            {failure}
          </p>
        )}
        {choosing &&
          (options.data ? (
            <ChoiceField
              label="Plan"
              name="planId"
              marking="Required"
              hint={`Plans of ${budget.year} without an allocation.`}
              options={options.data.items.map((item) => ({
                id: item.plan.id,
                label: item.plan.name,
                active: true,
              }))}
              value={planId}
              onChange={setPlanId}
              error={problemOf(problems, 'planId')}
            />
          ) : options.isError ? (
            <p className={styles.failure} role="alert">
              <CircleAlert aria-hidden="true" />
              The plans couldn&apos;t be loaded. Close this and try again.
            </p>
          ) : (
            <Skeleton lines={1} height="40px" />
          ))}
        {!choosing && planName && (
          <p className={styles.plan}>
            <span className={styles.planLabel}>Plan</span>
            {planName}
          </p>
        )}
        {option && (
          <p className={styles.note}>
            {option.budget ? `Its budget now: ${money(option.budget)}` : 'It has no budget yet.'}
          </p>
        )}
        <MoneyField
          label="Amount"
          name="amount"
          marking="Required"
          currency={currency}
          value={amount}
          onChange={setAmount}
          error={problemOf(problems, 'amount')}
        />
      </div>
    </FormDialog>
  );
}
