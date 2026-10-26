/**
 * Runs before every test file (vite.config.ts → test.setupFiles).
 *
 * - Adds the jest-dom matchers such as toBeInTheDocument() and toHaveAccessibleName().
 * - Fills in the browser features jsdom does not have, which Radix and the theme code use.
 * - Gives every test an API without answers, so no test can ever reach a real server; a test that
 *   makes a request without a canned answer fails (research R19).
 */
import '@testing-library/jest-dom/vitest';
import { cleanup, configure } from '@testing-library/react';
import { afterEach, beforeEach, vi } from 'vitest';
import { mockApi, takeUnmockedRequests } from './mock-api';

// findBy… and waitFor wait up to 3 seconds: on a busy laptop a page can need more than the default 1 s.
configure({ asyncUtilTimeout: 3000 });

// matchMedia answers "no" to every media query: light mode and no reduced motion. Tests that
// need the computer's dark setting replace it with vi.stubGlobal('matchMedia', …).
window.matchMedia = (query: string): MediaQueryList => ({
  matches: false,
  media: query,
  onchange: null,
  addEventListener: () => {},
  removeEventListener: () => {},
  addListener: () => {},
  removeListener: () => {},
  dispatchEvent: () => false,
});

// Radix measures its floating content (popovers, menus, tooltips) with ResizeObserver.
window.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
};

// Radix calls these on pointer and focus events; jsdom does not implement them.
Element.prototype.scrollIntoView = () => {};
Element.prototype.hasPointerCapture = () => false;
Element.prototype.releasePointerCapture = () => {};

// React Router's ScrollRestoration scrolls to the top after every navigation; jsdom cannot scroll.
window.scrollTo = () => {};

beforeEach(() => {
  mockApi({});
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();

  const unmocked = takeUnmockedRequests();
  if (unmocked.length > 0) {
    throw new Error(
      `The test made requests without a canned answer: ${unmocked.join(', ')}. ` +
        'Add them to mockApi({ … }).',
    );
  }
});
