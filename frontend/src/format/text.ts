/**
 * Small helpers for sentences built from data (feature 002 plan, "Front-end rules mirrored from
 * Go"): a label in the middle of a sentence, counted words and lists joined with "and".
 */
import { formatWhole } from './decimal';

/**
 * A label as it reads in the middle of a sentence: the first letter becomes lower case only when
 * the second one is lower case too — "Region" → "region", "Business unit" → "business unit", but
 * "GL code" and "SMP" stay as they are (Go's midSentence).
 */
export function midSentence(label: string): string {
  const [first, second] = label;
  if (first === undefined || second === undefined || second.toLowerCase() !== second) {
    return label;
  }
  if (second.toUpperCase() === second) {
    // The second character is not a letter ("A-team"): keep the label.
    return label;
  }
  return first.toLowerCase() + label.slice(1);
}

/**
 * A count with its word, singular for exactly 1: "1 model", "5 models", "1,217 options",
 * "2 activities" (plural("activity", …, "activities")).
 */
export function plural(count: number | bigint, singular: string, pluralForm?: string): string {
  const word = count === 1 || count === 1n ? singular : (pluralForm ?? `${singular}s`);
  return `${formatWhole(count)} ${word}`;
}

/** "a", "a and b", "a, b and c": a list in a sentence (design system §8). */
export function joinWithAnd(parts: string[]): string {
  if (parts.length <= 1) {
    return parts.join('');
  }
  return `${parts.slice(0, -1).join(', ')} and ${parts[parts.length - 1]}`;
}
