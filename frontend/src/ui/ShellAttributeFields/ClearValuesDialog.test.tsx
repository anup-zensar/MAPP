import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { ClearValuesDialog } from './ClearValuesDialog';

describe('ClearValuesDialog', () => {
  it('lists the values a save would clear and asks first (FR-025)', async () => {
    const onConfirm = vi.fn();
    const onOpenChange = vi.fn();
    const { user } = renderWithProviders(
      <ClearValuesDialog
        open
        onOpenChange={onOpenChange}
        onConfirm={onConfirm}
        cleared={[
          {
            attributeId: 'SA-REGION',
            label: 'Region',
            optionIds: ['SO-EU', 'SO-ASIA'],
            optionLabels: ['B2B – EUROPE', 'B2B – ASIA'],
          },
        ]}
      />,
    );

    const dialog = screen.getByRole('alertdialog', { name: 'Clear values that no longer fit?' });
    expect(dialog).toHaveTextContent('Region: B2B – EUROPE, B2B – ASIA');
    expect(screen.getByRole('button', { name: 'Keep editing' })).toHaveFocus();

    await user.click(screen.getByRole('button', { name: 'Clear and save' }));
    expect(onConfirm).toHaveBeenCalledOnce();
  });
});
