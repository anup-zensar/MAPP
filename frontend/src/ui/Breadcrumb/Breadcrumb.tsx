import { ChevronLeft } from 'lucide-react';
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
 * (design system §6.3). Usually given to PageHeader, which shows it above the title; the last
 * item is the current page. On phones only the parent is shown, as "‹ Parent" (Shell).
 */
export function Breadcrumb({ items }: BreadcrumbProps) {
  return (
    <nav aria-label="Breadcrumb" className={styles.breadcrumb}>
      <ol className={styles.list}>
        {items.map((item, index) => {
          const isCurrent = index === items.length - 1;
          const isParent = index === items.length - 2;
          return (
            <li key={`${index}-${item.label}`} className={isParent ? styles.parent : styles.item}>
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
                  {isParent && <ChevronLeft aria-hidden="true" className={styles.back} />}
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
