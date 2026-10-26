/**
 * Field help for other features (016 US7, FR-047): the ⓘ texts of the built-in fields that
 * administrators keep in Settings › Field help.
 *
 *   import { FieldHelpTip, useFieldHelp } from '@/features/settings/field-help';
 *
 *   const { help } = useFieldHelp();
 *   <MoneyField label="Plan budget" tip={help('plan.budget')} … />   // a form field
 *   <FieldHelpTip helpKey="plan.plannedCost" label="Planned cost" />  // any other label
 *
 * Tests of a page that uses them answer 'GET /api/v1/field-help' with mockApi.
 */
export { FIELD_HELP_KEY } from './fieldHelpApi';
export type { FieldHelpEntry, FieldHelpLevel } from './fieldHelpApi';
export { FieldHelpTip } from './FieldHelpTip';
export type { FieldHelpTipProps } from './FieldHelpTip';
export { useFieldHelp } from './useFieldHelp';
export type { FieldHelp } from './useFieldHelp';
