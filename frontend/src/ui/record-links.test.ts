import { describe, expect, it } from 'vitest';
import { fieldHref, recordHref } from './record-links';

describe('recordHref (D-194)', () => {
  it.each([
    ['plan', 'P-1001', undefined, '/plans/P-1001'],
    ['campaign', 'C-1001', 'P-1001', '/plans/P-1001/campaigns/C-1001'],
    ['initiative', 'I-1002', 'P-1001', '/plans/P-1001/initiatives/I-1002'],
    ['activity', 'A-1003', 'P-1001', '/plans/P-1001/activities/A-1003'],
    ['activity', 'A-1003', undefined, '/activities/A-1003'],
    ['lineItem', 'L-1004', 'P-1001', '/line-items/L-1004'],
  ] as const)('links a %s %s to its page', (type, id, planId, expected) => {
    expect(recordHref(type, id, planId)).toBe(expected);
  });

  it('needs the plan for campaigns and initiatives, whose panels open over the plan page', () => {
    expect(() => recordHref('campaign', 'C-1001')).toThrow('needs its plan ID');
    expect(() => recordHref('initiative', 'I-1001')).toThrow('needs its plan ID');
  });
});

describe('fieldHref', () => {
  it.each([
    ['plan', 'P-1001', 'ownerId', undefined, '/plans/P-1001/edit?focus=ownerId'],
    [
      'campaign',
      'C-1001',
      'attributes.SA-1003',
      'P-1001',
      '/plans/P-1001/campaigns/C-1001/edit?focus=attributes.SA-1003',
    ],
    [
      'initiative',
      'I-1001',
      'startDate',
      'P-1001',
      '/plans/P-1001/initiatives/I-1001/edit?focus=startDate',
    ],
    [
      'activity',
      'A-1001',
      'ownerId',
      'P-1001',
      '/plans/P-1001/activities/A-1001/edit?field=ownerId',
    ],
    [
      'lineItem',
      'L-1001',
      'financeBudgetItemId',
      undefined,
      '/line-items/L-1001/edit?field=financeBudgetItemId',
    ],
  ] as const)('links a field of a %s to its edit form', (type, id, field, planId, expected) => {
    expect(fieldHref(type, id, field, planId)).toBe(expected);
  });

  it('needs the plan for an activity field', () => {
    expect(() => fieldHref('activity', 'A-1001', 'ownerId')).toThrow('needs its plan ID');
  });
});
