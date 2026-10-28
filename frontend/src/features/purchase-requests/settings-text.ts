export const CONFIG_DESCRIPTION =
  'Whether purchase requests are used, and whether lines may be added to approved purchase requests.';
export const PURCHASE_SUPPORT = 'Purchase Support';
export const PURCHASE_SUPPORT_HINT =
  'Off: nobody can raise, change, submit, approve, close or delete purchase requests; existing ones stay readable.';
export const LINES_ON_APPROVED = 'Lines on approved purchase requests';
export const LINES_ON_APPROVED_HINT =
  'What happens when someone adds a line to an approved purchase request.';
export const ON = 'On';
export const OFF = 'Off';
export const SAVE = 'Save';
export const CONFIG_SAVED = 'Purchase request settings saved';
export const CONFIG_NOUN = 'purchase request configuration';

export function lastChangedText(name: string, when: string): string {
  return `Last changed by ${name} on ${when}`;
}
