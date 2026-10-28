import { cx } from '../cx';
import styles from './Logo.module.css';

export interface LogoProps {
  size?: 'md' | 'lg';
}

/** The product's name is its logo (constitution X); role="img" stops screen readers reading it in pieces. */
export function Logo({ size = 'md' }: LogoProps) {
  return (
    <span role="img" aria-label="SP&A One" className={cx(styles.name, styles[size])}>
      SP<span className={styles.amp}>&amp;</span>A <span className={styles.one}>One</span>
    </span>
  );
}
