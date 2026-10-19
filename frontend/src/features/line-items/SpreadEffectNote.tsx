export interface SpreadEffectNoteProps {
  activityId: string;
  /** The dates typed in the activity form ("YYYY-MM-DD" or ""). */
  startDate: string;
  endDate: string;
  ongoing: boolean;
}

/**
 * What new activity dates do to its line items' spreads, under the dates of feature 006's activity
 * edit form (007 plan, 007-P11): "When you save, 3 line items are re-spread over the new dates." —
 * STUB of the shared foundation: nothing yet. Feature 007 replaces this file; 006's
 * ActivityFormPage already places it.
 */
export function SpreadEffectNote({
  activityId,
  startDate,
  endDate,
  ongoing,
}: SpreadEffectNoteProps) {
  void activityId;
  void startDate;
  void endDate;
  void ongoing;
  return null;
}
