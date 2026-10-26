import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { focusOf, PLAN_TABS, planTabOf } from './schemas';

const anyValue = fc.option(fc.string(), { nil: undefined });

function search(name: string, value: string | undefined): URLSearchParams {
  return new URLSearchParams(value === undefined ? {} : { [name]: value });
}

describe('the plan page tab in the address (design system §2.4 B)', () => {
  it('never fails on any ?tab=, and names a tab of the page', () => {
    fc.assert(
      fc.property(anyValue, (value) => {
        expect(PLAN_TABS).toContain(planTabOf(search('tab', value)));
      }),
    );
  });

  it.each(PLAN_TABS)('reads ?tab=%s', (tab) => {
    expect(planTabOf(search('tab', tab))).toBe(tab);
  });
});

describe('the field to focus from a completeness link (?focus=)', () => {
  it('never fails, and gives the field as written or null for none', () => {
    fc.assert(
      fc.property(anyValue, (value) => {
        expect(focusOf(search('focus', value))).toBe(value ? value : null);
      }),
    );
  });
});
