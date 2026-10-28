export const LINE_ITEM_TIP =
  'Purchase requests raised on this line item. Their amounts are reserved on it from the first save, whatever the status.';
export const ACTIVITY_TIP =
  "What the purchase requests on this activity's line items reserve, in the plan currency.";
export const NOT_VISIBLE = "You can't see the purchase requests here.";
export const NO_LINE_ITEM_PRS = 'No purchase requests on this line item yet.';
export const NO_ACTIVITY_PRS = "No purchase requests on this activity's line items yet.";
export const TOTAL_REQUESTED = 'Total requested';
export const AVAILABLE = 'Available to request';
export const PLANNED_COST = 'Planned cost';

/** "Requested above planned cost by 120.00 EUR": rates moved a line in another currency (D-303). */
export function aboveText(amount: string): string {
  return `Requested above planned cost by ${amount}`;
}

export function convertedText(amount: string): string {
  return `${amount} in the line item's currency`;
}
