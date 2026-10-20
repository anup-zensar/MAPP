/**
 * The campaign and initiative forms' values (plan "Forms"): what the fields hold, what is sent,
 * and where each problem is shown.
 */
import type { AttributeForm, AttributeValues, FieldProblem } from '@/components';
import { BUDGET_CURRENCY, budgetInput, budgetText } from './budget';
import type { Campaign, CampaignCreate, Initiative } from './campaignApi';
import type { RecordLevel } from './texts';

/** What the form's fields hold. Dates are "YYYY-MM-DD" or ""; the budget is the typed text. */
export interface RecordFormValues {
  name: string;
  ownerId: string;
  startDate: string;
  endDate: string;
  description: string;
  budget: string;
  attributes: AttributeValues;
}

/** Every field of a create or update request but the version (CampaignCreate = InitiativeCreate). */
export type RecordBody = Required<CampaignCreate>;

/** A stored campaign or initiative as the edit form starts from it. */
export function valuesOf(record: Campaign | Initiative): RecordFormValues {
  return {
    name: record.name,
    ownerId: record.ownerId ?? '',
    startDate: record.startDate ?? '',
    endDate: record.endDate ?? '',
    description: record.description,
    budget: budgetText(record.budget),
    attributes: record.attributes,
  };
}

/**
 * The request body: the API stores exactly what it receives (Decision D); empty fields are sent
 * as null or "", and the name as typed (the API trims and checks it). The budget is in the plan
 * currency (016 FR-011).
 */
export function bodyOf(values: RecordFormValues, currency = BUDGET_CURRENCY): RecordBody {
  return {
    name: values.name,
    description: values.description,
    ownerId: values.ownerId || null,
    startDate: values.startDate || null,
    endDate: values.endDate || null,
    budget: budgetInput(values.budget, currency),
    attributes: values.attributes,
  };
}

/** Whether two forms hold the same values (unsaved changes). */
export function sameValues(a: RecordFormValues, b: RecordFormValues): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

/**
 * The input id of a field the API names ("name", "startDate", "attributes.SA-1003"), for the error
 * summary and `?focus=`: "campaign-name", "campaign-startDate"; attributes keep the id
 * ShellAttributeFields gives them.
 */
export function fieldId(level: RecordLevel, field: string): string {
  return field.startsWith('attributes.') ? field : `${level}-${field}`;
}

/** The problems with Shell attributes, by field name, for ShellAttributeFields. */
export function attributeErrors(problems: FieldProblem[]): Record<string, string> {
  const errors: Record<string, string> = {};
  for (const problem of problems) {
    if (problem.field.startsWith('attributes.') && !(problem.field in errors)) {
      errors[problem.field] = problem.message;
    }
  }
  return errors;
}

/** FR-003: yes/no attributes start as No on a new record. */
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

/** R4: dates that start before 1 January or end after 31 December of the plan's year. */
export function outsideYear(startDate: string, endDate: string, year: number): boolean {
  return (
    (startDate !== '' && startDate < `${year}-01-01`) ||
    (endDate !== '' && endDate > `${year}-12-31`)
  );
}
