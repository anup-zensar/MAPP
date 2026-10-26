import { useQuery } from '@tanstack/react-query';
import type { UseQueryResult } from '@tanstack/react-query';
import { CircleAlert, Info } from 'lucide-react';
import { useEffect, useId, useState } from 'react';
import type { FormEvent } from 'react';
import { Navigate, useLocation, useSearchParams } from 'react-router';
import { api, call } from '@/api/client';
import { ApiError } from '@/api/errors';
import type { components } from '@/api/generated/schema';
import { PublicLayout } from '@/app/PublicLayout';
import { SessionCheckPage } from '@/app/SessionCheckPage';
import { useSession } from '@/app/useSession';
import type { SessionEnd } from '@/app/useSession';
import { Avatar } from '@/ui/Avatar/Avatar';
import { Button } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import { PasswordField } from '@/ui/PasswordField/PasswordField';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TextField } from '@/ui/TextField/TextField';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { safeNext } from '@/lib/safeNext';
import { usePageTitle } from '@/lib/usePageTitle';
import styles from './SignInPage.module.css';

type DemoPeople = components['schemas']['DemoPeople'];
type DemoPerson = components['schemas']['DemoPerson'];

/** Signs in; resolves to the error when that failed, or to null when it worked. */
type SignInAttempt = (email: string, password: string) => Promise<ApiError | null>;

/** The notice the page can be opened with, after a session ended (plan "Session"). */
const NOTICES: Record<SessionEnd, string> = {
  signedOut: 'You have been signed out.',
  sessionLost: 'Please sign in again',
};

/**
 * The one sign-in page (US1, FR-001 – FR-004, FR-009, FR-028): one white card on the grey page
 * (design system §5.8) — type a demo e-mail and the demo password, or choose a demo person.
 * Afterwards people go on to the page they asked for (the `next` address), or Home. The brand is
 * shown once, in the bar above the card (UI audit S-01).
 */
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
          <h1 className={styles.title}>Sign in to MAPP</h1>
          <p className={styles.subtitle}>Plan your marketing work, from plan to line item.</p>
        </div>
        <Notice />
        <SignInChoices />
      </div>
    </PublicLayout>
  );
}

/** "You have been signed out." or "Please sign in again", when the page was opened with one. */
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

/** The notice's text from the router state that endSession sends along ("" for none). */
function noticeText(routerState: unknown): string {
  const notice = (routerState as { notice?: SessionEnd } | null)?.notice;
  return notice ? NOTICES[notice] : '';
}

/**
 * Everything that depends on the demo people answer: the demo notice with the password, the
 * form and the demo people — or, when demo sign-in is off, only the sentence that says so.
 */
function SignInChoices() {
  const { signIn } = useSession();
  const demoPeople = useQuery({
    queryKey: ['demo-people'],
    queryFn: () => call(api.GET('/api/v1/auth/demo-people')),
  });
  // While a sign-in is on its way, further clicks are ignored; nothing is greyed out.
  const [signingIn, setSigningIn] = useState(false);
  // A failure shown above the form, such as "Incorrect e-mail or password.".
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

interface SignInFormProps {
  signingIn: boolean;
  onSignIn: SignInAttempt;
}

/** E-mail, password and Sign in. Every empty field is reported at once (FR-004). */
function SignInForm({ signingIn, onSignIn }: SignInFormProps) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  // Every problem of the last attempt; a new list each time, so the summary takes the focus.
  const [problems, setProblems] = useState<FieldProblem[]>([]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (signingIn) {
      return;
    }
    // Empty fields are reported before the API is asked anything.
    const emptyFields = findEmptyFields(email, password);
    setProblems(emptyFields);
    if (emptyFields.length > 0) {
      return;
    }
    const error = await onSignIn(email, password);
    if (error?.kind === 'validation') {
      setProblems(error.fields);
    } else if (error?.kind === 'incorrectCredentials') {
      // The e-mail stays, so only the password is typed again (US1 scenario 4).
      setPassword('');
    }
  }

  function problemOf(field: string): string | undefined {
    return problems.find((problem) => problem.field === field)?.message;
  }

  return (
    <form noValidate aria-busy={signingIn} className={styles.form} onSubmit={handleSubmit}>
      <FormErrorSummary errors={problems} />
      <TextField
        label="E-mail"
        name="email"
        type="email"
        autoComplete="username"
        value={email}
        onChange={setEmail}
        error={problemOf('email')}
      />
      <PasswordField
        label="Password"
        name="password"
        autoComplete="current-password"
        value={password}
        onChange={setPassword}
        error={problemOf('password')}
      />
      <Button type="submit" variant="primary" size="lg" className={styles.submit}>
        {signingIn ? 'Signing in…' : 'Sign in'}
      </Button>
    </form>
  );
}

/** The messages of the empty fields, in the order of the form. */
function findEmptyFields(email: string, password: string): FieldProblem[] {
  const problems: FieldProblem[] = [];
  if (email.trim() === '') {
    problems.push({ field: 'email', message: 'Enter your e-mail' });
  }
  if (password === '') {
    problems.push({ field: 'password', message: 'Enter the password' });
  }
  return problems;
}

interface DemoPeopleSectionProps {
  /** The answer of GET /api/v1/auth/demo-people. */
  query: UseQueryResult<DemoPeople>;
  onChoose: (person: DemoPerson) => void;
}

/** "Explore as a demo person": one click signs in as that person (FR-001, FR-003). */
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

/** The section's content while loading, after a failure, without demo people, or the people. */
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
          Stop MAPP's API, run <code>go run ./cmd/seed</code> in the backend folder, start the API
          again with <code>go run ./cmd/api</code>, then reload this page.
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
