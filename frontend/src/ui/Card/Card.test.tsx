import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Button } from '../Button/Button';
import { Card } from './Card';

describe('Card', () => {
  it('is a section named by its title, with the subtitle and content (FR-019)', () => {
    render(
      <Card title="Getting started" subtitle="The Plan module in five steps">
        <p>Step one</p>
      </Card>,
    );

    const card = screen.getByRole('region', { name: 'Getting started' });
    expect(card).toContainElement(
      screen.getByRole('heading', { level: 2, name: 'Getting started' }),
    );
    expect(card).toHaveTextContent('The Plan module in five steps');
    expect(card).toHaveTextContent('Step one');
  });

  it('shows its actions in the header (FR-019)', () => {
    render(
      <Card title="Sample folders" actions={<Button size="sm">Refresh</Button>}>
        <p>No sample folders yet</p>
      </Card>,
    );

    expect(screen.getByRole('button', { name: 'Refresh' })).toBeInTheDocument();
  });

  it('can hold content without a title (FR-019)', () => {
    render(
      <Card>
        <p>Coming soon</p>
      </Card>,
    );

    expect(screen.getByText('Coming soon')).toBeInTheDocument();
    expect(screen.queryByRole('heading')).not.toBeInTheDocument();
  });
});
