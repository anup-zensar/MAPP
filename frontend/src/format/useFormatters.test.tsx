import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { useFormatters } from './useFormatters';

/** Shows what the hook gives, one line each. */
function Shown() {
  const format = useFormatters();
  return (
    <ul>
      <li>format: {format.dateFormat}</li>
      <li>currency: {format.displayCurrency}</li>
      <li>date: {format.date('2026-03-12')}</li>
      <li>dateTime: {format.dateTime('2026-03-12T14:32:00Z')}</li>
      <li>range: {format.dateRange('2026-02-01', '2026-11-30')}</li>
      <li>parsed: {format.parseDate('12/03/2026') ?? 'null'}</li>
      <li>money: {format.money({ amount: '12500.00', currency: 'EUR' })}</li>
      <li>compact: {format.moneyCompact({ amount: '1250000.00', currency: 'USD' })}</li>
      <li>none: {format.money(null)}</li>
    </ul>
  );
}

describe('useFormatters', () => {
  it('uses the defaults when nobody is signed in (FR-024)', () => {
    renderWithProviders(<Shown />);

    expect(screen.getByText('format: DD MMM YYYY')).toBeInTheDocument();
    expect(screen.getByText('currency: USD')).toBeInTheDocument();
    expect(screen.getByText('date: 12 Mar 2026')).toBeInTheDocument();
    expect(screen.getByText('range: 01 Feb – 30 Nov 2026')).toBeInTheDocument();
    expect(screen.getByText('parsed: null')).toBeInTheDocument();
  });

  it('shows and reads dates in the person’s format (FR-024, SC-006)', () => {
    const user = testUser({
      preferences: { ...testUser().preferences, dateFormat: 'DD/MM/YYYY', displayCurrency: 'EUR' },
    });
    renderWithProviders(<Shown />, { user });

    expect(screen.getByText('format: DD/MM/YYYY')).toBeInTheDocument();
    expect(screen.getByText('currency: EUR')).toBeInTheDocument();
    expect(screen.getByText('date: 12/03/2026')).toBeInTheDocument();
    expect(screen.getByText('dateTime: 12/03/2026, 14:32')).toBeInTheDocument();
    expect(screen.getByText('range: 01/02/2026 – 30/11/2026')).toBeInTheDocument();
    expect(screen.getByText('parsed: 2026-03-12')).toBeInTheDocument();
  });

  it('formats amounts exactly as the API answered them, never converting (P6)', () => {
    renderWithProviders(<Shown />, { user: testUser() });

    expect(screen.getByText('money: 12,500.00 EUR')).toBeInTheDocument();
    expect(screen.getByText('compact: $1.25M')).toBeInTheDocument();
    expect(screen.getByText('none: —')).toBeInTheDocument();
  });
});
