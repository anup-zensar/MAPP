import { countFilled } from '@/ui/FormRail/form-progress';
import type { RailSection } from '@/ui/FormRail/form-progress';
import type { PurchaseRequestFormValues } from './form-schema';
import { HEADER_SECTION, LINES_SECTION } from './form-text';

/** The rail's sections: the required and needed-to-submit standard fields still empty. */
export function railSections(values: PurchaseRequestFormValues): RailSection[] {
  const header = countFilled([
    values.name.trim() !== '',
    values.ownerId !== '',
    values.moaApproverId !== '',
    values.approvalWorkflowId !== '',
    values.modificationWorkflowId !== '',
  ]);
  const lines = countFilled(
    values.lines.flatMap((line) => [
      line.currency !== '',
      line.amount.trim() !== '',
      line.deliveryDate !== '',
      line.overallLimit.trim() !== '',
    ]),
  );
  return [
    { id: 'pr-header', title: HEADER_SECTION, ...header },
    { id: 'pr-lines', title: LINES_SECTION, ...lines },
  ];
}
