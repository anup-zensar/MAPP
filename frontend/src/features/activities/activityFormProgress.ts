import { missingAttributes, visibleAttributes } from '@/components';
import type { AttributeForm } from '@/components';
import type { FormState } from './activityFormState';
import { countFilled as count } from './formProgress';
import type { RailSection } from './formProgress';

/** The ids of the activity form's section cards, which the rail links to. */
export const ACTIVITY_SECTION = {
  basics: 'section-basics',
  dates: 'section-dates',
  attributes: 'section-attributes',
  goals: 'section-goals',
  more: 'section-more',
} as const;

/** The Shell attributes that must be filled before submitting: shown, active, not optional. */
function neededAttributeCount(form: AttributeForm, businessUnit: string[]): number {
  return visibleAttributes(form, businessUnit).filter(
    (attribute) =>
      attribute.active && attribute.marking !== 'optional' && attribute.type !== 'yesNo',
  ).length;
}

/**
 * The activity form's sections with how many required and needed-to-submit fields each still
 * lacks (design system §2.4 C, §7; functional review P-04): the rail's links and the count "9 of
 * 15 needed fields filled". Shell attributes count once their form has loaded.
 */
export function activityRailSections(
  value: FormState,
  attributeForm: AttributeForm | undefined,
  businessUnit: string[],
): RailSection[] {
  const attributes = attributeForm
    ? {
        needed: neededAttributeCount(attributeForm, businessUnit),
        missing: missingAttributes(attributeForm, value.attributes, businessUnit).length,
      }
    : { needed: 0, missing: 0 };
  return [
    {
      id: ACTIVITY_SECTION.basics,
      title: 'Basics',
      ...count([
        value.name.trim() !== '',
        value.activityTypeId !== '',
        value.ownerId !== '',
        value.approvalWorkflowId !== '',
      ]),
    },
    {
      id: ACTIVITY_SECTION.dates,
      title: 'Dates & cost',
      ...count([
        value.startDate !== '' && (value.ongoing || value.endDate !== ''),
        value.plannedCost.trim() !== '',
      ]),
    },
    { id: ACTIVITY_SECTION.attributes, title: 'Shell attributes', ...attributes },
    { id: ACTIVITY_SECTION.goals, title: 'Goals', needed: 0, missing: 0 },
    { id: ACTIVITY_SECTION.more, title: 'More details', needed: 0, missing: 0 },
  ];
}
