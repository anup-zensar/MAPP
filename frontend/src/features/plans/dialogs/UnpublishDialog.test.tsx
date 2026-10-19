import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { UnpublishDialog } from './UnpublishDialog';

describe('UnpublishDialog (FR-025)', () => {
  it('sends the typed reason and shows the problem the API answers', async () => {
    const onUnpublish = vi.fn(async (reason: string) => (reason ? null : 'Enter a reason.'));
    const { user } = renderWithProviders(
      <UnpublishDialog
        open
        planName="FY2025 Lubricants Europe"
        onUnpublish={onUnpublish}
        onClose={() => {}}
      />,
    );

    expect(
      screen.getByRole('dialog', { name: "Unpublish plan 'FY2025 Lubricants Europe'?" }),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Unpublish plan' }));
    expect(await screen.findByText('Enter a reason.')).toBeInTheDocument();

    await user.type(screen.getByRole('textbox', { name: /Why are you unpublishing/ }), 'Rework');
    await user.click(screen.getByRole('button', { name: 'Unpublish plan' }));

    expect(onUnpublish).toHaveBeenLastCalledWith('Rework');
    expect(screen.queryByText('Enter a reason.')).not.toBeInTheDocument();
  });

  it('closes with Cancel', async () => {
    const onClose = vi.fn();
    const { user } = renderWithProviders(
      <UnpublishDialog open planName="P" onUnpublish={async () => null} onClose={onClose} />,
    );

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalled();
  });
});
