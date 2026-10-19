import styles from './SpreadEditor.module.css';
import { useSpreadEffectText } from './useSpreadEffectText';

export interface SpreadEffectNoteProps {
  activityId: string;
  /** The dates typed in the activity form ("YYYY-MM-DD" or ""). */
  startDate: string;
  endDate: string;
  ongoing: boolean;
}

/**
 * What new activity dates do to its line items' spreads, under the dates of feature 006's activity
 * edit form (FR-022, D-150): "When you save, 3 line items are re-spread over the new dates." Nothing
 * when the dates are the saved ones or the activity has no line items.
 */
export function SpreadEffectNote({
  activityId,
  startDate,
  endDate,
  ongoing,
}: SpreadEffectNoteProps) {
  const text = useSpreadEffectText(activityId, { startDate, endDate, ongoing });
  if (!text) {
    return null;
  }
  return (
    <p role="status" className={styles.note}>
      {text}
    </p>
  );
}
