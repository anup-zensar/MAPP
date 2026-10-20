import { useMemo } from 'react';
import { useFieldHelpEntries } from './fieldHelpApi';

/** What useFieldHelp() gives a page. */
export interface FieldHelp {
  /**
   * The help text of a built-in field, such as help('plan.budget'), for its ⓘ tip:
   * - once the entries have arrived: the text an administrator keeps in Settings › Field help, or
   *   undefined when it is empty — an empty text hides the ⓘ (FR-046);
   * - before they arrive, or when they can't be loaded: `fallback` (a page's former fixed tip),
   *   else undefined, so the ⓘ simply appears once the text is known.
   * A key that is not in the catalogue answers `fallback` too.
   */
  help: (key: string, fallback?: string) => string | undefined;
  /** The entries have arrived. */
  ready: boolean;
}

/**
 * The ⓘ texts of the built-in fields (016 US7, FR-047; plan "Shared parts", D-271), for every
 * form, details page, panel and summary card of the planning levels and All work:
 *
 *   const { help } = useFieldHelp();
 *   <MoneyField label="Plan budget" tip={help('plan.budget')} … />
 *
 * All the texts come in one request per page load (GET /api/field-help, asked again when a page
 * opens), so a text edited in Settings shows on the next form anyone opens (SC-008).
 */
export function useFieldHelp(): FieldHelp {
  const entries = useFieldHelpEntries();
  const texts = useMemo(() => {
    const byKey = new Map<string, string>();
    for (const entry of entries.data?.items ?? []) {
      byKey.set(entry.key, entry.text.trim());
    }
    return byKey;
  }, [entries.data]);
  const ready = entries.data !== undefined;

  return {
    ready,
    help: (key, fallback) => {
      const text = ready ? texts.get(key) : undefined;
      if (text === undefined) {
        return fallback;
      }
      return text === '' ? undefined : text;
    },
  };
}
