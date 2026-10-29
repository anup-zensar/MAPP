import { expect, test as base } from '@playwright/test';

/**
 * A `test` that fails when the page breaks its content security policy (038 FR-015, SC-005): every
 * `securitypolicyviolation` the browser raises during the test is collected and must be none.
 * Specs import `test` and `expect` from here instead of from '@playwright/test'.
 */
export const test = base.extend<{ cspViolations: string[] }>({
  cspViolations: [
    async ({ page }, use) => {
      const violations: string[] = [];
      await page.exposeFunction('__reportCspViolation', (text: string) => violations.push(text));
      await page.addInitScript(() => {
        document.addEventListener('securitypolicyviolation', (event) => {
          const report: unknown = Reflect.get(window, '__reportCspViolation');
          if (typeof report === 'function') {
            report(`${event.effectiveDirective} blocked ${event.blockedURI || 'inline'}`);
          }
        });
      });
      await use(violations);
      expect(violations, 'content security policy violations').toEqual([]);
    },
    { auto: true },
  ],
});

export { expect };
