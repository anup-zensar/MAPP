import { describe, expect, it } from 'vitest';
import { activityForm, campaignForm, LUBRICANTS, MOBILITY, planForm } from '@/test/attributeForms';
import {
  allowedOptions,
  autoFill,
  businessUnitOf,
  missingAttributes,
  noOptionsForParent,
  valuesToClear,
  visibleAttributes,
} from './rules';

// The same cases as Go's settings form_test.go (feature 002, FR-024 – FR-027).

function attributeOf(form: ReturnType<typeof campaignForm>, id: string) {
  const found = form.sections.flatMap((section) => section.attributes).find((a) => a.id === id);
  if (!found) {
    throw new Error(`No attribute ${id}`);
  }
  return found;
}

describe('businessUnitOf and visibleAttributes (FR-023, FR-026)', () => {
  it('takes the plan’s business unit below the plan, and the form’s own value on the plan form', () => {
    expect(businessUnitOf(campaignForm(), {}, [LUBRICANTS])).toEqual([LUBRICANTS]);
    expect(businessUnitOf(planForm(), { 'SA-BU': [MOBILITY] }, null)).toEqual([MOBILITY]);
    expect(businessUnitOf(planForm(), {}, null)).toEqual([]);
  });

  it('shows a conditional attribute only when the business unit includes one of its options', () => {
    const ids = (units: string[]) => visibleAttributes(campaignForm(), units).map((a) => a.id);

    expect(ids([LUBRICANTS])).toContain('SA-BRANDS');
    expect(ids([MOBILITY])).not.toContain('SA-BRANDS');
    expect(ids([])).not.toContain('SA-BRANDS');
    expect(ids([])).toContain('SA-CLUSTER');
  });
});

describe('allowedOptions and noOptionsForParent (FR-025, D-034)', () => {
  it('offers only active options linked to a chosen parent option', () => {
    const form = campaignForm();
    const region = attributeOf(form, 'SA-REGION');

    expect(allowedOptions(region, {})).toEqual([]);
    expect(allowedOptions(region, { 'SA-CLUSTER': ['SO-B2B'] }).map((o) => o.label)).toEqual([
      'B2B – EUROPE',
      'B2B – ASIA',
    ]);
  });

  it('knows when the chosen parent options have no options at all (D-034)', () => {
    const form = campaignForm();
    const region = attributeOf(form, 'SA-REGION');

    expect(noOptionsForParent(region, { 'SA-CLUSTER': ['SO-FLEET'] })).toBe(true);
    expect(noOptionsForParent(region, { 'SA-CLUSTER': ['SO-RETAIL'] })).toBe(false);
    expect(noOptionsForParent(region, {})).toBe(false);
  });
});

describe('valuesToClear (FR-025, US3-4)', () => {
  it('clears dependent values that no longer fit a parent changed in this save', () => {
    const cleared = valuesToClear(
      campaignForm(),
      { 'SA-CLUSTER': ['SO-RETAIL'], 'SA-REGION': ['SO-CHINA', 'SO-EU'] },
      { 'SA-CLUSTER': ['SO-RETAIL', 'SO-B2B'], 'SA-REGION': ['SO-CHINA', 'SO-EU'] },
    );

    expect(cleared).toEqual([
      {
        attributeId: 'SA-REGION',
        label: 'Region',
        optionIds: ['SO-EU'],
        optionLabels: ['B2B – EUROPE'],
      },
    ]);
  });

  it('keeps values whose parent did not change, even when they no longer fit', () => {
    const values = { 'SA-CLUSTER': ['SO-RETAIL'], 'SA-REGION': ['SO-EU'] };

    expect(valuesToClear(campaignForm(), values, values)).toEqual([]);
  });

  it('treats every parent value as new on a create form', () => {
    expect(
      valuesToClear(campaignForm(), { 'SA-CLUSTER': ['SO-RETAIL'], 'SA-REGION': ['SO-EU'] }),
    ).toHaveLength(1);
  });
});

describe('missingAttributes (FR-026, FR-027)', () => {
  it('lists active, shown, needed attributes without a value', () => {
    const missing = missingAttributes(campaignForm(), {}, [LUBRICANTS]);

    expect(missing).toEqual([
      { field: 'attributes.SA-CLUSTER', label: 'Cluster' },
      { field: 'attributes.SA-REGION', label: 'Region' },
      { field: 'attributes.SA-BRANDS', label: 'Brands' },
    ]);
  });

  it('never counts hidden attributes or dependents without options for their parent (D-034)', () => {
    const missing = missingAttributes(campaignForm(), { 'SA-CLUSTER': ['SO-FLEET'] }, [MOBILITY]);

    expect(missing).toEqual([]);
  });

  it('counts an inactive option as a value, and never an inactive attribute (FR-027)', () => {
    expect(
      missingAttributes(campaignForm(), { 'SA-CLUSTER': ['SO-B2B'], 'SA-REGION': ['SO-OLD'] }, []),
    ).toEqual([]);
    expect(missingAttributes(activityForm(), {}, []).map((m) => m.label)).toEqual(['Brand split']);
  });

  it('counts a split only when its shares add up to 100.00 % (R10)', () => {
    const half = { 'SA-SPLIT': [{ optionId: 'SO-HELIX', share: '90.00' }] };

    expect(missingAttributes(activityForm(), half, [])).toEqual([
      {
        field: 'attributes.SA-SPLIT',
        label: 'Brand split',
        reason: 'Shares add up to 90.00 % — they must add up to 100.00 %.',
      },
    ]);
    const full = { 'SA-SPLIT': [{ optionId: 'SO-HELIX', share: '100' }] };
    expect(missingAttributes(activityForm(), full, [])).toEqual([]);
  });
});

describe('autoFill (design system §7)', () => {
  it('fills in an empty dependent when exactly one option fits', () => {
    const result = autoFill(campaignForm(), { 'SA-CLUSTER': ['SO-RETAIL'] }, 'SA-CLUSTER');

    expect(result.values['SA-REGION']).toEqual(['SO-CHINA']);
    expect(result.filled).toEqual([{ attributeId: 'SA-REGION', optionId: 'SO-CHINA' }]);
  });

  it('leaves it alone when several fit or it has a value', () => {
    expect(autoFill(campaignForm(), { 'SA-CLUSTER': ['SO-B2B'] }, 'SA-CLUSTER').filled).toEqual([]);
    const chosen = { 'SA-CLUSTER': ['SO-RETAIL'], 'SA-REGION': ['SO-EU'] };
    expect(autoFill(campaignForm(), chosen, 'SA-CLUSTER').filled).toEqual([]);
  });
});
