import { missingAttributes, visibleAttributes } from '@/ui/ShellAttributeFields/rules';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { countFilled } from '@/ui/FormRail/form-progress';
import type { RailSection } from '@/ui/FormRail/form-progress';
import type { LineItemFormState } from './line-item-form-state';
import {
  SECTION_BASICS,
  SECTION_BUDGET,
  SECTION_COST,
  SECTION_FINANCE_CODING,
  SECTION_GSAP,
  SECTION_SPREAD,
} from './line-item-text';

/** The ids of the line-item form's section cards, which the rail links to. */
export const LINE_ITEM_SECTION = {
  basics: 'section-basics',
  cost: 'section-cost',
  spread: 'section-spread',
  coding: 'section-coding',
  budget: 'section-budget',
  gsap: 'section-gsap',
} as const;

/**
 * The line-item form's sections with how many required and needed-to-submit fields each still
 * lacks (design system §2.4 C, §7): name and owner; currency and planned cost; the needed finance
 * coding attributes; the finance budget year and item.
 */
export function lineItemRailSections(
  state: LineItemFormState,
  attributeForm: AttributeForm,
  businessUnit: string[],
): RailSection[] {
  const needed = visibleAttributes(attributeForm, businessUnit).filter(
    (attribute) =>
      attribute.active && attribute.marking !== 'optional' && attribute.type !== 'yesNo',
  ).length;
  const missing = missingAttributes(attributeForm, state.attributes, businessUnit).length;
  return [
    {
      id: LINE_ITEM_SECTION.basics,
      title: SECTION_BASICS,
      ...countFilled([state.name.trim() !== '', state.ownerId !== '']),
    },
    {
      id: LINE_ITEM_SECTION.cost,
      title: SECTION_COST,
      ...countFilled([state.currency !== '', state.plannedCost.trim() !== '']),
    },
    { id: LINE_ITEM_SECTION.spread, title: SECTION_SPREAD, needed: 0, missing: 0 },
    { id: LINE_ITEM_SECTION.coding, title: SECTION_FINANCE_CODING, needed, missing },
    {
      id: LINE_ITEM_SECTION.budget,
      title: SECTION_BUDGET,
      ...countFilled([state.financeBudgetYear !== null, state.financeBudgetItemId !== '']),
    },
    { id: LINE_ITEM_SECTION.gsap, title: SECTION_GSAP, needed: 0, missing: 0 },
  ];
}
