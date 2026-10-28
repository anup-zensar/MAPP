import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { PEOPLE, RULE_OPTIONS } from '@/testing/fixtures/approval-workflows';
import { reply } from '@/testing/replies';
import { emptyLevel, toInputs } from './level-drafts';
import type { LevelDraft } from './level-drafts';
import { LevelsEditor } from './LevelsEditor';
import type { ObjectType } from './schemas';

function Harness({ objectType }: { objectType: ObjectType }) {
  const [levels, setLevels] = useState<LevelDraft[]>([{ ...emptyLevel(), key: 'a', mode: 'all' }]);
  return (
    <>
      <LevelsEditor levels={levels} onChange={setLevels} problems={[]} objectType={objectType} />
      <output>{JSON.stringify(toInputs(levels))}</output>
    </>
  );
}

function renderEditor(objectType: ObjectType) {
  mockApi([
    reply('GET /api/v1/people', PEOPLE),
    reply('GET /api/v1/approval-rule-options', RULE_OPTIONS),
  ]);
  return renderWithProviders(<Harness objectType={objectType} />, { user: testUser() });
}

const sent = (): unknown => JSON.parse(document.querySelector('output')?.textContent ?? '[]');
const sources = () =>
  within(screen.getByRole('group', { name: 'Who approves' }))
    .getAllByRole('radio')
    .map((radio) => radio.closest('label')?.textContent);

describe('LevelsEditor sources (feature 031, US3)', () => {
  it('an activity workflow offers the record owner but no person named on the record (031 FR-010, FR-011)', async () => {
    const { user } = renderEditor('activity');

    expect(sources()).toEqual([
      'Named people',
      'Record owner',
      "The owner's line manager",
      'People with a role',
    ]);
    await user.click(screen.getByRole('radio', { name: 'Record owner' }));

    expect(screen.queryByRole('combobox', { name: /Approvers/ })).not.toBeInTheDocument();
    expect(
      screen.queryByRole('group', { name: 'When there are several approvers' }),
    ).not.toBeInTheDocument();
    expect(sent()).toEqual([
      expect.objectContaining({ approverType: 'owner', approverIds: [], mode: 'any' }),
    ]);
  });

  it('a purchase request workflow names the MoA approver as the one approver (031 FR-011, FR-013)', async () => {
    const { user } = renderEditor('purchaseRequest');

    expect(sources()).toContain('Person named on the record');
    await user.click(screen.getByRole('radio', { name: 'Person named on the record' }));
    const field = await screen.findByRole('combobox', { name: /Field that names the approver/ });
    await screen.findByRole('option', { name: 'MoA approver' });
    await user.selectOptions(field, 'MoA approver');

    expect(
      screen.queryByRole('group', { name: 'When there are several approvers' }),
    ).not.toBeInTheDocument();
    expect(sent()).toEqual([
      expect.objectContaining({
        approverType: 'recordField',
        recordField: 'moaApprover',
        approverIds: [],
        mode: 'any',
      }),
    ]);
    await expectNoAxeViolations();
  });
});
