import { CircleAlert, CloudOff, RotateCw } from 'lucide-react';
import { ApiError } from '@/api/errors';
import { Button } from '../Button/Button';
import styles from './ErrorState.module.css';

export interface ErrorStateProps {
  /** What went wrong: usually an ApiError; anything else is shown as a general failure. */
  error: unknown;
  /** Tries again: reloads the data (or the page). */
  onRetry: () => void;
}

/**
 * Shown instead of content that failed to load: a plain-language message and Try again (FR-020).
 * Technical details never reach the screen; anything that is not an API error is shown with the
 * general message "Something went wrong. Please try again."
 */
export function ErrorState({ error, onRetry }: ErrorStateProps) {
  const apiError = error instanceof ApiError ? error : new ApiError('other');

  return (
    <div className={styles.error}>
      <div className={styles.icon}>
        {apiError.kind === 'unreachable' ? <CloudOff /> : <CircleAlert />}
      </div>
      <p role="alert" className={styles.message}>
        {apiError.message}
      </p>
      <Button variant="primary" icon={<RotateCw />} onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}
