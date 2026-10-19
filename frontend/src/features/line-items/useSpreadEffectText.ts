/** The dates typed in the activity form. */
export interface ActivityDates {
  /** "YYYY-MM-DD" or "". */
  startDate: string;
  endDate: string;
  ongoing: boolean;
}

/**
 * The sentence saying what new activity dates do to the activity's line items (007 plan, 007-P11:
 * "When you save, 3 line items are re-spread over the new dates."), for feature 006's re-approval
 * dialog — the same text SpreadEffectNote shows under the dates. STUB of the shared foundation:
 * null (nothing to say) until feature 007 replaces this file.
 */
export function useSpreadEffectText(activityId: string, dates: ActivityDates): string | null {
  void activityId;
  void dates;
  return null;
}
