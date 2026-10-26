import { act, screen, waitFor, within } from '@testing-library/react';
import { useState } from 'react';
import { Link } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { api, call } from '@/api/client';
import type { CurrentUser } from '@/app/useSession';
import { mockApi } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { useUnsavedChanges } from './useUnsavedChanges';
import { testUser } from '@/test/users';

const PRIYA: CurrentUser = testUser();

const DANIEL: CurrentUser = testUser({
  id: 'U-1003',
  name: 'Daniel Okoro',
  firstName: 'Daniel',
  email: 'daniel.okoro@shell.com',
  jobTitle: 'Marketing team leader',
  initials: 'DO',
});

/**
 * A form with one field, as a page has it: a link to leave the page, Cancel to close the form,
 * and Save, which sends the name to the API.
 */
function NameForm({ onClose = () => {} }: { onClose?: () => void }) {
  const [name, setName] = useState('');
  const { dialog, confirmDiscard } = useUnsavedChanges(name !== '');

  return (
    <>
      <label htmlFor="name">Name</label>
      <input id="name" value={name} onChange={(event) => setName(event.target.value)} />
      <Link to="/plans">Plans</Link>
      <button type="button" onClick={() => confirmDiscard(onClose)}>
        Cancel
      </button>
      <button
        type="button"
        onClick={() => call(api.POST('/api/v1/sample-folders', { body: { name } })).catch(() => {})}
      >
        Save
      </button>
      {dialog}
    </>
  );
}

/** The form at /form; every other address shows a stand-in page. */
function renderForm(onClose?: () => void) {
  return renderWithProviders(<NameForm onClose={onClose} />, {
    user: PRIYA,
    route: '/form',
    path: '/form',
  });
}

function discardQuestion() {
  return screen.getByRole('alertdialog', { name: 'Discard changes?' });
}

/** What the browser does before a reload or closing the tab; true when the page asks to stay. */
function pageAsksBeforeUnload(): boolean {
  const event = new Event('beforeunload', { cancelable: true });
  window.dispatchEvent(event);
  return event.defaultPrevented;
}

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('useUnsavedChanges', () => {
  it('asks "Discard changes?" before leaving a page with typed changes, with Keep editing focused (design system §6.4)', async () => {
    const { user, router } = renderForm();
    await user.type(screen.getByLabelText('Name'), 'Ideas');

    await user.click(screen.getByRole('link', { name: 'Plans' }));

    const question = discardQuestion();
    expect(question).toHaveAccessibleDescription("Your changes haven't been saved.");
    expect(within(question).getByRole('button', { name: 'Keep editing' })).toHaveFocus();
    expect(within(question).getByRole('button', { name: 'Discard changes' })).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/form');
  });

  it('stays on the page with the typed changes on Keep editing (design system §6.4)', async () => {
    const { user, router } = renderForm();
    await user.type(screen.getByLabelText('Name'), 'Ideas');
    await user.click(screen.getByRole('link', { name: 'Plans' }));

    await user.click(screen.getByRole('button', { name: 'Keep editing' }));

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(screen.getByLabelText('Name')).toHaveValue('Ideas');
    expect(router.state.location.pathname).toBe('/form');
  });

  it('leaves the page, dropping the changes, on Discard changes (design system §6.4)', async () => {
    const { user, router } = renderForm();
    await user.type(screen.getByLabelText('Name'), 'Ideas');
    await user.click(screen.getByRole('link', { name: 'Plans' }));

    await user.click(screen.getByRole('button', { name: 'Discard changes' }));

    expect(await screen.findByText('Another page')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/plans');
  });

  it('asks nothing when nothing was typed (design system §6.4)', async () => {
    const { user } = renderForm();

    await user.click(screen.getByRole('link', { name: 'Plans' }));

    expect(await screen.findByText('Another page')).toBeInTheDocument();
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  });

  it('also asks before going Back with the browser (plan "End-to-end tests", history)', async () => {
    const { user, router } = renderWithProviders(<NameForm />, {
      user: PRIYA,
      route: '/plans',
      path: '/form',
    });
    await act(() => router.navigate('/form'));
    await user.type(screen.getByLabelText('Name'), 'Ideas');

    await act(() => router.navigate(-1));

    expect(discardQuestion()).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/form');

    await user.click(screen.getByRole('button', { name: 'Discard changes' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/plans'));
  });

  it('asks the browser to confirm a reload or closing the tab only while there are typed changes (plan "Shared hooks")', async () => {
    const { user } = renderForm();
    expect(pageAsksBeforeUnload()).toBe(false);

    await user.type(screen.getByLabelText('Name'), 'Ideas');
    expect(pageAsksBeforeUnload()).toBe(true);

    await user.clear(screen.getByLabelText('Name'));
    expect(pageAsksBeforeUnload()).toBe(false);
  });

  it('closes a form at once when nothing was typed (design system §6.2)', async () => {
    const onClose = vi.fn();
    const { user } = renderForm(onClose);

    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(onClose).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  });

  it('asks before closing a form with typed changes, and closes it only on Discard changes (design system §6.2)', async () => {
    const onClose = vi.fn();
    const { user } = renderForm(onClose);
    await user.type(screen.getByLabelText('Name'), 'Ideas');

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(discardQuestion()).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Keep editing' }));
    expect(onClose).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    await user.click(screen.getByRole('button', { name: 'Discard changes' }));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  });

  it('lets the sign-in page open when the session has ended, as nothing can be saved without one (US4 scenario 2)', async () => {
    mockApi({
      'POST /api/v1/sample-folders': {
        status: 401,
        body: { error: { code: 'session_expired', message: 'Please sign in again', fields: [] } },
      },
    });
    const { user, router } = renderForm();
    await user.type(screen.getByLabelText('Name'), 'Ideas');

    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Sign-in page')).toBeInTheDocument();
    expect(router.state.location.search).toBe('?next=%2Fform');
    expect(router.state.location.state).toEqual({ notice: 'sessionLost' });
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  });

  it('lets Home open when another person signed in in another tab, so nothing is saved in their name (spec edge case)', async () => {
    mockApi({ 'GET /api/v1/me': { body: DANIEL } });
    const { user, router, queryClient } = renderForm();
    await user.type(screen.getByLabelText('Name'), 'Ideas');

    // The session is checked again, for example when the window regains focus.
    await act(() => queryClient.refetchQueries({ queryKey: ['session'] }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/'));
    expect(await screen.findByText('Another page')).toBeInTheDocument();
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  });
});
