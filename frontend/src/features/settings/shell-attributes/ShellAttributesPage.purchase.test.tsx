import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import {
  offered,
  renderAttributeEditor,
  renderShellAttributesPage,
  rowNames,
} from '@/testing/fixtures/shell-attribute-pages';
import {
  ACTIVITY_ATTRIBUTES,
  BUSINESS_UNIT,
  emptyForm,
  page,
  shellAttribute,
} from '@/testing/fixtures/shell-attributes';
import { reply } from '@/testing/replies';

const VENDOR = shellAttribute({
  id: 'SA-1101',
  label: 'Vendor',
  level: 'purchaseRequest',
  type: 'text',
  marking: 'neededToSubmit',
  section: 'Purchase request',
  locks: [{ field: 'delete', reason: 'SP&A One reads this attribute for purchase requests.' }],
});
const MATERIAL_GROUP = shellAttribute({
  id: 'SA-1121',
  label: 'Material group',
  level: 'purchaseRequestLine',
  type: 'text',
  section: 'PR line',
});

const PR_FORMS = [
  reply(
    'GET /api/v1/shell-attribute-forms/{level}',
    { body: emptyForm('purchaseRequest') },
    { level: 'purchaseRequest' },
  ),
  reply(
    'GET /api/v1/shell-attribute-forms/{level}',
    { body: emptyForm('purchaseRequestLine') },
    { level: 'purchaseRequestLine' },
  ),
];

describe('032 FR-006: Purchase request and PR line levels', () => {
  it('lists the two purchase-request levels with their attributes', async () => {
    const { user } = renderShellAttributesPage('/settings/shell-attributes', [
      reply('GET /api/v1/shell-attributes', {
        body: page([BUSINESS_UNIT, ...ACTIVITY_ATTRIBUTES, VENDOR, MATERIAL_GROUP]),
      }),
    ]);

    const levels = await screen.findByRole('group', { name: 'Level' });
    expect(within(levels).getByRole('button', { name: 'PR line 1' })).toBeInTheDocument();
    await user.click(within(levels).getByRole('button', { name: 'Purchase request 1' }));

    expect(rowNames()).toEqual(['Purchase request', 'Vendor']);
    expect(
      within(screen.getByRole('row', { name: /Vendor/ })).getByText('Needed to submit'),
    ).toBeVisible();
    await expectNoAxeViolations();
  });

  it('offers Optional and Needed to submit, and no % split, on the purchase-request levels', async () => {
    const { user } = renderAttributeEditor('/settings/shell-attributes/new', PR_FORMS);

    await screen.findByRole('heading', { level: 1, name: 'New Shell attribute' });
    await user.selectOptions(screen.getByRole('combobox', { name: /Level/ }), 'purchaseRequest');
    expect(offered(/^Marking/)).toEqual(['Optional', 'Needed to submit']);
    expect(offered(/^Type/)).not.toContain('Multiple choice with % split');

    await user.selectOptions(
      screen.getByRole('combobox', { name: /Level/ }),
      'purchaseRequestLine',
    );
    expect(offered(/^Marking/)).toEqual(['Optional', 'Needed to submit']);
    expect(offered(/^Type/)).not.toContain('Multiple choice with % split');
  });
});
