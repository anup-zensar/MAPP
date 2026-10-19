import { render, screen } from '@testing-library/react';
import { FolderPlus } from 'lucide-react';
import { describe, expect, it } from 'vitest';
import { Button } from '../Button/Button';
import { EmptyState } from './EmptyState';

describe('EmptyState', () => {
  it('explains what is missing and offers the way to create it (FR-019)', () => {
    render(
      <EmptyState
        icon={<FolderPlus />}
        title="No sample folders yet — create your first sample folder"
        action={<Button variant="primary">New sample folder</Button>}
      />,
    );

    expect(
      screen.getByText('No sample folders yet — create your first sample folder'),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'New sample folder' })).toBeInTheDocument();
  });

  it('can add one more sentence (FR-015)', () => {
    render(<EmptyState title="Coming soon" text="This part of NorthStar isn't built yet." />);

    expect(screen.getByText('Coming soon')).toBeInTheDocument();
    expect(screen.getByText("This part of NorthStar isn't built yet.")).toBeInTheDocument();
  });
});
