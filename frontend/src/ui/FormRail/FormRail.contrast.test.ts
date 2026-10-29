import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';

const railCss = readFileSync(join(import.meta.dirname, 'FormRail.module.css'), 'utf8');
const tokensCss = readFileSync(join(import.meta.dirname, '..', 'tokens.css'), 'utf8');

/** A token's value in the block of `selector` of tokens.css, following var() references. */
function token(name: string, selector: string): string {
  const start = tokensCss.indexOf(`${selector} {`);
  const block = tokensCss.slice(start, tokensCss.indexOf('}', start));
  const value = new RegExp(`${name}:\\s*([^;]+);`).exec(block)?.[1]?.trim();
  if (value === undefined) {
    return selector === ':root' ? '' : token(name, ':root');
  }
  const reference = /^var\((--[\w-]+)\)$/.exec(value)?.[1];
  return reference ? token(reference, selector) : value;
}

function luminance(hex: string): number {
  const channels = [1, 3, 5].map((at) => Number.parseInt(hex.slice(at, at + 2), 16) / 255);
  const [r = 0, g = 0, b = 0] = channels.map((c) =>
    c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4,
  );
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrast(a: string, b: string): number {
  const [light = 0, dark = 0] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (light + 0.05) / (dark + 0.05);
}

describe('FormRail on the grey page', () => {
  it.each([':root', ":root[data-theme='dark']"])(
    '038 FR-038 (M-17): the count above the form reaches 4.5:1 on the page grey (%s)',
    (theme) => {
      const rule = /\.progress\s*\{([^}]*)\}/.exec(railCss)?.[1] ?? '';
      const colour = /color:\s*var\((--[\w-]+)\)/.exec(rule)?.[1] ?? '';
      expect(contrast(token(colour, theme), token('--page-bg', theme))).toBeGreaterThanOrEqual(4.5);
    },
  );
});
