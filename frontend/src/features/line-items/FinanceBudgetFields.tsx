import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { problemOf } from '@/ui/FormPanel/problem-of';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { useFieldHelp } from '@/features/settings';
import type { LineItemFormState } from './line-item-form-state';
import type { FinanceBudgetItemRef, LineItemActivity } from './api';
import {
  BUDGET_SEARCH,
  inactiveItemText,
  itemClearedText,
  LABEL,
  listedFirstText,
  noBudgetItemsText,
} from './line-item-text';

const SEARCHABLE_ABOVE = 10;

export interface FinanceBudgetFieldsProps {
  state: LineItemFormState;
  activity: LineItemActivity;
  years: number[];
  /** The active items of those years, the plan's default budget first, and a linked inactive one. */
  items: FinanceBudgetItemRef[];
  problems: FieldProblem[];
  onYearChange: (year: number | null) => void;
  onItemChange: (id: string) => void;
}

/** The finance budget year, then that year's items labelled "<item> · <budget> · <code>". */
export function FinanceBudgetFields({
  state,
  activity,
  years,
  items,
  problems,
  onYearChange,
  onItemChange,
}: FinanceBudgetFieldsProps) {
  const year = state.financeBudgetYear;
  const yearOptions = [...new Set([...years, ...(year === null ? [] : [year])])].sort();
  const ofYear = items.filter((item) => item.year === year);
  const chosen = ofYear.find((item) => item.id === state.financeBudgetItemId);
  const defaultBudget = activity.defaultFinanceBudget;
  const listedFirst =
    defaultBudget && ofYear.some((item) => item.budgetName === defaultBudget && item.active);

  const notes = [
    state.sources.financeBudgetItemId,
    state.clearedItem && itemClearedText(state.clearedItem.name, state.clearedItem.year),
    chosen && !chosen.active && inactiveItemText(chosen.name),
  ].filter(Boolean);
  const { help } = useFieldHelp();

  return (
    <>
      <ChoiceField
        label={LABEL.financeBudgetYear}
        tip={help('lineItem.financeBudgetYear')}
        name="financeBudgetYear"
        marking="Needed to submit"
        options={yearOptions.map((option) => ({
          id: String(option),
          label: String(option),
          active: true,
        }))}
        value={year === null ? '' : String(year)}
        onChange={(value) => onYearChange(value === '' ? null : Number(value))}
        error={problemOf(problems, 'financeBudgetYear')}
        note={state.sources.financeBudgetYear}
      />
      <ChoiceField
        label={LABEL.financeBudgetItem}
        tip={help('lineItem.financeBudgetItem')}
        name="financeBudgetItemId"
        marking="Needed to submit"
        hint={
          [
            ofYear.length > SEARCHABLE_ABOVE ? BUDGET_SEARCH : null,
            listedFirst && !state.sources.financeBudgetItemId
              ? listedFirstText(defaultBudget)
              : null,
          ]
            .filter(Boolean)
            .join(' · ') || undefined
        }
        options={ofYear.map((item) => ({
          id: item.id,
          label: [item.name, item.budgetName, item.code].filter(Boolean).join(' · '),
          active: item.active,
        }))}
        value={state.financeBudgetItemId}
        onChange={onItemChange}
        disabledReason={
          year !== null && ofYear.length === 0
            ? [noBudgetItemsText(year), ...notes].join(' ')
            : undefined
        }
        error={problemOf(problems, 'financeBudgetItemId')}
        note={notes.length > 0 ? notes.join(' ') : undefined}
      />
    </>
  );
}
