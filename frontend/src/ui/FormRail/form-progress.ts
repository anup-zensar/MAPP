import { plural } from '@/format/text';

/** One section of a form page as the rail lists it (design system §2.4 C). */
export interface RailSection {
  /** The id of the section's card, the rail's link target ("section-basics"). */
  id: string;
  title: string;
  /** How many required and needed-to-submit fields the section has. */
  needed: number;
  /** How many of them are still empty. */
  missing: number;
}

/** "9 of 15 needed fields filled" (design system §7). */
export function filledText(sections: RailSection[]): string {
  const needed = sections.reduce((sum, section) => sum + section.needed, 0);
  const missing = sections.reduce((sum, section) => sum + section.missing, 0);
  return `${needed - missing} of ${plural(needed, 'needed field')} filled`;
}

/** A section's counts from whether each of its needed fields is filled. */
export function countFilled(filled: boolean[]): { needed: number; missing: number } {
  return { needed: filled.length, missing: filled.filter((each) => !each).length };
}
