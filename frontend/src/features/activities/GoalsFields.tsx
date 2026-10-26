import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { DecimalField } from '@/ui/DecimalField/DecimalField';
import { FormSection } from '@/ui/FormSection/FormSection';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { formatAmount, formatWhole, parseWholeInput } from '@/format/decimal';
import { formatMoney } from '@/format/intl';
import { waterfall } from '@/features/settings/models/waterfall';
import type { GoalsState } from './activityFormState';
import { fieldId, usdOf } from './activityFormState';
import { EMPTY, goalsFormula } from './activityText';
import type { ModelFacts } from './activityText';
import styles from './GoalsFields.module.css';

const WHOLE_NUMBER = 'Enter a whole number of 0 or more.';
const TOO_LARGE = 'Too large to calculate — try fewer inquiries.';

export interface GoalsFieldsProps {
  value: GoalsState;
  onChange: (value: GoalsState) => void;
  /** The plan's model: its rates calculate MQL and revenue. */
  model: ModelFacts;
  /** The problems of the last Save by field name ("goals.inq"). */
  errors: { inq?: string; mql?: string; revenue?: string };
}

/**
 * The Goals section of the activity form (FR-018): INQ as typed; MQL and revenue calculated from
 * the plan's model ("Calculated", with the formula in an ⓘ) unless **Set later stages myself** is
 * on, when they are typed too and kept when INQ changes. Turning it off lists the typed values
 * that saving will replace.
 */
export function GoalsFields({ value, onChange, model, errors }: GoalsFieldsProps) {
  const set = (change: Partial<GoalsState>) => onChange({ ...value, ...change });
  const calculated = calculate(value.inq, model);
  const typedNow = value.setLaterStagesMyself;
  const typedKept = !typedNow && (value.mql.trim() !== '' || value.revenue.trim() !== '');

  function turnTyping(on: boolean) {
    // Typing starts from the calculated values when nothing was typed yet.
    if (on && value.mql === '' && value.revenue === '' && calculated.kind === 'ok') {
      set({
        setLaterStagesMyself: true,
        mql: formatWhole(calculated.mql),
        revenue: formatAmount(calculated.revenue),
      });
      return;
    }
    set({ setLaterStagesMyself: on });
  }

  return (
    <FormSection title="Goals">
      {/* The funnel on one row, INQ → MQL → Revenue, so the form stays short (UI audit AF-01). */}
      <div className={styles.funnel}>
        <DecimalField
          label="INQ"
          name={fieldId('goals.inq')}
          decimals={0}
          value={value.inq}
          onChange={(inq) => set({ inq })}
          error={errors.inq ?? (isWholeOrEmpty(value.inq) ? undefined : WHOLE_NUMBER)}
        />
        {typedNow ? (
          <>
            <DecimalField
              label="MQL"
              name={fieldId('goals.mql')}
              decimals={0}
              value={value.mql}
              onChange={(mql) => set({ mql })}
              error={errors.mql ?? (isWholeOrEmpty(value.mql) ? undefined : WHOLE_NUMBER)}
            />
            <MoneyField
              label="Revenue"
              name={fieldId('goals.revenue')}
              currency="USD"
              value={value.revenue}
              onChange={(revenue) => set({ revenue })}
              error={errors.revenue}
            />
          </>
        ) : (
          <dl className={styles.stages}>
            <CalculatedStage stage="MQL" text={stageText(calculated, 'mql')} model={model} />
            <CalculatedStage
              stage="Revenue"
              text={stageText(calculated, 'revenue')}
              model={model}
            />
          </dl>
        )}
      </div>
      <CheckboxField
        label="Set later stages myself"
        name={fieldId('goals.setLaterStagesMyself')}
        checked={typedNow}
        onChange={turnTyping}
      />
      {typedKept && (
        <p role="status" className={styles.note}>
          These typed values will be replaced when you save: MQL {value.mql || EMPTY} →{' '}
          {stageText(calculated, 'mql')}; Revenue {revenueText(value.revenue)} →{' '}
          {stageText(calculated, 'revenue')}.
        </p>
      )}
    </FormSection>
  );
}

function CalculatedStage({
  stage,
  text,
  model,
}: {
  stage: string;
  text: string;
  model: ModelFacts;
}) {
  return (
    <div className={styles.stage}>
      <dt>{stage}</dt>
      <dd>
        <span className={styles.value}>{text}</span>
        <span className={styles.calculated}>
          Calculated
          <InfoTip label={`${stage} calculation`} text={goalsFormula(model)} />
        </span>
      </dd>
    </div>
  );
}

type Calculation =
  { kind: 'ok'; mql: bigint; revenue: string } | { kind: 'empty' } | { kind: 'tooLarge' };

function calculate(inqText: string, model: ModelFacts): Calculation {
  const inq = inqText.trim() === '' ? null : parseWholeInput(inqText);
  if (inq === null || inq < 0n) {
    return { kind: 'empty' };
  }
  const result = waterfall(inq, model);
  return result === 'tooLarge' ? { kind: 'tooLarge' } : { kind: 'ok', ...result };
}

function stageText(calculation: Calculation, stage: 'mql' | 'revenue'): string {
  if (calculation.kind === 'empty') {
    return EMPTY;
  }
  if (calculation.kind === 'tooLarge') {
    return TOO_LARGE;
  }
  return stage === 'mql' ? formatWhole(calculation.mql) : formatMoney(calculation.revenue, 'USD');
}

function revenueText(text: string): string {
  const money = usdOf(text);
  return money ? formatMoney(money.amount, 'USD') : text || EMPTY;
}

function isWholeOrEmpty(text: string): boolean {
  if (text.trim() === '') {
    return true;
  }
  const number = parseWholeInput(text);
  return number !== null && number >= 0n;
}
