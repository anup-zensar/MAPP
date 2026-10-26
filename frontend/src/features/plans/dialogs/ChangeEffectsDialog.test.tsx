import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import type { PlanChangeEffects } from '../plansApi';
import { ChangeEffectsDialog } from './ChangeEffectsDialog';

function renderDialog(effects: PlanChangeEffects, modelOnly = false) {
  const onConfirm = vi.fn();
  const onKeepEditing = vi.fn();
  const result = renderWithProviders(
    <ChangeEffectsDialog
      effects={effects}
      planId="P-1001"
      modelOnly={modelOnly}
      onConfirm={onConfirm}
      onKeepEditing={onKeepEditing}
    />,
  );
  return { ...result, onConfirm, onKeepEditing };
}

describe('ChangeEffectsDialog (FR-019, FR-020)', () => {
  it('names one activity the new model does not allow', async () => {
    const { user, onConfirm } = renderDialog(
      {
        modelChanged: true,
        disallowedActivities: [{ type: 'activity', id: 'A-1003', name: 'Fleet webinar' }],
        hiddenGroups: [],
        shownGroups: [],
      },
      true,
    );

    expect(screen.getByRole('alertdialog', { name: 'Change the model?' })).toHaveTextContent(
      "1 activity has a type the new model doesn't allow:",
    );
    expect(screen.getByRole('button', { name: 'Keep editing' })).toHaveFocus();
    await user.click(screen.getByRole('button', { name: 'Change model' }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it('says which groups are hidden and shown, and where', async () => {
    const { user, onKeepEditing } = renderDialog({
      modelChanged: false,
      disallowedActivities: [],
      hiddenGroups: [
        {
          unit: 'Lubricants',
          levels: ['plan', 'campaign', 'activity'],
          attributes: ['Lubricant type'],
        },
      ],
      shownGroups: [{ unit: 'Mobility', levels: ['plan'], attributes: ['SMP'] }],
    });

    const dialog = screen.getByRole('alertdialog', { name: 'Save these changes?' });
    expect(dialog).toHaveTextContent(
      'Lubricants attributes will be hidden on the plan, its campaigns and its activities. Their values are kept and come back if you add Lubricants again.',
    );
    expect(dialog).toHaveTextContent('Mobility attributes will be shown on the plan.');
    await user.click(screen.getByRole('button', { name: 'Keep editing' }));
    expect(onKeepEditing).toHaveBeenCalled();
  });
});
