import type { UseQueryResult } from '@tanstack/react-query';
import { CircleAlert, Info } from 'lucide-react';
import { useEffect, useId, useState } from 'react';
import { Navigate, useLocation, useSearchParams } from 'react-router';
import { ApiError } from '@/api/errors';
import type { components } from '@/api/generated/schema';
import { PublicLayout } from '@/ui/PublicLayout/PublicLayout';
import { SessionCheckPage } from '@/session/SessionCheckPage';
import { useSession } from '@/session/useSession';
import type { SessionEnd } from '@/session/useSession';
import { Avatar } from '@/ui/Avatar/Avatar';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { safeNext } from '@/session/safe-next';
import { ProductName } from '@/ui/Logo/Logo';
import { usePageTitle } from '@/ui/usePageTitle';
import { useDemoPeople } from './api';
import { lostChangesOf, noticeOf } from './schemas';
import { SignInForm } from './SignInForm';
import styles from './SignInPage.module.css';

type DemoPeople = components['schemas']['DemoPeople'];
type DemoPerson = components['schemas']['DemoPerson'];

const NOTICES: Record<SessionEnd, string> = {
  signedOut: 'You have been signed out.',
  sessionLost: 'Please sign in again',
};

export function SignInPage() {
  usePageTitle('Sign in');
  const { status } = useSession();
  const [searchParams] = useSearchParams();

  if (status === 'loading' || status === 'unreachable') {
    return <SessionCheckPage title="Sign in" />;
  }
  if (status === 'signedIn') {
    // Someone who is signed in — also the person who has just signed in here — goes on.
    return <Navigate to={safeNext(searchParams.get('next'))} replace />;
  }
  return (
    <PublicLayout>
      <div className={styles.card}>
        <div className={styles.heading}>
          <h1 className={styles.title}>
            Sign in to <ProductName />
          </h1>
          <p className={styles.subtitle}>Plan your marketing work, from plan to line item.</p>
        </div>
        <Notice />
        <SignInChoices />
      </div>
    </PublicLayout>
  );
}

function Notice() {
  const location = useLocation();
  const notice = noticeText(location.state);
  const [text, setText] = useState('');

  // Screen readers announce a change in a status region, not text that is there from the start:
  // so the region starts empty, and the notice is put in right after the page has appeared.
  useEffect(() => {
    const timer = window.setTimeout(() => setText(notice), 0);
    return () => window.clearTimeout(timer);
  }, [notice]);

  return (
    <p role="status" className={text ? styles.notice : styles.emptyNotice}>
      {text}
    </p>
  );
}

function noticeText(routerState: unknown): string {
  const notice = noticeOf(routerState);
  const lostChanges = lostChangesOf(routerState);
  if (notice === 'sessionLost' && lostChanges) {
    return `Your session ended, so your changes in '${lostChanges}' weren't saved. ${NOTICES.sessionLost}`;
  }
  return notice ? NOTICES[notice] : '';
}

function SignInChoices() {
  const { signIn } = useSession();
  const demoPeople = useDemoPeople();
  // While a sign-in is on its way, further clicks are ignored; nothing is greyed out.
  const [signingIn, setSigningIn] = useState(false);
  const [failure, setFailure] = useState('');

  async function attemptSignIn(email: string, password: string): Promise<ApiError | null> {
    setSigningIn(true);
    setFailure('');
    try {
      await signIn(email, password);
      return null;
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      // Field problems are shown at the fields; anything else above the form (FR-020).
      if (apiError.kind !== 'validation') {
        setFailure(apiError.message);
      }
      return apiError;
    } finally {
      setSigningIn(false);
    }
  }

  function signInAs(person: DemoPerson) {
    if (!signingIn && demoPeople.data) {
      // One sign-in for both ways: the person's e-mail and the demo password (research R6).
      attemptSignIn(person.email, demoPeople.data.password);
    }
  }

  // Outside the local and development environments the API has no demo sign-in (FR-009).
  if (demoPeople.error instanceof ApiError && demoPeople.error.kind === 'notFound') {
    return (
      <p className={styles.box}>
        Demo sign-in is turned off in this environment. Shell single sign-on comes later.
      </p>
    );
  }

  return (
    <>
      <div className={styles.demoNotice} role="note">
        <Info aria-hidden="true" />
        <div>
          <p className={styles.strong}>Demo sign-in — Shell single sign-on comes later</p>
          {demoPeople.data && (
            <p>
              All demo people share the password <code>{demoPeople.data.password}</code>.
            </p>
          )}
        </div>
      </div>
      {failure && (
        <p role="alert" className={styles.failure}>
          <CircleAlert />
          {failure}
        </p>
      )}
      <SignInForm signingIn={signingIn} onSignIn={attemptSignIn} />
      <DemoPeopleSection query={demoPeople} onChoose={signInAs} />
    </>
  );
}

interface DemoPeopleSectionProps {
  query: UseQueryResult<DemoPeople>;
  onChoose: (person: DemoPerson) => void;
}

function DemoPeopleSection({ query, onChoose }: DemoPeopleSectionProps) {
  const headingId = useId();

  return (
    <section aria-labelledby={headingId} className={styles.demoPeople}>
      <h2 id={headingId} className={styles.sectionTitle}>
        Explore as a demo person
      </h2>
      <p className={styles.hint}>Choose a person to sign in as them — no typing needed.</p>
      <DemoPeopleContent query={query} onChoose={onChoose} />
    </section>
  );
}

function DemoPeopleContent({ query, onChoose }: DemoPeopleSectionProps) {
  if (query.isPending) {
    return <Skeleton lines={3} height="40px" />;
  }
  if (query.isError) {
    return <ErrorState error={query.error} onRetry={() => query.refetch()} />;
  }
  if (query.data.people.length === 0) {
    // The demo data was not loaded yet (FR-028).
    return (
      <div className={styles.box}>
        <p className={styles.strong}>No demo users found — load the demo data first</p>
        <p>
          Stop SP&A One's API, run <code>go run ./cmd/seed</code> in the backend folder, start the
          API again with <code>go run ./cmd/api</code>, then reload this page.
        </p>
      </div>
    );
  }
  return (
    <ul className={styles.people}>
      {query.data.people.map((person) => (
        <li key={person.id}>
          {/* The button's name comes from its text: "Sign in as Priya Raman, Marketer". The hidden
              words keep the visible name and job title inside it (WCAG 2.5.3, label in name). */}
          <button type="button" className={styles.person} onClick={() => onChoose(person)}>
            <Avatar initials={person.initials} size="sm" />
            <span className={styles.personText}>
              <span className={styles.personName}>
                <span className="visually-hidden">Sign in as</span> {person.name}
              </span>
              {person.jobTitle && (
                <span className={styles.personTitle}>
                  <span className="visually-hidden">,</span> {person.jobTitle}
                </span>
              )}
            </span>
          </button>
        </li>
      ))}
    </ul>
  );
}
