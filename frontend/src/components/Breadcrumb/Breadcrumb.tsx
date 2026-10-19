import { Link } from 'react-router';
import styles from './Breadcrumb.module.css';

/** One step of the trail: a link, except the page you are on. */
export interface BreadcrumbItem {
  label: string;
  /** Where the step leads. The last item is the current page and never a link. */
  to?: string;
}

export interface BreadcrumbProps {
  items: BreadcrumbItem[];
}

/**
 * Where a page sits: "Settings › Models", "Plans › FY2026 Lubricants › Helix launch"
 * (design system §6.3). Placed above the page header; the last item is the current page.
 */
export function Breadcrumb({ items }: BreadcrumbProps) {
  return (
    <nav aria-label="Breadcrumb" className={styles.breadcrumb}>
      <ol className={styles.list}>
        {items.map((item, index) => {
          const isCurrent = index === items.length - 1;
          return (
            <li key={`${index}-${item.label}`} className={styles.item}>
              {index > 0 && (
                <span aria-hidden="true" className={styles.separator}>
                  ›
                </span>
              )}
              {isCurrent || !item.to ? (
                <span aria-current={isCurrent ? 'page' : undefined} className={styles.text}>
                  {item.label}
                </span>
              ) : (
                <Link to={item.to} className={styles.link}>
                  {item.label}
                </Link>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
