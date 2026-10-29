import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { allWorkOptions, standard2026 } from '@/testing/fixtures/all-work';
import { BRANDS, COUNTRY } from '@/testing/fixtures/all-work-breakdown';
import type { AttributeFilter, ViewState } from './all-work-types';
import { attributeChipLabel, attrParams, knownAttributeFilters } from './attribute-filters';
import { chipsOf, removeChip } from './view-chips';
import { leftOutOf, sameState, toQueryParams } from './view-state';

const ATTR_PATTERN = /^[A-Z]+-[0-9]+:[A-Za-z0-9,-]+$/;

function withAttributes(attributes: AttributeFilter[]): ViewState {
  return { ...standard2026(), filters: { ...standard2026().filters, attributes } };
}

describe('016 FR-039: Shell-attribute filters', () => {
  it('asks the API with one attr parameter per attribute, Not filled in as none', () => {
    const state = withAttributes([
      { attributeId: 'SA-1020', values: ['SO-1201', 'none'] },
      { attributeId: 'SA-1013', values: ['SO-1101'] },
    ]);

    expect(toQueryParams(state).attr).toEqual(['SA-1020:SO-1201,none', 'SA-1013:SO-1101']);
    expect(toQueryParams(standard2026())).not.toHaveProperty('attr');
  });

  it('shows one chip per attribute, its values in Settings’ order and Not filled in last; removing it clears the attribute', () => {
    const state = withAttributes([{ attributeId: 'SA-1020', values: ['none', 'SO-1201'] }]);
    const options = allWorkOptions({ attributes: [BRANDS] });

    expect(chipsOf(state, options)).toEqual([
      { key: 'attributes:SA-1020', label: 'Brands: Helix, Not filled in' },
    ]);
    expect(removeChip(state, 'attributes:SA-1020').filters.attributes).toEqual([]);
  });

  it('names an attribute by its id while the filter bar’s choices are loading', () => {
    expect(attributeChipLabel({ attributeId: 'SA-1020', values: ['yes'] }, undefined)).toBe(
      'SA-1020: yes',
    );
  });

  it('leaves out of a saved view the attributes and options that no longer exist, and says so', () => {
    const state = withAttributes([
      { attributeId: 'SA-1020', values: ['SO-1201', 'SO-1999', 'none'] },
      { attributeId: 'SA-1999', values: ['yes'] },
    ]);

    const { state: kept, names } = leftOutOf(state, allWorkOptions({ attributes: [BRANDS] }));

    expect(kept.filters.attributes).toEqual([
      { attributeId: 'SA-1020', values: ['SO-1201', 'none'] },
    ]);
    expect(names).toEqual(['Brands: SO-1999', 'Shell attribute: SA-1999']);
  });

  it('keeps every attribute filter when the API does not list attributes', () => {
    const filters = [{ attributeId: 'SA-1020', values: ['SO-1201'] }];
    expect(knownAttributeFilters(filters, undefined)).toEqual({ filters, names: [] });
  });

  it('writes every filter as the contract’s attr pattern, and the order of filters never makes a view Modified', () => {
    const value = fc.constantFrom('SO-1201', 'SO-1202', 'SO-1101', 'yes', 'no', 'none');
    const filter = fc.record({
      attributeId: fc.constantFrom(BRANDS.id, COUNTRY.id, 'SA-7'),
      values: fc.uniqueArray(value, { minLength: 1 }),
    });
    const filters = fc.uniqueArray(filter, { selector: (f) => f.attributeId });

    fc.assert(
      fc.property(filters, (chosen) => {
        expect(attrParams(chosen).every((param) => ATTR_PATTERN.test(param))).toBe(true);
        const shuffled = chosen.map((f) => ({ ...f, values: [...f.values].reverse() })).reverse();
        expect(sameState(withAttributes(chosen), withAttributes(shuffled))).toBe(true);
      }),
    );
  });
});
