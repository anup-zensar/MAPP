import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, describe, expect, it } from 'vitest';
import { SESSION_KEY } from '@/session/useSession';
import { resetAllWorkStore, useAllWorkState } from './allWorkStore';
import { standard2026 } from '@/testing/fixtures/all-work';

afterEach(() => resetAllWorkStore());

function wrapperWith(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe('allWorkStore (FR-009, FR-016, D-122)', () => {
  it('keeps the state, page, page size and scroll position between page visits', () => {
    const queryClient = new QueryClient();
    const first = renderHook(() => useAllWorkState('U-1002'), {
      wrapper: wrapperWith(queryClient),
    });
    expect(first.result.current[0].initialized).toBe(false);

    const state = { ...standard2026(), search: 'helix' };
    act(() =>
      first.result.current[1]({ initialized: true, state, page: 3, pageSize: 100, scrollY: 640 }),
    );
    first.unmount();

    const again = renderHook(() => useAllWorkState('U-1002'), {
      wrapper: wrapperWith(queryClient),
    });
    expect(again.result.current[0]).toMatchObject({
      initialized: true,
      state,
      page: 3,
      pageSize: 100,
      scrollY: 640,
    });
  });

  it('keeps each person’s state apart', () => {
    const queryClient = new QueryClient();
    const priya = renderHook(() => useAllWorkState('U-1002'), {
      wrapper: wrapperWith(queryClient),
    });
    act(() => priya.result.current[1]({ initialized: true, state: standard2026() }));

    const daniel = renderHook(() => useAllWorkState('U-1003'), {
      wrapper: wrapperWith(queryClient),
    });
    expect(daniel.result.current[0].initialized).toBe(false);
  });

  it('changes with a function of the entry as it is', () => {
    const queryClient = new QueryClient();
    const { result } = renderHook(() => useAllWorkState('U-1002'), {
      wrapper: wrapperWith(queryClient),
    });
    act(() => result.current[1]({ page: 2 }));
    act(() => result.current[1]((entry) => ({ page: entry.page + 1 })));

    expect(result.current[0].page).toBe(3);
  });

  it('starts a new visit after signing out and in again (the cached data is cleared)', () => {
    const queryClient = new QueryClient();
    const first = renderHook(() => useAllWorkState('U-1002'), {
      wrapper: wrapperWith(queryClient),
    });
    act(() => first.result.current[1]({ initialized: true, state: standard2026(), page: 2 }));
    first.unmount();

    // What the session does when someone signs out: every cached answer but the session goes.
    queryClient.removeQueries({ predicate: (query) => query.queryKey[0] !== SESSION_KEY[0] });

    const again = renderHook(() => useAllWorkState('U-1002'), {
      wrapper: wrapperWith(queryClient),
    });
    expect(again.result.current[0]).toMatchObject({ initialized: false, state: null, page: 1 });
  });
});
