import { describe, expect, it } from 'vitest';
import { joinWithAnd, midSentence, plural } from './text';

describe('midSentence', () => {
  it.each([
    ['Region', 'region'],
    ['Business unit', 'business unit'],
    ['GL code', 'GL code'],
    ['SMP', 'SMP'],
    ['A-team', 'A-team'],
    ['X', 'X'],
    ['', ''],
  ])('writes "%s" as "%s" inside a sentence', (label, expected) => {
    expect(midSentence(label)).toBe(expected);
  });
});

describe('plural', () => {
  it('uses the singular for exactly one and groups large counts', () => {
    expect(plural(1, 'model')).toBe('1 model');
    expect(plural(0, 'model')).toBe('0 models');
    expect(plural(1217, 'option')).toBe('1,217 options');
    expect(plural(2, 'activity', 'activities')).toBe('2 activities');
    expect(plural(1n, 'person', 'people')).toBe('1 person');
  });
});

describe('joinWithAnd', () => {
  it('joins with commas and a final "and"', () => {
    expect(joinWithAnd([])).toBe('');
    expect(joinWithAnd(['Lubricants'])).toBe('Lubricants');
    expect(joinWithAnd(['Lubricants', 'Mobility'])).toBe('Lubricants and Mobility');
    expect(joinWithAnd(['2 initiatives', '7 activities', '15 line items'])).toBe(
      '2 initiatives, 7 activities and 15 line items',
    );
  });
});
