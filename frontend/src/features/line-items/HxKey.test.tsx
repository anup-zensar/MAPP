import { screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { expectNoAxeViolations } from '@/testing/axe';
import { testUser } from '@/testing/fixtures/users';
import { HxKey } from './HxKey';

function renderKey() {
  mockApi([]);
  return renderWithProviders(<HxKey value="HX1006" />, { user: testUser() });
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe('030 FR-016: the HX key', () => {
  it('is read-only text labelled as generated from the line-item number', async () => {
    renderKey();

    const key = screen.getByText('HX1006');
    expect(key.tagName).not.toBe('INPUT');
    expect(key).toHaveAccessibleDescription('Generated from the line-item number');
    await expectNoAxeViolations();
  });

  it('copies the key and says so', async () => {
    const { user } = renderKey();

    await user.click(screen.getByRole('button', { name: 'Copy HX key HX1006' }));

    expect(await screen.findByText('HX key copied')).toBeInTheDocument();
    expect(await navigator.clipboard.readText()).toBe('HX1006');
  });

  it('says what to do when the browser refuses to copy', async () => {
    const { user } = renderKey();
    vi.spyOn(navigator.clipboard, 'writeText').mockRejectedValue(new Error('denied'));

    await user.click(screen.getByRole('button', { name: 'Copy HX key HX1006' }));

    expect(
      await screen.findByText("Couldn't copy the HX key — select it and copy it yourself."),
    ).toBeInTheDocument();
  });
});
