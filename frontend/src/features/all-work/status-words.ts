/**
 * One word per status colour (design system §4.1, §11 rule 6; functional review A-02, UI audit
 * X-14): the API writes the approval roll-up of plans, campaigns and initiatives as "Partly
 * approved", so every label is shown as sent.
 */

/** The word shown for a status chip of All work. */
export function statusWord(label: string): string {
  return label;
}
