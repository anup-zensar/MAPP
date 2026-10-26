import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { filtersToParams } from './log-filters';
import { filtersFromParams } from './schemas';

const NAMES = ['person', 'type', 'action', 'plan', 'record', 'inside', 'range', 'from', 'to', 'q'];
const VALUES = ['custom', 'week', '1', 'deleted', 'moved', '2026-10-01', '2026-13-45', ''];

const anyAddress = fc
  .array(
    fc.tuple(fc.constantFrom(...NAMES, 'other'), fc.oneof(fc.constantFrom(...VALUES), fc.string())),
  )
  .map((pairs) => new URLSearchParams(pairs));

describe('the activity log’s address (011 FR-003)', () => {
  it('reads any address without failing, into filters that survive write-then-read', () => {
    fc.assert(
      fc.property(anyAddress, (address) => {
        const filters = filtersFromParams(address);
        const written = new URLSearchParams(filtersToParams(filters).toString());

        expect(filtersFromParams(written)).toEqual(filters);
      }),
    );
  });
});
