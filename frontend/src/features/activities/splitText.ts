// Split activity's texts (016 plan, "User-visible messages").

export const NOTHING_CHOSEN = 'Choose line items or an amount to move.';
export const REAPPROVAL = 'This activity will need approval again.';
export const SPLIT_ACTIVITY = 'Split activity';

/** "Split 'UK launch event'". */
export function splitTitle(name: string): string {
  return `Split '${name}'`;
}

/** "Goals stay on 'UK launch event'." */
export function goalsStayText(name: string): string {
  return `Goals stay on '${name}'.`;
}

/** "'UK launch event' split — 'Ireland roadshow' created." */
export function splitToast(original: string, created: string): string {
  return `'${original}' split — '${created}' created.`;
}

/** "Split from 'UK launch event' (A-1012)", with "(deleted)" when it is in the recycle bin. */
export function splitFromText(name: string, id: string, deleted: boolean): string {
  return `Split from '${name}' (${id})${deleted ? ' (deleted)' : ''}`;
}

/** "Split into 'Ireland roadshow' (A-1014)", with "(deleted)" when it is in the recycle bin. */
export function splitIntoText(name: string, id: string, deleted: boolean): string {
  return `Split into '${name}' (${id})${deleted ? ' (deleted)' : ''}`;
}
