import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

/**
 * The security headers of every web page (038 FR-015, D-355, research R7): one source for the Vite
 * preview (enforced), the Vite dev server (report only) and nginx.conf, which a test keeps equal.
 */

/** The sha256 of index.html's one inline script, the theme script, exactly as the browser hashes it. */
function themeScriptHash(): string {
  const html = readFileSync(join(import.meta.dirname, 'index.html'), 'utf8');
  const script = /<script>([\s\S]*?)<\/script>/.exec(html)?.[1];
  if (script === undefined) {
    throw new Error('index.html has no inline theme script to hash');
  }
  return createHash('sha256').update(script).digest('base64');
}

const policy = [
  "default-src 'self'",
  `script-src 'self' 'sha256-${themeScriptHash()}'`,
  "style-src 'self'",
  "img-src 'self' data:",
  "font-src 'self'",
  "connect-src 'self'",
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
  "frame-ancestors 'none'",
].join('; ');

const others = {
  'X-Frame-Options': 'DENY',
  'Referrer-Policy': 'same-origin',
  'X-Content-Type-Options': 'nosniff',
};

export const securityHeaders: Record<string, string> = {
  'Content-Security-Policy': policy,
  ...others,
};

/** Vite's dev runtime injects its own inline preamble, so the dev server only reports violations. */
export const developmentHeaders: Record<string, string> = {
  'Content-Security-Policy-Report-Only': policy,
  ...others,
};
