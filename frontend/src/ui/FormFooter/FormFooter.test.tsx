import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Button } from '../Button/Button';
import { FormFooter } from './FormFooter';

describe('FormFooter', () => {
  it('holds the form’s buttons, Save first (design system §6.2)', () => {
    render(
      <FormFooter>
        <Button type="submit" variant="primary">
          Save
        </Button>
        <Button>Save and add another</Button>
        <Button>Cancel</Button>
      </FormFooter>,
    );

    expect(screen.getAllByRole('button').map((button) => button.textContent)).toEqual([
      'Save',
      'Save and add another',
      'Cancel',
    ]);
  });
});
