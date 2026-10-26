import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { PublishBlockedDialog } from './PublishBlockedDialog';

describe('PublishBlockedDialog (FR-024)', () => {
  it('says how many fields publishing needs and links each one', async () => {
    const onClose = vi.fn();
    const { user } = renderWithProviders(
      <PublishBlockedDialog
        open
        planId="P-1002"
        missing={[
          {
            record: { type: 'plan', id: 'P-1002', name: 'FY2026 Mobility China' },
            fields: [{ field: 'attributes.SA-BU', label: 'Business unit' }],
          },
        ]}
        onClose={onClose}
      />,
    );

    const dialog = screen.getByRole('dialog', { name: 'Publish needs 1 field' });
    expect(within(dialog).getByRole('heading', { level: 3, name: 'Plan' })).toBeInTheDocument();
    expect(within(dialog).getByRole('link', { name: 'Business unit' })).toHaveAttribute(
      'href',
      '/plans/P-1002/edit?focus=attributes.SA-BU',
    );
    expect(within(dialog).queryByRole('button', { name: 'Cancel' })).not.toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: 'Close' }));
    expect(onClose).toHaveBeenCalled();
  });
});
