import { describe, expect, it } from 'vitest';
import { cx } from './cx';

describe('cx', () => {
  it('joins class names with spaces', () => {
    expect(cx('button', 'primary')).toBe('button primary');
  });

  it('leaves out false, null and undefined, so a class can depend on a condition', () => {
    const isActive = false;
    expect(cx('item', isActive && 'active', null, undefined)).toBe('item');
  });
});
