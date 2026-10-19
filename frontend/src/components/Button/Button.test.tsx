import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Plus, X } from 'lucide-react';
import type { FormEvent } from 'react';
import { MemoryRouter } from 'react-router';
import { describe, expect, it, vi } from 'vitest';
import { Button, LinkButton } from './Button';

describe('Button', () => {
  it('is a button named by its label, which never submits a form by accident (FR-019)', () => {
    render(<Button variant="primary">Create plan</Button>);

    const button = screen.getByRole('button', { name: 'Create plan' });
    expect(button).toHaveAttribute('type', 'button');
  });

  it('runs its action when clicked (FR-019)', async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Save</Button>);

    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('submits its form when it is the form’s submit button', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn((event: FormEvent) => event.preventDefault());
    render(
      <form onSubmit={onSubmit}>
        <Button type="submit" variant="primary">
          Sign in
        </Button>
      </form>,
    );

    await user.click(screen.getByRole('button', { name: 'Sign in' }));

    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it('keeps its name when it shows an icon, because the icon is decoration (FR-019)', () => {
    render(<Button icon={<Plus />}>New sample folder</Button>);

    expect(screen.getByRole('button', { name: 'New sample folder' })).toBeInTheDocument();
  });

  it('can be an icon-only button named by aria-label (FR-019)', () => {
    render(<Button variant="ghost" icon={<X />} aria-label="Close" />);

    expect(screen.getByRole('button', { name: 'Close' })).toBeInTheDocument();
  });

  it.each(['primary', 'secondary', 'danger', 'ghost'] as const)(
    'offers the %s variant (FR-019)',
    (variant) => {
      render(<Button variant={variant}>Go</Button>);

      expect(screen.getByRole('button', { name: 'Go' })).toBeInTheDocument();
    },
  );
});

describe('LinkButton', () => {
  it('is a link that looks like a button (FR-019)', () => {
    render(
      <MemoryRouter>
        <LinkButton to="/" variant="primary">
          Go to Home
        </LinkButton>
      </MemoryRouter>,
    );

    expect(screen.getByRole('link', { name: 'Go to Home' })).toHaveAttribute('href', '/');
  });
});
