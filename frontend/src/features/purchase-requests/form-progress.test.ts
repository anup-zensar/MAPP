import { describe, expect, it } from 'vitest';
import { filledText } from '@/ui/FormRail/form-progress';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { attribute } from '@/testing/fixtures/attribute-forms';
import { purchaseRequest } from '@/testing/fixtures/purchase-requests';
import { fromPurchaseRequest } from './form-state';
import { railSections } from './form-progress';

function formOf(
  level: AttributeForm['level'],
  ...attributes: ReturnType<typeof attribute>[]
): AttributeForm {
  return {
    level,
    businessUnitAttributeId: null,
    businessUnitOptions: [],
    sections: [{ name: 'Shell attributes', conditional: false, attributes }],
  };
}

const HEADER = formOf(
  'purchaseRequest',
  attribute({
    id: 'SA-1201',
    label: 'Invoice approver name',
    type: 'text',
    marking: 'neededToSubmit',
  }),
  attribute({ id: 'SA-1202', label: 'Notes', type: 'text', marking: 'optional' }),
);
const LINE = formOf(
  'purchaseRequestLine',
  attribute({ id: 'SA-1301', label: 'GL account', type: 'text', marking: 'neededToSubmit' }),
);

describe('038 FR-026 (M-8): the form counts the needed attributes like the page', () => {
  it('counts an empty needed attribute of the header and of each line', () => {
    const values = fromPurchaseRequest(purchaseRequest());

    const [header, lines] = railSections(values, HEADER, LINE);

    expect(header).toMatchObject({ needed: 6, missing: 1 });
    expect(lines).toMatchObject({ needed: 5, missing: 1 });
  });

  it('counts them filled once they have a value', () => {
    const values = fromPurchaseRequest(
      purchaseRequest({ attributes: { 'SA-1201': 'Daniel Okoro' } }),
    );
    values.lines = values.lines.map((line) => ({ ...line, attributes: { 'SA-1301': '6100' } }));

    expect(filledText(railSections(values, HEADER, LINE))).toBe('11 of 11 needed fields filled');
  });
});
