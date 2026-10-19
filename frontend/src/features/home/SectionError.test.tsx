import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { SectionError } from './SectionError';

describe('SectionError', () => {
  it('says the section could not load and tries again on request (FR-022)', async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    render(<SectionError onRetry={onRetry} />);

    expect(screen.getByRole('alert')).toHaveTextContent("This section couldn't load");
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(onRetry).toHaveBeenCalledOnce();
  });
});
