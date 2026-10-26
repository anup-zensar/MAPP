/**
 * axe in component tests (test-only helper of Settings, People and Roles and access): the screens
 * whose API is not served yet are checked against WCAG 2.2 A and AA here, with the tags of the
 * end-to-end check (e2e/fixtures.ts). jsdom draws nothing, so colour contrast is left to the
 * browser checks.
 */
import axe from 'axe-core';
import { expect } from 'vitest';

export async function expectNoAxeViolations(container: Element = document.body): Promise<void> {
  const results = await axe.run(container, {
    runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'] },
    rules: { 'color-contrast': { enabled: false } },
  });
  const summary = results.violations.map(
    (violation) =>
      `${violation.id}: ${violation.nodes.map((node) => node.target.join(' ')).join(', ')}`,
  );
  expect(summary).toEqual([]);
}
