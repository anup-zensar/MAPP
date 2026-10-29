import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import type { ApiError } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import { FormField } from '@/ui/FormField/FormField';
import { PasswordField } from '@/ui/PasswordField/PasswordField';
import { TextField } from '@/ui/TextField/TextField';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import { signInSchema } from './schemas';
import type { SignInValues } from './schemas';
import styles from './SignInPage.module.css';

/** Signs in; resolves to the error when that failed, or to null when it worked. */
export type SignInAttempt = (email: string, password: string) => Promise<ApiError | null>;

const FIELDS = ['email', 'password'] as const;

interface SignInFormProps {
  signingIn: boolean;
  onSignIn: SignInAttempt;
}

/** E-mail, password and Sign in. Every empty field is reported at once (FR-004). */
export function SignInForm({ signingIn, onSignIn }: SignInFormProps) {
  const form = useForm<SignInValues>({
    resolver: zodResolver(signInSchema),
    defaultValues: { email: '', password: '' },
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems, clearProblems } = useFormProblems(form, FIELDS);

  const submit = form.handleSubmit(async ({ email, password }) => {
    clearProblems();
    const error = await onSignIn(email, password);
    if (error?.kind === 'validation') {
      showServerProblems(error.fields);
    } else if (error?.kind === 'incorrectCredentials' || error?.kind === 'tooManyAttempts') {
      // The e-mail stays, so only the password is typed again (US1 scenario 4, 038 FR-014).
      form.setValue('password', '');
    }
  }, onInvalid);

  return (
    <form
      noValidate
      aria-busy={signingIn}
      className={styles.form}
      onSubmit={(event) => {
        event.preventDefault();
        if (!signingIn) {
          void submit(event);
        }
      }}
    >
      <FormErrorSummary errors={problems} />
      <FormField
        control={form.control}
        name="email"
        render={(field) => (
          <TextField label="E-mail" type="email" autoComplete="username" {...field} />
        )}
      />
      <FormField
        control={form.control}
        name="password"
        render={(field) => (
          <PasswordField label="Password" autoComplete="current-password" {...field} />
        )}
      />
      <Button type="submit" variant="primary" size="lg" className={styles.submit}>
        {signingIn ? 'Signing in…' : 'Sign in'}
      </Button>
    </form>
  );
}
