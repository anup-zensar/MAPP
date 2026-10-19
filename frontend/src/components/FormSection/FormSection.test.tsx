import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Button } from '../Button/Button';
import { FormFooter } from '../FormFooter/FormFooter';
import { FormSection } from './FormSection';

describe('FormSection and FormFooter', () => {
  it('names each section of a full-page form by its heading (design system §7)', () => {
    render(
      <form aria-label="New plan">
        <FormSection title="Basics" description="What the plan is called and who owns it.">
          <input aria-label="Plan name" />
        </FormSection>
        <FormSection title="Budget">
          <input aria-label="Plan budget" />
        </FormSection>
        <FormFooter>
          <Button type="submit" variant="primary">
            Save
          </Button>
          <Button>Cancel</Button>
        </FormFooter>
      </form>,
    );

    const basics = screen.getByRole('region', { name: 'Basics' });
    expect(basics).toHaveTextContent('What the plan is called and who owns it.');
    expect(screen.getByRole('heading', { level: 2, name: 'Budget' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Save' })).toBeEnabled();
  });
});
