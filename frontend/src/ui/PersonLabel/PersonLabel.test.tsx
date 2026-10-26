import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { PersonLabel } from './PersonLabel';

const PRIYA = { name: 'Priya Raman', initials: 'PR', active: true, jobTitle: 'Marketer' };

describe('PersonLabel', () => {
  it('names the person in full with their initials beside, hidden from screen readers', () => {
    const { container } = render(<PersonLabel person={PRIYA} detail="Marketer" />);

    expect(screen.getByText('Priya Raman')).toBeInTheDocument();
    expect(screen.getByText('Marketer')).toBeInTheDocument();
    expect(container.querySelector('[aria-hidden="true"]')).toHaveTextContent('PR');
  });

  it('writes " (you)" and " (inactive)" out, never by colour alone (SC-003)', () => {
    render(<PersonLabel person={{ ...PRIYA, active: false }} you />);

    expect(screen.getByText('Priya Raman (you) (inactive)')).toBeInTheDocument();
  });
});
