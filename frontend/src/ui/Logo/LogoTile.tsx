import styles from './Logo.module.css';

/** Stands for the product where its name does not fit; the same artwork as public/favicon.svg. */
export function LogoTile() {
  return (
    <svg className={styles.tile} viewBox="0 0 32 32" aria-hidden="true">
      <rect width="32" height="32" rx="9" />
      <path d="M19 8h-2.7l-4.4 3.1v3l4.1-2.8V24h3z" />
    </svg>
  );
}
