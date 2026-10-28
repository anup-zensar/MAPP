import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { DecimalField } from '@/ui/DecimalField/DecimalField';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { useFieldHelp } from '@/features/settings';
import type { WaterfallModel } from '@/features/settings';
import styles from './PlanForm.module.css';
import { goalRows } from './goal-waterfall';
import type { WaterfallStage } from './goal-waterfall';
import { GOAL_STAGES, goalAmountLabel, STAGE_OPTIONS } from './plan-text';
import type { GoalStage } from './plan-text';

const ROW_LABELS: Record<WaterfallStage, string> = {
  inq: 'INQ',
  mql: 'MQL',
  deals: 'Deals',
  revenue: 'Revenue',
};

export interface GoalFieldsProps {
  stage: GoalStage;
  amount: string;
  onStageChange: (stage: GoalStage) => void;
  onAmountChange: (amount: string) => void;
  /** The chosen model's rates and deal size; null without a model. */
  model: WaterfallModel | null;
  stageError?: string;
  amountError?: string;
  /** "Default: INQ" while the stage is the default one. */
  stageHint?: string;
}

/** The goal stage and amount, and the other stages worked out live from the model. */
export function GoalFields({
  stage,
  amount,
  onStageChange,
  onAmountChange,
  model,
  stageError,
  amountError,
  stageHint,
}: GoalFieldsProps) {
  const rows = goalRows(stage, amount, model).filter((row) => row.calculated);
  const { help } = useFieldHelp();

  return (
    <>
      <ChoiceField
        label="Goal stage"
        name="goalStage"
        marking="Needed to publish"
        tip={help('plan.goalStage')}
        hint={stageHint}
        options={GOAL_STAGES.map((id) => ({
          id,
          label: STAGE_OPTIONS[id],
          active: true,
        }))}
        value={stage}
        onChange={(value) => onStageChange(GOAL_STAGES.find((known) => known === value) ?? stage)}
        error={stageError}
      />
      {stage === 'revenue' ? (
        <MoneyField
          label={goalAmountLabel(stage)}
          name="goalAmount"
          tip={help('plan.goalAmount')}
          currency="USD"
          value={amount}
          onChange={onAmountChange}
          error={amountError}
        />
      ) : (
        <DecimalField
          label={goalAmountLabel(stage)}
          name="goalAmount"
          tip={help('plan.goalAmount')}
          decimals={0}
          value={amount}
          onChange={onAmountChange}
          error={amountError}
        />
      )}
      <dl className={styles.goals} aria-label="Calculated goals">
        {rows.map((row) => (
          <div key={row.stage} className={styles.goalRow}>
            <dt>{ROW_LABELS[row.stage]}</dt>
            <dd>
              <span className={styles.goalValue}>{row.value ?? '—'}</span>
              {row.reason ? (
                <span className={styles.goalNote}>{row.reason}</span>
              ) : (
                <span className={styles.goalNote}>
                  <span className={styles.calculated}>Calculated</span> {row.formula}
                </span>
              )}
            </dd>
          </div>
        ))}
      </dl>
    </>
  );
}
