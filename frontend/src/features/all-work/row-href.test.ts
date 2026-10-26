import { describe, expect, it } from 'vitest';
import { rowHref } from './row-href';

describe('rowHref (FR-016, D-194)', () => {
  it.each([
    ['plan', 'P-1001', '/plans/P-1001'],
    ['campaign', 'C-1001', '/plans/P-1001/campaigns/C-1001'],
    ['initiative', 'I-1001', '/plans/P-1001/initiatives/I-1001'],
    ['activity', 'A-1001', '/plans/P-1001/activities/A-1001'],
    ['lineItem', 'L-1001', '/line-items/L-1001'],
    ['unassigned', 'A-1001', '/plans/P-1001/activities/A-1001'],
  ] as const)('opens a %s row at its record’s page', (level, recordId, href) => {
    expect(rowHref({ level, recordId, planId: 'P-1001' })).toBe(href);
  });
});
