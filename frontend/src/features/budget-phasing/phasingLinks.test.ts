import { describe, expect, it } from 'vitest';
import { completenessFieldHref, phasingHref } from './phasingLinks';

describe('phasingLinks (016 T053)', () => {
  it('sends the phasing problem to Budget by month on the page or in the panel', () => {
    expect(completenessFieldHref('plan', 'P-1004', 'budgetPhasing', 'P-1004')).toBe(
      '/plans/P-1004#budget-by-month-P-1004',
    );
    expect(completenessFieldHref('campaign', 'C-1005', 'budgetPhasing', 'P-1004')).toBe(
      '/plans/P-1004/campaigns/C-1005#budget-by-month-C-1005',
    );
    expect(phasingHref('initiative', 'I-1009', 'P-1004')).toBe(
      '/plans/P-1004/initiatives/I-1009#budget-by-month-I-1009',
    );
  });

  it('sends every other field to the edit form with that field focused', () => {
    expect(completenessFieldHref('plan', 'P-1004', 'ownerId', 'P-1004')).toBe(
      '/plans/P-1004/edit?focus=ownerId',
    );
    expect(completenessFieldHref('campaign', 'C-1005', 'startDate', 'P-1004')).toBe(
      '/plans/P-1004/campaigns/C-1005/edit?focus=startDate',
    );
  });
});
