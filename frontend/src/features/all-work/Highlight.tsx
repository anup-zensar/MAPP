import styles from './Highlight.module.css';

export interface HighlightProps {
  text: string;
  /** The searched text; every place it occurs in `text`, ignoring letter case, is marked. */
  search: string;
}

/** A name or excerpt with the searched text marked (FR-006). */
export function Highlight({ text, search }: HighlightProps) {
  const needle = search.trim().toLowerCase();
  // Lower case must keep the length, or the places found would not fit the original text.
  const haystack = text.toLowerCase();
  if (needle === '' || haystack.length !== text.length) {
    return <>{text}</>;
  }
  const parts: { text: string; hit: boolean }[] = [];
  let from = 0;
  let at = haystack.indexOf(needle);
  while (at !== -1) {
    if (at > from) parts.push({ text: text.slice(from, at), hit: false });
    parts.push({ text: text.slice(at, at + needle.length), hit: true });
    from = at + needle.length;
    at = haystack.indexOf(needle, from);
  }
  if (from < text.length) parts.push({ text: text.slice(from), hit: false });

  return (
    <>
      {parts.map((part, index) =>
        part.hit ? (
          <mark key={index} className={styles.mark}>
            {part.text}
          </mark>
        ) : (
          part.text
        ),
      )}
    </>
  );
}
