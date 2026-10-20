import { describe, expect, it } from 'vitest';
import { PARTLY_APPROVED, statusWord } from './statusWords';

describe('statusWord (design system §4.1, functional review A-02, UI audit X-14)', () => {
  it('writes the approval roll-up "In progress" (amber) as "Partly approved"', () => {
    expect(statusWord('In progress', 'review')).toBe(PARTLY_APPROVED);
  });

  it('keeps "In progress" for a running activity (blue) and every other word', () => {
    expect(statusWord('In progress', 'active')).toBe('In progress');
    expect(statusWord('Submitted', 'review')).toBe('Submitted');
    expect(statusWord('All approved', 'positive')).toBe('All approved');
    expect(statusWord('In progress', undefined)).toBe('In progress');
  });
});
