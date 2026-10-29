import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { developmentHeaders, securityHeaders } from './security-headers';

const indexHtml = readFileSync(join(import.meta.dirname, 'index.html'), 'utf8');
const nginxConf = readFileSync(join(import.meta.dirname, 'nginx.conf'), 'utf8');

/** The inline scripts of index.html (those without a src). */
function inlineScripts(): string[] {
  return [...indexHtml.matchAll(/<script>([\s\S]*?)<\/script>/g)].map((match) => match[1] ?? '');
}

/** Each nginx `location` block that serves the web app (not the API, which sends its own) by path. */
function webLocations(): Map<string, Map<string, string>> {
  const locations = new Map<string, Map<string, string>>();
  for (const block of nginxConf.matchAll(/location\s+(\S+)\s*\{([\s\S]*?)\n\s*\}/g)) {
    const [, path = '', body = ''] = block;
    if (path === '/api/') {
      continue;
    }
    const headers = new Map<string, string>();
    for (const line of body.matchAll(/add_header\s+(\S+)\s+"([^"]*)"\s+always;/g)) {
      headers.set(line[1] ?? '', line[2] ?? '');
    }
    headers.delete('Cache-Control');
    locations.set(path, headers);
  }
  return locations;
}

describe('security headers of the web app (038 FR-015, D-355)', () => {
  it('038 FR-015: admits the one inline theme script by its sha256 and nothing else inline', () => {
    const scripts = inlineScripts();
    expect(scripts).toHaveLength(1);
    const hash = createHash('sha256')
      .update(scripts[0] ?? '')
      .digest('base64');

    const policy = securityHeaders['Content-Security-Policy'];
    expect(policy).toContain(`script-src 'self' 'sha256-${hash}';`);
    expect(policy).not.toMatch(/script-src[^;]*'unsafe-(inline|eval)'/);
    expect(policy).toContain("frame-ancestors 'none'");
  });

  it("038 FR-015, ADR-0018: admits Radix's scroll-lock <style> element, never inline style attributes", () => {
    const policy = securityHeaders['Content-Security-Policy'];
    expect(policy).toContain("style-src 'self';");
    expect(policy).toContain("style-src-elem 'self' 'unsafe-inline';");
  });

  it('038 FR-015: forbids framing, foreign referrers and content sniffing', () => {
    expect(securityHeaders).toMatchObject({
      'X-Frame-Options': 'DENY',
      'Referrer-Policy': 'same-origin',
      'X-Content-Type-Options': 'nosniff',
    });
  });

  it('038 FR-015: every web location of nginx.conf sends exactly the same headers, also on errors', () => {
    const locations = webLocations();
    expect([...locations.keys()]).toEqual(['/assets/', '/']);
    for (const headers of locations.values()) {
      expect(Object.fromEntries(headers)).toEqual(securityHeaders);
    }
  });

  it('038 FR-011: nginx accepts an upload of the documented 8 MB maximum and forwards the client address', () => {
    const api = /location \/api\/ \{([\s\S]*?)\n\s*\}/.exec(nginxConf)?.[1] ?? '';
    expect(api).toContain('client_max_body_size 8m;');
    expect(api).toContain('proxy_set_header X-Forwarded-For $remote_addr;');
  });

  it('038 FR-015: the development server only reports, since Vite injects its own inline preamble', () => {
    expect(developmentHeaders['Content-Security-Policy-Report-Only']).toBe(
      securityHeaders['Content-Security-Policy'],
    );
    expect(developmentHeaders).not.toHaveProperty('Content-Security-Policy');
    expect(developmentHeaders['X-Frame-Options']).toBe('DENY');
  });
});
