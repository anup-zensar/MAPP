import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { InfoTip } from './InfoTip';

const TEXT = 'The budget given to the plan when it was created.';

function renderTip() {
  return render(
    <>
      <span>Planned budget</span>
      <InfoTip label="Planned budget" text={TEXT} />
      <button type="button">Somewhere else</button>
    </>,
  );
}

function tipButton() {
  return screen.getByRole('button', { name: 'About Planned budget' });
}

function bubble() {
  return screen.queryByRole('dialog', { name: 'About Planned budget' });
}

describe('InfoTip', () => {
  it('is a button named "About <label>" (FR-019)', () => {
    renderTip();

    expect(tipButton()).toBeInTheDocument();
    expect(bubble()).not.toBeInTheDocument();
  });

  it('opens on hover and closes when the pointer leaves (FR-019)', async () => {
    const user = userEvent.setup();
    renderTip();

    await user.hover(tipButton());
    expect(bubble()).toHaveTextContent(TEXT);

    await user.unhover(tipButton());
    await waitFor(() => expect(bubble()).not.toBeInTheDocument());
  });

  it('opens on keyboard focus and closes on Escape, keeping the focus (FR-019)', async () => {
    const user = userEvent.setup();
    renderTip();

    await user.tab();
    expect(tipButton()).toHaveFocus();
    expect(bubble()).toHaveTextContent(TEXT);

    await user.keyboard('{Escape}');
    expect(bubble()).not.toBeInTheDocument();
    expect(tipButton()).toHaveFocus();
  });

  it('closes when the focus moves on (FR-019)', async () => {
    const user = userEvent.setup();
    renderTip();
    await user.tab();
    expect(bubble()).toBeInTheDocument();

    await user.tab();

    expect(screen.getByRole('button', { name: 'Somewhere else' })).toHaveFocus();
    expect(bubble()).not.toBeInTheDocument();
  });

  it('opens on a click or tap, stays open on a second click, and closes on a click outside (FR-019)', async () => {
    const user = userEvent.setup();
    renderTip();

    await user.click(tipButton());
    expect(bubble()).toHaveTextContent(TEXT);

    await user.click(tipButton());
    expect(bubble()).toBeInTheDocument();

    await user.click(screen.getByText('Planned budget'));
    await waitFor(() => expect(bubble()).not.toBeInTheDocument());
  });

  it('stays open when its own text is clicked, for example to select it (FR-019)', async () => {
    const user = userEvent.setup();
    renderTip();
    await user.click(tipButton());

    await user.click(screen.getByText(TEXT));

    expect(bubble()).toBeInTheDocument();
  });

  it('opens on a touch tap, which has no hover (FR-019)', async () => {
    const user = userEvent.setup();
    renderTip();

    await user.pointer({ keys: '[TouchA]', target: tipButton() });

    expect(bubble()).toHaveTextContent(TEXT);
  });
});
