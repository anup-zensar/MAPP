import type { Control } from 'react-hook-form';
import {
  businessUnitOf,
  missingAttributes,
  visibleAttributes,
} from '@/ui/ShellAttributeFields/rules';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import type { Plan } from './api';
import type { PlanFormValues } from './schemas';

/** The form's sections, in the order of design system §7, with the ids the rail links to. */
export const SECTIONS = [
  { id: 'plan-basics', title: 'Basics' },
  { id: 'plan-budget', title: 'Budget' },
  { id: 'plan-attributes', title: 'Shell attributes' },
  { id: 'plan-goals', title: 'Goals' },
] as const;

export type SectionId = (typeof SECTIONS)[number]['id'];

/** "Default: …" while the field still holds the value the defaults gave it (FR-006). */
export type DefaultHint = (
  key: keyof PlanFormValues,
  text: string,
  given?: boolean,
) => string | undefined;

export interface SectionProps {
  control: Control<PlanFormValues>;
  values: PlanFormValues;
  mode: 'create' | 'edit';
  /** Edit: the plan as it was when the form opened. */
  plan?: Plan;
  with016: boolean;
  hint: DefaultHint;
}

interface NeededCount {
  missing: number;
  total: number;
}

export interface NeededFields {
  filled: number;
  total: number;
  bySection: Record<SectionId, NeededCount>;
}

/** The needed fields of each section and how many are still empty (design system §2.4 C, §7). */
export function neededFields(
  values: PlanFormValues,
  form: AttributeForm,
  with016: boolean,
): NeededFields {
  const count = (checks: boolean[]): NeededCount => ({
    missing: checks.filter((filled) => !filled).length,
    total: checks.length,
  });
  const units = businessUnitOf(form, values.attributes, null);
  const attributesTotal = visibleAttributes(form, units).filter(
    (attribute) =>
      attribute.active && attribute.marking !== 'optional' && attribute.type !== 'yesNo',
  ).length;
  const attributesMissing = missingAttributes(form, values.attributes, units).length;
  const bySection: Record<SectionId, NeededCount> = {
    'plan-basics': count([
      values.name.trim() !== '',
      values.year !== '',
      values.modelId !== '',
      values.teamIds.length > 0,
      values.ownerId !== '',
    ]),
    'plan-budget': count(with016 ? [values.currency !== ''] : []),
    'plan-attributes': {
      missing: Math.min(attributesMissing, attributesTotal),
      total: attributesTotal,
    },
    'plan-goals': count([values.goalStage !== undefined]),
  };
  const sections = Object.values(bySection);
  const total = sections.reduce((sum, section) => sum + section.total, 0);
  const missing = sections.reduce((sum, section) => sum + section.missing, 0);
  return { filled: total - missing, total, bySection };
}

/** "9 of 15 needed fields filled". */
export function progressText({ filled, total }: { filled: number; total: number }): string {
  return `${filled} of ${total} needed fields filled`;
}
