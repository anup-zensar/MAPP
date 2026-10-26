import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { TextField } from '../TextField/TextField';
import { FormPanel } from './FormPanel';
import { problemOf } from './problem-of';
import type { SaveFailure } from './problem-of';

const PROBLEMS = [{ field: 'teamName', message: 'Enter a name.' }];

function Example({
  onSave,
  onClose,
  failure = null,
}: {
  onSave: () => void;
  onClose: () => void;
  failure?: SaveFailure | null;
}) {
  const [name, setName] = useState('');
  return (
    <FormPanel
      title="New team"
      isDirty={name !== ''}
      saving={false}
      problems={name === '' ? PROBLEMS : []}
      failure={failure}
      onSave={onSave}
      onClose={onClose}
      saveLabel="Save team"
    >
      <TextField
        label="Name"
        name="teamName"
        value={name}
        onChange={setName}
        error={problemOf(name === '' ? PROBLEMS : [], 'teamName')}
      />
    </FormPanel>
  );
}

describe('FormPanel', () => {
  it('saves with its own label and shows every problem at once (design system §6.2)', async () => {
    const onSave = vi.fn();
    const { user } = renderWithProviders(<Example onSave={onSave} onClose={vi.fn()} />);

    expect(screen.getByRole('dialog', { name: 'New team' })).toBeInTheDocument();
    expect(screen.getAllByText('Enter a name.').length).toBeGreaterThan(0);

    await user.click(screen.getByRole('button', { name: 'Save team' }));
    expect(onSave).toHaveBeenCalledOnce();
  });

  it('asks before discarding typed changes', async () => {
    const onClose = vi.fn();
    const { user } = renderWithProviders(<Example onSave={vi.fn()} onClose={onClose} />);

    await user.type(screen.getByRole('textbox', { name: 'Name' }), 'Lubricants');
    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(
      await screen.findByRole('alertdialog', { name: 'Discard changes?' }),
    ).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
  });

  it('shows a failure that is not about one field, with the next step', () => {
    renderWithProviders(
      <Example
        onSave={vi.fn()}
        onClose={vi.fn()}
        failure={{ message: 'Not saved — Daniel changed this team.', nextStep: 'Save again.' }}
      />,
    );

    // The failure comes first, above the error summary.
    expect(screen.getAllByRole('alert')[0]).toHaveTextContent(
      'Not saved — Daniel changed this team.Save again.',
    );
  });
});
