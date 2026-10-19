import { cx } from '@/lib/cx';
import styles from './Avatar.module.css';

export interface AvatarProps {
  /** Two letters from the API ("PR" for Priya Raman). */
  initials: string;
  /** sm 28 px (lists), md 32 px (account button), lg 40 px (account menu header). */
  size?: 'sm' | 'md' | 'lg';
}

/**
 * A person's initials in a circle on the brand gradient (design system §5). Hidden from screen
 * readers, because the person's name is always written next to it.
 */
export function Avatar({ initials, size = 'md' }: AvatarProps) {
  return (
    <span className={cx(styles.avatar, styles[size])} aria-hidden="true">
      {initials}
    </span>
  );
}
