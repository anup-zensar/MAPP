import { describe, expect, it } from 'vitest';
import { VIEWER } from '@/testing/fixtures/viewer';
import { accessOf } from './access';
import type { MyAccess } from './access';
import { refusalText } from './refusal-text';

function accessWithRoles(names: string[]): MyAccess {
  const base = VIEWER.access;
  const [viewer] = base?.roles ?? [];
  if (!base || !viewer) {
    throw new Error('VIEWER holds the Viewer role');
  }
  const roles = names.map((name, i) => ({
    ...viewer,
    id: `RA-${i}`,
    role: { ...viewer.role, name },
  }));
  return { ...base, roles };
}

describe('014 FR-016: the reason of an action no role of the person includes', () => {
  it.each([
    [[], "Exporting needs a role, and you don't have one yet. Ask an administrator."],
    [
      ['Viewer'],
      "Exporting isn't part of your role (Viewer). Ask an administrator if you need it.",
    ],
    [
      ['Approver', 'Planner'],
      "Exporting isn't part of your roles (Approver, Planner). Ask an administrator if you need it.",
    ],
  ])('roles %j', (names, text) => {
    expect(refusalText('Exporting', accessOf(accessWithRoles(names)))).toBe(text);
  });
});
