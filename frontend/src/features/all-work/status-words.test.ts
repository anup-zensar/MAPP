import { describe, expect, it } from 'vitest';
import { statusWord } from './status-words';

describe('statusWord (design system §4.1, functional review A-02, UI audit X-14)', () => {
  it('038 FR-033 (M-11): shows every status word as the API sends it, "Partly approved" included', () => {
    expect(statusWord('Partly approved')).toBe('Partly approved');
    expect(statusWord('In progress')).toBe('In progress');
    expect(statusWord('All approved')).toBe('All approved');
  });
});
