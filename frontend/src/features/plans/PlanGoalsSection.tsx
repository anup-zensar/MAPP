import type { Control } from 'react-hook-form';
import { FormField } from '@/ui/FormField/FormField';
import type { WaterfallModel } from '@/features/settings';
import { GoalFields } from './GoalFields';
import type { DefaultHint } from './plan-form-sections';
import { PlanFormCard } from './PlanFormCard';
import type { PlanFormValues } from './schemas';

export interface PlanGoalsSectionProps {
  control: Control<PlanFormValues>;
  model: WaterfallModel | null;
  hint: DefaultHint;
}

/** Goals: the stage and the goal amount, with the other stages worked out from the model. */
export function PlanGoalsSection({ control, model, hint }: PlanGoalsSectionProps) {
  return (
    <PlanFormCard id="plan-goals" title="Goals">
      <FormField
        control={control}
        name="goalStage"
        render={(stage) => (
          <FormField
            control={control}
            name="goalAmount"
            render={(amount) => (
              <GoalFields
                stage={stage.value}
                amount={amount.value}
                onStageChange={stage.onChange}
                onAmountChange={amount.onChange}
                model={model}
                stageHint={hint('goalStage', 'Default: INQ')}
                stageError={stage.error}
                amountError={amount.error}
              />
            )}
          />
        )}
      />
    </PlanFormCard>
  );
}
