import { describe, expect, it } from 'vitest';
import { allows, gateItem } from './gate-action';

const item = { label: 'Edit', onSelect: () => undefined };

describe('gateItem and allows (014 FR-011)', () => {
  it('keeps an available action, or one from an API without access', () => {
    expect(gateItem(item, { available: true, reason: '' })).toEqual([item]);
    expect(gateItem(item, undefined)).toEqual([item]);
    expect(allows(undefined)).toBe(true);
    expect(allows({ available: true, reason: '' })).toBe(true);
  });

  it('disables an unavailable action with the server reason as its hint', () => {
    const reason = "Editing plans isn't part of your role (Viewer).";
    expect(gateItem(item, { available: false, reason })).toEqual([
      { ...item, disabled: true, hint: reason },
    ]);
    expect(allows({ available: false, reason })).toBe(false);
  });

  it('leaves out an unavailable action without a reason', () => {
    expect(gateItem(item, { available: false, reason: '' })).toEqual([]);
  });
});
