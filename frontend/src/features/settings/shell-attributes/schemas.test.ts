import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { withChoice } from '../search-params';
import { LEVELS } from './attribute-words';
import { attributeChoicesOf } from './schemas';

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
