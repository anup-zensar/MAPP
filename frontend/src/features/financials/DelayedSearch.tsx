import { useEffect, useEffectEvent, useState } from 'react';
import { SearchField } from '@/ui/SearchField/SearchField';
import { SEARCH_PLACEHOLDER } from './financials-text';

/** How long the search waits after the last key before it filters. */
const SEARCH_DELAY_MS = 300;

/**
 * The search box: it filters 300 ms after the last key, and shows the address's search again when
 * that changes elsewhere (Clear filters, Back).
 */
export function DelayedSearch({
  value,
  onChange,
}: {
  value: string;
  onChange: (q: string) => void;
}) {
  const [draft, setDraft] = useState<{ text: string; over: string } | null>(null);
  const text = draft && draft.over === value ? draft.text : value;
  const apply = useEffectEvent((q: string) => onChange(q));

  useEffect(() => {
    if (!draft || draft.over !== value || draft.text === value) {
      return;
    }
    const timer = setTimeout(() => apply(draft.text), SEARCH_DELAY_MS);
    return () => clearTimeout(timer);
  }, [draft, value]);

  return (
    <SearchField
      label="Search transactions"
      placeholder={SEARCH_PLACEHOLDER}
      value={text}
      onChange={(next) => setDraft({ text: next, over: value })}
    />
  );
}
