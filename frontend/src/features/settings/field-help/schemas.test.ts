import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { HELP_LEVELS } from './field-help-words';
import { helpChoicesOf, withChoice } from './schemas';

const SHOWN = ['all', 'edited'] as const;

describe('the Field help choices in the address (016 FR-045)', () => {
  it('never fail on any address, and give a known level and Show choice', () => {
    fc.assert(
      fc.property(fc.string(), fc.string(), fc.string(), (query, level, show) => {
        const params = new URLSearchParams(query);
        params.set('level', level);
        params.set('show', show);
        const choices = helpChoicesOf(params);
        expect(HELP_LEVELS).toContain(choices.level);
        expect(SHOWN).toContain(choices.shown);
      }),
    );
  });

  it('read back every choice the page writes', () => {
    fc.assert(
      fc.property(
        fc.constantFrom(...HELP_LEVELS),
        fc.constantFrom(...SHOWN),
        fc.string(),
        fc.string(),
        (level, shown, q, openKey) => {
          let params = new URLSearchParams();
          params = withChoice(params, 'level', level, 'plan');
          params = withChoice(params, 'show', shown, 'all');
          params = withChoice(params, 'q', q, '');
          params = withChoice(params, 'key', openKey, '');
          expect(helpChoicesOf(params)).toEqual({ level, shown, q, openKey });
        },
      ),
    );
  });
});
