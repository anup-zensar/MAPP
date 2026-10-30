import { expect, test as base } from '@playwright/test';

/** Fails on any `securitypolicyviolation` (038 FR-015); specs import `test` and `expect` from here. */
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
