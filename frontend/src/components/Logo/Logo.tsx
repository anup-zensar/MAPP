import { cx } from '@/lib/cx';
import styles from './Logo.module.css';

export interface LogoProps {
  /** md: a 28 px tile (sidebar, top bars); lg: a 32 px tile. */
  size?: 'md' | 'lg';
  /** Writes "NorthStar" next to the mark. */
  withWordmark?: boolean;
}

/**
 * The NorthStar mark: a white four-point star on a navy tile, the same as the favicon (never the
 * Shell pecten, which needs Shell's consent).
 * The tile itself is decoration; without the wordmark, give the link around it a name
 * (aria-label="NorthStar home").
 */
export function Logo({ size = 'md', withWordmark = false }: LogoProps) {
  return (
    <span className={cx(styles.logo, styles[size])}>
      <span className={styles.tile} aria-hidden="true">
        <svg viewBox="0 0 32 32">
          <path d="M16 5.5 18.6 13.4 26.5 16 18.6 18.6 16 26.5 13.4 18.6 5.5 16 13.4 13.4Z" />
        </svg>
      </span>
      {withWordmark && <span className={styles.wordmark}>NorthStar</span>}
    </span>
  );
}
