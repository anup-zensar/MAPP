import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
import { emptyCondition } from './levelDrafts';
import type { ConditionDraft } from './levelDrafts';
import { RuleConditionFields } from './RuleConditionFields';
import { RULE_OPTIONS } from './testData';

function Harness({ reason = '' }: { reason?: string }) {
  const [value, setValue] = useState<ConditionDraft>(emptyCondition());
  return (
    <>
      <RuleConditionFields
        label="Level 1: only needed when"
        idPrefix="levels.1.appliesWhen"
        value={value}
        onChange={setValue}
        options={reason ? undefined : RULE_OPTIONS.body}
        optionsReason={reason}
      />
      <output>{JSON.stringify(value)}</output>
    </>
  );
}

describe('RuleConditionFields (feature 017, US1)', () => {
  it('has the four parts, amounts in US dollars, inactive values offered only while chosen (FR-005)', async () => {
    const { user } = renderWithProviders(<Harness />, { user: testUser() });

    const group = screen.getByRole('group', { name: 'Level 1: only needed when' });
    const below = within(group).getByRole('textbox', { name: /Planned cost below \(USD\)/ });
    await user.type(below, '10000');
    await user.tab();
    expect(below).toHaveValue('10,000.00');
    expect(
      within(group).getByRole('group', { name: /Business unit is one of/ }),
    ).toBeInTheDocument();
    expect(
      within(group)
        .getAllByRole('checkbox')
        .map((box) => box.getAttribute('id')),
    ).toEqual([
      'levels.1.appliesWhen.businessUnitIds-0',
      'levels.1.appliesWhen.businessUnitIds-1',
      'levels.1.appliesWhen.activityTypeIds-0',
      'levels.1.appliesWhen.activityTypeIds-1',
    ]);
    await user.click(within(group).getByRole('checkbox', { name: 'Sponsorship' }));

    expect(JSON.parse(document.querySelector('output')?.textContent ?? '{}')).toEqual({
      plannedCostAtLeast: '',
      plannedCostBelow: '10,000.00',
      businessUnitIds: [],
      activityTypeIds: ['AT-1002'],
    });
  });

  it('says why the lists cannot be used while their choices load', () => {
    renderWithProviders(<Harness reason="Loading the choices…" />, { user: testUser() });

    expect(screen.getAllByText('Loading the choices…')).toHaveLength(2);
  });
});
