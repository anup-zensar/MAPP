import { cx } from '../cx';
import styles from './Logo.module.css';

export interface LogoProps {
  size?: 'md' | 'lg';
}

/** The name in its colours, in the surrounding font; role="img" stops screen readers reading it in pieces. */
export function ProductName() {
  return (
    <span role="img" aria-label="SP&A One" className={styles.productName}>
      SP<span className={styles.amp}>&amp;</span>A <span className={styles.one}>One</span>
    </span>
  );
}

export function Logo({ size = 'md' }: LogoProps) {
  return (
    <span className={cx(styles.name, styles[size])}>
      <ProductName />
    </span>
  );
}
