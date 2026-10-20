import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { Switch } from './Switch';

describe('Switch', () => {
  it('is a switch named by its words, on or off (FR-012)', async () => {
    const onChange = vi.fn();
    render(<Switch label="Unassigned planned cost" checked onChange={onChange} />);

    const control = screen.getByRole('switch', { name: 'Unassigned planned cost' });
    expect(control).toHaveAttribute('aria-checked', 'true');
    await userEvent.click(control);
    expect(onChange).toHaveBeenCalledWith(false);
  });

  it('keeps the whole name when only the short words show (WCAG 2.5.3)', () => {
    render(
      <>
        <Switch
          label="Unassigned planned cost"
          shortLabel="Unassigned"
          checked={false}
          onChange={() => {}}
        />
        <Switch
          label="Compare with phased budget"
          shortLabel="Compare"
          shortOn="phone"
          checked={false}
          onChange={() => {}}
        />
      </>,
    );

    const unassigned = screen.getByRole('switch', { name: 'Unassigned planned cost' });
    expect(unassigned.querySelector('[class*="rest"]')).toHaveTextContent('planned cost');
    // The Compare switch shortens on phones only, not in a crowded toolbar.
    const compare = screen.getByRole('switch', { name: 'Compare with phased budget' });
    expect(compare.querySelector('[class*="restOnPhones"]')).toHaveTextContent(
      'with phased budget',
    );
  });
});
