import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { noticeOf } from './schemas';

describe('the sign-in notice from the router state (001 FR-006)', () => {
  it('never fails on any state, and names only an ended session', () => {
    fc.assert(
      fc.property(fc.anything(), (state) => {
        expect([null, 'signedOut', 'sessionLost']).toContain(noticeOf(state));
      }),
    );
  });

  it.each(['signedOut', 'sessionLost'] as const)('reads %s', (notice) => {
    expect(noticeOf({ notice })).toBe(notice);
  });
});
