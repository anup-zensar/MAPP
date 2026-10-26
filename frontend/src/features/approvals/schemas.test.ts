import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { approvalsAddressOf } from './schemas';

const TABS = ['waiting', 'open', 'decided', 'monitoring'] as const;

describe('the approvals page address (008 FR-029, 017 FR-024)', () => {
  it('never fails on any address, and names only a tab that exists', () => {
    fc.assert(
      fc.property(fc.dictionary(fc.string(), fc.string()), (query) => {
        const address = approvalsAddressOf(new URLSearchParams(query));
        expect([null, ...TABS]).toContain(address.tab);
        expect(address.delegationsOpen).toBe(query.panel === 'delegations');
      }),
    );
  });

  it('reads back the tab and the preview it was given', () => {
    fc.assert(
      fc.property(fc.constantFrom(...TABS), fc.string({ minLength: 1 }), (tab, activity) => {
        const address = approvalsAddressOf(new URLSearchParams({ tab, activity }));
        expect(address).toEqual({ tab, previewId: activity, delegationsOpen: false });
      }),
    );
  });
});
