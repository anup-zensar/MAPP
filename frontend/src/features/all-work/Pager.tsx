import { useId } from 'react';
import { Button } from '@/ui/Button/Button';
import { formatWhole } from '@/format/decimal';
import styles from './Pager.module.css';
import { PAGE_SIZES } from './viewState';
import type { PageSize, RowPage } from './viewState';

export interface PagerProps {
  page: RowPage;
  pageSize: PageSize;
  onPage: (page: number) => void;
  onPageSize: (pageSize: PageSize) => void;
}

/**
 * The rows on screen, 50 per page (or 100, 200) (D-115): "Rows 1–50 of 236", Previous, Next and
 * Rows per page. Shown only when there are more rows than the smallest page holds.
 */
export function Pager({ page, pageSize, onPage, onPageSize }: PagerProps) {
  const sizeId = useId();
  if (page.total <= PAGE_SIZES[0]) {
    return null;
  }
  return (
    <nav className={styles.pager} aria-label="Pages">
      <p className={styles.range}>
        Rows {formatWhole(page.from)}–{formatWhole(page.to)} of {formatWhole(page.total)}
      </p>
      <div className={styles.buttons}>
        <Button size="sm" disabled={page.page <= 1} onClick={() => onPage(page.page - 1)}>
          Previous
        </Button>
        <Button
          size="sm"
          disabled={page.page >= page.pageCount}
          onClick={() => onPage(page.page + 1)}
        >
          Next
        </Button>
      </div>
      <div className={styles.size}>
        <label htmlFor={sizeId}>Rows per page</label>
        <select
          id={sizeId}
          value={pageSize}
          onChange={(event) => onPageSize(Number(event.target.value) as PageSize)}
        >
          {PAGE_SIZES.map((size) => (
            <option key={size} value={size}>
              {size}
            </option>
          ))}
        </select>
      </div>
    </nav>
  );
}
