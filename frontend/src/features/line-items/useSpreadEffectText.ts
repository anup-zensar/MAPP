import { useLineItemSection } from './api';
import { spreadEffect, spreadEffectText } from './spread-effect';

/** The dates typed in the activity form. */
export interface ActivityDates {
  /** "YYYY-MM-DD" or "". */
  startDate: string;
  endDate: string;
  ongoing: boolean;
}

/**
 * The sentence saying what new activity dates do to the activity's line items (FR-022, D-150):
 * "When you save, 2 line items are re-spread over the new dates and 1 custom spread is kept as
 * typed." — for SpreadEffectNote under the dates of feature 006's edit form, and for 006's
 * re-approval dialog. Null while the line items load, when the activity has none, or when the
 * dates are the saved ones.
 */
export function useSpreadEffectText(activityId: string, dates: ActivityDates): string | null {
  const section = useLineItemSection(activityId, activityId !== '');
  if (!section.data || section.data.items.length === 0) {
    return null;
  }
  const saved = section.data.activity;
  const typed = {
    startDate: dates.startDate || null,
    endDate: dates.ongoing ? null : dates.endDate || null,
    ongoing: dates.ongoing,
  };
  if (
    typed.startDate === saved.startDate &&
    typed.endDate === (saved.ongoing ? null : saved.endDate) &&
    typed.ongoing === saved.ongoing
  ) {
    return null;
  }
  return spreadEffectText(
    spreadEffect(
      section.data.items.map((item) => ({
        method: item.spread.method,
        months: item.spread.method === 'custom' ? item.spread.months.map((m) => m.month) : [],
      })),
      typed,
    ),
  );
}
