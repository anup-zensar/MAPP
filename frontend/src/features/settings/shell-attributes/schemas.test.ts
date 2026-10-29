import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { withChoice } from '../search-params';
import { LEVELS } from './attribute-words';
import { attributeChoicesOf, attributeProblems } from './schemas';
import type { AttributeValues } from './schemas';

describe('038 FR-030 (M-12): the Shell attribute form finds every problem at the first Save', () => {
  const VALUES: AttributeValues = {
    label: ' channel ',
    helpText: 'x'.repeat(301),
    level: 'activity',
    type: 'singleChoice',
    marking: '',
    section: '',
    parentAttributeId: '',
    conditionOptionIds: [],
    options: [],
    optionLinks: {},
  };
  const OTHERS = [
    { id: 'SA-1001', label: 'Channel', level: 'activity' as const },
    { id: 'SA-1002', label: 'Audience', level: 'plan' as const },
  ];

  it('names a taken label, a long help text, the marking and the options at once', () => {
    expect(attributeProblems(VALUES, { isNew: true, optionsToLink: null, others: OTHERS })).toEqual(
      [
        {
          field: 'label',
          message: "The Activity form already has a Shell attribute called 'Channel'.",
        },
        { field: 'helpText', message: 'Use 300 characters or fewer.' },
        { field: 'marking', message: 'Choose a marking.' },
        { field: 'options', message: 'Add at least one option.' },
      ],
    );
  });

  it('lets an attribute keep its own label, and a label of another level', () => {
    const values = { ...VALUES, helpText: '', marking: 'optional', type: 'text' };
    const rules = { isNew: false, optionsToLink: null, others: OTHERS, ownId: 'SA-1001' };
    expect(attributeProblems(values, rules)).toEqual([]);
    expect(
      attributeProblems({ ...values, label: 'Audience' }, { ...rules, ownId: undefined }),
    ).toEqual([]);
  });
});

const STATUSES = ['active', 'inactive', 'all'] as const;
const HELP = ['all', 'missing'] as const;

describe('the Shell attributes choices in the address (002 FR-016, 016 FR-048)', () => {
  it('never fail on any address, and give a known level, status and help filter', () => {
    fc.assert(
      fc.property(
        fc.string(),
        fc.string(),
        fc.string(),
        fc.string(),
        (query, level, status, help) => {
          const params = new URLSearchParams(query);
          params.set('level', level);
          params.set('status', status);
          params.set('help', help);
          const choices = attributeChoicesOf(params);
          expect(LEVELS).toContain(choices.level);
          expect(STATUSES).toContain(choices.status);
          expect(HELP).toContain(choices.help);
        },
      ),
    );
  });

  it('read back every choice the list writes', () => {
    fc.assert(
      fc.property(
        fc.constantFrom(...LEVELS),
        fc.constantFrom(...STATUSES),
        fc.string(),
        fc.constantFrom(...HELP),
        (level, status, q, help) => {
          let params = new URLSearchParams();
          params = withChoice(params, 'level', level, 'plan');
          params = withChoice(params, 'status', status, 'active');
          params = withChoice(params, 'q', q, '');
          params = withChoice(params, 'help', help, 'all');
          expect(attributeChoicesOf(params)).toEqual({
            level,
            levelGiven: level !== 'plan',
            status,
            q,
            help,
          });
        },
      ),
    );
  });
});
