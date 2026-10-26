import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { accessTabOf, copyOfParam, openPersonOf, withAccessTab, withOpenPerson } from './schemas';

const anyAddress = fc.string().map((query) => new URLSearchParams(query));
const tabs = fc.constantFrom('people' as const, 'roles' as const, 'rules' as const);

describe('the address of Roles and access (014 plan "Pages")', () => {
  it('reads any ?tab= as one of the three tabs, and a chosen tab back', () => {
    fc.assert(
      fc.property(anyAddress, tabs, (params, tab) => {
        expect(['people', 'roles', 'rules']).toContain(accessTabOf(params));
        expect(accessTabOf(withAccessTab(params, tab))).toBe(tab);
        expect(openPersonOf(withAccessTab(params, tab))).toBe('');
      }),
    );
  });

  it('reads any ?person= as text, and an opened person back', () => {
    fc.assert(
      fc.property(anyAddress, fc.string(), (params, personId) => {
        expect(typeof openPersonOf(params)).toBe('string');
        expect(openPersonOf(withOpenPerson(params, personId))).toBe(personId);
      }),
    );
  });

  it('reads any ?copyOf= as text, and the role a link names', () => {
    fc.assert(
      fc.property(anyAddress, fc.string(), (params, roleId) => {
        expect(typeof copyOfParam(params)).toBe('string');
        expect(copyOfParam(new URLSearchParams({ copyOf: roleId }))).toBe(roleId);
      }),
    );
  });
});
