import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { Breadcrumb } from './Breadcrumb';

describe('Breadcrumb', () => {
  it('links every step but the current page, which is marked (design system §6.3)', () => {
    renderWithProviders(
      <Breadcrumb
        items={[
          { label: 'Plans', to: '/plans' },
          { label: 'FY2026 Lubricants Europe', to: '/plans/P-1001' },
          { label: 'Helix launch' },
        ]}
      />,
    );

    const trail = screen.getByRole('navigation', { name: 'Breadcrumb' });
    expect(within(trail).getByRole('link', { name: 'Plans' })).toHaveAttribute('href', '/plans');
    expect(within(trail).getByRole('link', { name: 'FY2026 Lubricants Europe' })).toHaveAttribute(
      'href',
      '/plans/P-1001',
    );
    expect(within(trail).queryByRole('link', { name: 'Helix launch' })).not.toBeInTheDocument();
    expect(within(trail).getByText('Helix launch')).toHaveAttribute('aria-current', 'page');
    expect(trail).toHaveTextContent('Plans›FY2026 Lubricants Europe›Helix launch');
  });

  it('never links the last item, even when it has an address', () => {
    renderWithProviders(
      <Breadcrumb
        items={[
          { label: 'Settings', to: '/settings' },
          { label: 'Models', to: '/settings/models' },
        ]}
      />,
    );

    expect(screen.queryByRole('link', { name: 'Models' })).not.toBeInTheDocument();
    expect(screen.getByText('Models')).toHaveAttribute('aria-current', 'page');
  });
});
