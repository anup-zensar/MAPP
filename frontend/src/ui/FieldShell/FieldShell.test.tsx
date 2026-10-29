import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { DateRangeField } from '../DateRangeField/DateRangeField';
import { MoneyField } from '../MoneyField/MoneyField';
import { TextAreaField } from '../TextAreaField/TextAreaField';
import { TextField } from '../TextField/TextField';

const ignore = () => {};
const shellCss = readFileSync(join(import.meta.dirname, 'FieldShell.module.css'), 'utf8');
const tokensCss = readFileSync(join(import.meta.dirname, '..', 'tokens.css'), 'utf8');

afterEach(() => {
  delete document.documentElement.dataset.theme;
});

/** The fields of a short form: two required to save, one needed to publish. */
function Fields() {
  return (
    <form aria-label="New plan">
      <TextField label="Plan name" name="name" value="" onChange={ignore} marking="Required" />
      <MoneyField
        label="Budget"
        name="budget"
        currency="USD"
        value=""
        onChange={ignore}
        marking="Required"
        hint="In US dollars."
      />
      <TextAreaField
        label="Objective"
        name="objective"
        value=""
        onChange={ignore}
        marking="Needed to publish"
      />
      <DateRangeField
        idPrefix="plan"
        label="Dates"
        start=""
        end=""
        onChange={ignore}
        marking="Required"
      />
    </form>
  );
}

describe('FieldShell marking (038 FR-063, D-358)', () => {
  it('038 FR-063: a field required to save shows a red asterisk after its label instead of the word "Required"', () => {
    render(<Fields />);

    expect(screen.queryByText('Required', { exact: true })).not.toBeInTheDocument();
    for (const label of ['Plan name', 'Budget']) {
      const row = screen.getByText(label, { selector: 'label' }).parentElement;
      const asterisk = within(row ?? document.body).getByText('*');
      expect(asterisk).toHaveAttribute('aria-hidden', 'true');
    }
    const dates = screen.getByRole('group', { name: 'Dates' });
    expect(within(dates).getAllByText('*')[0]).toHaveAttribute('aria-hidden', 'true');
  });

  it('038 FR-063: the control is aria-required and its description says "required" once', () => {
    render(<Fields />);

    const name = screen.getByRole('textbox', { name: 'Plan name' });
    expect(name).toHaveAttribute('aria-required', 'true');
    expect(name).toHaveAccessibleDescription('required');
    const budget = screen.getByRole('textbox', { name: 'Budget' });
    expect(budget).toHaveAttribute('aria-required', 'true');
    expect(budget.getAttribute('aria-describedby')).toBeTruthy();
    expect(countRequired(budget)).toBe(1);
    expect(screen.getByLabelText('Start date')).toHaveAttribute('aria-required', 'true');
    expect(screen.getByLabelText('End date')).toHaveAttribute('aria-required', 'true');
    expect(countRequired(screen.getByRole('group', { name: 'Dates' }))).toBe(1);
  });

  it('038 FR-063: "Needed to publish" keeps its word tag, with no asterisk and no aria-required', () => {
    render(<Fields />);

    const objective = screen.getByRole('textbox', { name: 'Objective' });
    expect(objective).not.toHaveAttribute('aria-required');
    expect(objective).toHaveAccessibleDescription(/^Needed to publish/);
    const row = screen.getByText('Objective', { selector: 'label' }).parentElement;
    expect(within(row ?? document.body).queryByText('*')).not.toBeInTheDocument();
    expect(within(row ?? document.body).getByText('Needed to publish')).toBeVisible();
  });

  it.each(['light', 'dark'])(
    '038 FR-063, SC-013: axe finds nothing on required fields in the %s theme',
    async (theme) => {
      document.documentElement.dataset.theme = theme;
      const { container } = render(<Fields />);
      await expectNoAxeViolations(container);
    },
  );

  it('038 FR-063: the asterisk is bold Secondary Red at 19 px or more, large text in both themes', () => {
    const rule = /\.asterisk\s*\{([^}]*)\}/.exec(shellCss)?.[1] ?? '';
    expect(rule).toMatch(/color:\s*var\(--danger\)/);
    expect(rule).toMatch(/font-weight:\s*var\(--weight-bold\)/);
    const size = /font-size:\s*var\((--[\w-]+)\)/.exec(rule)?.[1] ?? '';
    expect(Number.parseFloat(token(tokensCss, size, ':root'))).toBeGreaterThanOrEqual(19);
    for (const theme of [':root', ":root[data-theme='dark']"]) {
      const red = token(tokensCss, '--danger', theme);
      const surface = token(tokensCss, '--surface', theme);
      expect(contrast(red, surface)).toBeGreaterThanOrEqual(3);
    }
  });
});

/** How often "required" is said in an element's accessible description. */
function countRequired(element: HTMLElement): number {
  const ids = element.getAttribute('aria-describedby')?.split(' ') ?? [];
  const text = ids.map((id) => document.getElementById(id)?.textContent ?? '').join(' ');
  return text.match(/\brequired\b/gi)?.length ?? 0;
}

/** A token's value in the block of `selector` of tokens.css. */
function token(css: string, name: string, selector: string): string {
  const start = css.indexOf(`${selector} {`);
  const block = css.slice(start, css.indexOf('}', start));
  return new RegExp(`${name}:\\s*([^;]+);`).exec(block)?.[1]?.trim() ?? '';
}

function contrast(a: string, b: string): number {
  const [light, dark] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return ((light ?? 0) + 0.05) / ((dark ?? 0) + 0.05);
}

function luminance(hex: string): number {
  const channels = [1, 3, 5].map((at) => Number.parseInt(hex.slice(at, at + 2), 16) / 255);
  const [r = 0, g = 0, b = 0] = channels.map((c) =>
    c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4,
  );
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}
