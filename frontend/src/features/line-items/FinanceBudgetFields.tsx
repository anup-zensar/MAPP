import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { problemOf } from '@/ui/FormPanel/problemOf';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { LineItemFormState } from './lineItemFormState';
import type { FinanceBudgetItemRef, LineItemActivity } from './lineItemsApi';
import {
  BUDGET_SEARCH,
  inactiveItemText,
  itemClearedText,
  LABEL,
  listedFirstText,
  noBudgetItemsText,
} from './lineItemText';

/** ChoiceField turns into a searchable list above this many options (design system §5). */
const SEARCHABLE_ABOVE = 10;

export interface FinanceBudgetFieldsProps {
  state: LineItemFormState;
  activity: LineItemActivity;
  /** The plan's year and every year with active items (FR-015). */
  years: number[];
  /**
   * The active items of those years, the plan's default budget first, and on an edit form the
   * linked item when it is inactive (FR-016).
   */
  items: FinanceBudgetItemRef[];
  problems: FieldProblem[];
  onYearChange: (year: number | null) => void;
  onItemChange: (id: string) => void;
}

/**
 * Link to finance budget (FR-015, FR-016): the year, then that year's items grouped by budget —
 * each labelled "<item> · <budget> · <code>", so they can be searched on any of them — with the
 * plan's default budget first; an inactive linked item kept and named as a problem; the item a
 * year change will clear.
 */
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

  return (
    <>
      <ChoiceField
        label={LABEL.financeBudgetYear}
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
          // The lock reason takes the note's place, so it carries the notes too.
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
