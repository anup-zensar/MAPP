import { describe, expect, it } from 'vitest';
import { monthLabel } from './dashboard-text';

describe('dashboard texts', () => {
  it('038 FR-058: writes September as "Sep", and in the person\'s date format', () => {
    expect(monthLabel('2026-09')).toBe('Sep 2026');
    expect(monthLabel('2026-09', 'YYYY-MM-DD')).toBe('2026-09');
  });
});
