import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { AttributeForm, AttributeValues } from '@/ui/ShellAttributeFields/rules';
import { BUDGET_CURRENCY, budgetInput, budgetText } from './budget';
import type { Campaign, CampaignCreate, Initiative } from './api';
import type { RecordFormValues } from './schemas';
import type { RecordLevel } from './texts';

export const RECORD_FIELDS = [
  'name',
  'ownerId',
  'startDate',
  'endDate',
  'description',
  'budget',
] as const;

export type RecordBody = Required<CampaignCreate>;

export function valuesOf(record: Campaign | Initiative): RecordFormValues {
  return {
    name: record.name,
    ownerId: record.ownerId ?? '',
    startDate: record.startDate ?? '',
    endDate: record.endDate ?? '',
    description: record.description,
    budget: budgetText(record.budget),
    overspend: record.overspend.mode,
    attributes: record.attributes,
  };
}

export function bodyOf(values: RecordFormValues, currency = BUDGET_CURRENCY): RecordBody {
  return {
    name: values.name,
    description: values.description,
    ownerId: values.ownerId || null,
    startDate: values.startDate || null,
    endDate: values.endDate || null,
    budget: budgetInput(values.budget, currency),
    overspend: values.overspend,
    attributes: values.attributes,
  };
}

export function fieldId(level: RecordLevel, field: string): string {
  return field.startsWith('attributes.') ? field : `${level}-${field}`;
}

export function attributeErrors(problems: FieldProblem[]): Record<string, string> {
  const errors: Record<string, string> = {};
  for (const problem of problems) {
    if (problem.field.startsWith('attributes.') && !(problem.field in errors)) {
      errors[problem.field] = problem.message;
    }
  }
  return errors;
}

export function defaultAttributes(form: AttributeForm): AttributeValues {
  const values: AttributeValues = {};
  for (const section of form.sections) {
    for (const attribute of section.attributes) {
      if (attribute.active && attribute.type === 'yesNo') {
        values[attribute.id] = false;
      }
    }
  }
  return values;
}

export function outsideYear(startDate: string, endDate: string, year: number): boolean {
  return (
    (startDate !== '' && startDate < `${year}-01-01`) ||
    (endDate !== '' && endDate > `${year}-12-31`)
  );
}
