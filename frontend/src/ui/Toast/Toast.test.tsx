import { act, render, renderHook, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ToastProvider } from './Toast';
import { useToast } from './useToast';
import type { ToastOptions } from './useToast';

const DELETED = "Sample folder 'Ideas' deleted";

/** A page that shows the given message when its button is pressed. */
function ShowButton({ toast }: { toast: ToastOptions }) {
  const { show } = useToast();
  return (
    <button type="button" onClick={() => show(toast)}>
      Show message
    </button>
  );
}

function renderWithToasts(toast: ToastOptions) {
  return render(
    <ToastProvider>
      <ShowButton toast={toast} />
    </ToastProvider>,
  );
}

/**
 * Fake timers that also tick on their own, because Testing Library waits for a real setTimeout(0)
 * after every user action; the tests still jump ahead with vi.advanceTimersByTime().
 */
function startFakeClock() {
  vi.useFakeTimers({ shouldAdvanceTime: true });
}

function messages() {
  return screen.getByRole('region', { name: 'Messages (F8)' });
}

describe('ToastProvider and useToast', () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('closes a message after 4 seconds (FR-019)', async () => {
    startFakeClock();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderWithToasts({ message: "Sample folder 'Ideas' created" });
    await user.click(screen.getByRole('button', { name: 'Show message' }));

    act(() => vi.advanceTimersByTime(3_900));
    expect(messages()).toHaveTextContent("Sample folder 'Ideas' created");

    act(() => vi.advanceTimersByTime(200));
    expect(messages()).not.toHaveTextContent("Sample folder 'Ideas' created");
  });

  it('keeps a message with Undo for 10 seconds (FR-026)', async () => {
    startFakeClock();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderWithToasts({
      message: DELETED,
      action: { label: 'Undo', altText: 'Restore it from the recycle bin', onClick: vi.fn() },
    });
    await user.click(screen.getByRole('button', { name: 'Show message' }));

    act(() => vi.advanceTimersByTime(9_900));
    expect(screen.getByRole('button', { name: 'Undo' })).toBeInTheDocument();

    act(() => vi.advanceTimersByTime(200));
    expect(screen.queryByRole('button', { name: 'Undo' })).not.toBeInTheDocument();
  });

  it('runs the action and closes the message when the action is chosen (FR-026)', async () => {
    const user = userEvent.setup();
    const undo = vi.fn();
    renderWithToasts({
      message: DELETED,
      action: { label: 'Undo', altText: 'Restore it from the recycle bin', onClick: undo },
    });
    await user.click(screen.getByRole('button', { name: 'Show message' }));

    await user.click(screen.getByRole('button', { name: 'Undo' }));

    expect(undo).toHaveBeenCalledTimes(1);
    expect(messages()).not.toHaveTextContent(DELETED);
  });

  it('waits while the pointer rests on the message (FR-026)', async () => {
    startFakeClock();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderWithToasts({
      message: DELETED,
      action: { label: 'Undo', altText: 'Restore it from the recycle bin', onClick: vi.fn() },
    });
    await user.click(screen.getByRole('button', { name: 'Show message' }));
    act(() => vi.advanceTimersByTime(8_000));

    await user.hover(screen.getByText(DELETED));
    act(() => vi.advanceTimersByTime(20_000));
    expect(screen.getByRole('button', { name: 'Undo' })).toBeInTheDocument();

    await user.unhover(screen.getByText(DELETED));
    act(() => vi.advanceTimersByTime(2_100));
    expect(screen.queryByRole('button', { name: 'Undo' })).not.toBeInTheDocument();
  });

  it('waits while the keyboard focus is on the message, reached with F8 (FR-026)', async () => {
    startFakeClock();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderWithToasts({
      message: DELETED,
      action: { label: 'Undo', altText: 'Restore it from the recycle bin', onClick: vi.fn() },
    });
    await user.click(screen.getByRole('button', { name: 'Show message' }));

    await user.keyboard('{F8}');
    await user.tab();
    act(() => vi.advanceTimersByTime(20_000));

    expect(screen.getByRole('button', { name: 'Undo' })).toBeInTheDocument();
  });

  it('announces the message at once, with the other way to do the action (FR-026)', async () => {
    const user = userEvent.setup();
    renderWithToasts({
      message: DELETED,
      action: { label: 'Undo', altText: 'Restore it from the recycle bin', onClick: vi.fn() },
    });

    await user.click(screen.getByRole('button', { name: 'Show message' }));

    const announcement = await screen.findByRole('status');
    expect(announcement).toHaveAttribute('aria-live', 'assertive');
    await waitFor(() =>
      expect(announcement).toHaveTextContent(`${DELETED} Restore it from the recycle bin`),
    );
  });

  it('038 FR-049: a message with an action reads "<message> — <action>"', async () => {
    const user = userEvent.setup();
    renderWithToasts({
      message: "Plan 'FY2026 Lubricants' restored",
      action: { label: 'View', altText: 'Open the plan', onClick: vi.fn() },
    });

    await user.click(screen.getByRole('button', { name: 'Show message' }));

    await waitFor(() =>
      expect(messages()).toHaveTextContent("Plan 'FY2026 Lubricants' restored — View"),
    );
  });

  it('lets the caller close its message early, for example when a newer one replaces it', async () => {
    const user = userEvent.setup();
    function ReplaceablePage() {
      const { show } = useToast();
      const [close, setClose] = useState<() => void>();
      return (
        <>
          <button type="button" onClick={() => setClose(() => show({ message: DELETED }))}>
            Delete
          </button>
          <button type="button" onClick={() => close?.()}>
            Close the message
          </button>
        </>
      );
    }
    render(
      <ToastProvider>
        <ReplaceablePage />
      </ToastProvider>,
    );
    await user.click(screen.getByRole('button', { name: 'Delete' }));
    expect(messages()).toHaveTextContent(DELETED);

    await user.click(screen.getByRole('button', { name: 'Close the message' }));

    expect(messages()).not.toHaveTextContent(DELETED);
  });

  it('says what is missing when used outside the provider', () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});

    expect(() => renderHook(() => useToast())).toThrow(
      'useToast needs the ToastProvider around the app (App.tsx).',
    );
  });
});
