import { countFilled } from '@/ui/FormRail/form-progress';
import type { RailSection } from '@/ui/FormRail/form-progress';
import {
  businessUnitOf,
  missingAttributes,
  visibleAttributes,
} from '@/ui/ShellAttributeFields/rules';
import type { AttributeForm, AttributeValues } from '@/ui/ShellAttributeFields/rules';
import type { PurchaseRequestFormValues } from './form-schema';
import { HEADER_SECTION, LINES_SECTION } from './form-text';

interface Counts {
  needed: number;
  missing: number;
}

/** The needed Shell attributes of one set of values, counted as the page's "fields needed" does. */
function attributeCounts(form: AttributeForm, values: AttributeValues): Counts {
  const units = businessUnitOf(form, values, null);
  const needed = visibleAttributes(form, units).filter(
    (attribute) =>
      attribute.active && attribute.marking !== 'optional' && attribute.type !== 'yesNo',
  ).length;
  return { needed, missing: missingAttributes(form, values, units).length };
}

function plus(a: Counts, b: Counts): Counts {
  return { needed: a.needed + b.needed, missing: a.missing + b.missing };
}

/**
 * The rail's sections: the required and needed-to-submit fields still empty — the standard fields
 * and the Shell attributes of the header and of each line (038 FR-026).
 */
export function railSections(
  values: PurchaseRequestFormValues,
  attributeForm: AttributeForm,
  lineAttributeForm: AttributeForm,
): RailSection[] {
  const header = plus(
    countFilled([
      values.name.trim() !== '',
      values.ownerId !== '',
      values.moaApproverId !== '',
      values.approvalWorkflowId !== '',
      values.modificationWorkflowId !== '',
    ]),
    attributeCounts(attributeForm, values.attributes),
  );
  const lines = values.lines.reduce(
    (sum, line) => plus(sum, attributeCounts(lineAttributeForm, line.attributes)),
    countFilled(
      values.lines.flatMap((line) => [
        line.currency !== '',
        line.amount.trim() !== '',
        line.deliveryDate !== '',
        line.overallLimit.trim() !== '',
      ]),
    ),
  );
  return [
    { id: 'pr-header', title: HEADER_SECTION, ...header },
    { id: 'pr-lines', title: LINES_SECTION, ...lines },
  ];
}
