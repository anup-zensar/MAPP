import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { useFieldHelp } from './useFieldHelp';

export interface FieldHelpTipProps {
  /** The Field help key, such as "plan.budget" (016 data-model §6). */
  helpKey: string;
  /** The label it explains: the ⓘ is named "About <label>". */
  label: string;
  /** The text to show until the entries arrive, such as the page's former fixed tip. */
  fallback?: string;
}

/**
 * The ⓘ of a built-in value that is not a form field — a summary card, a details label, a column
 * header (016 FR-047) — with its Field help text; nothing when the text is empty. Form fields pass
 * `tip={help(key)}` from useFieldHelp() instead.
 */
export function FieldHelpTip({ helpKey, label, fallback }: FieldHelpTipProps) {
  const { help } = useFieldHelp();
  const text = help(helpKey, fallback);
  return text ? <InfoTip label={label} text={text} /> : null;
}
