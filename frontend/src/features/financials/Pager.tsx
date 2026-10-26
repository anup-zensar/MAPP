import { ChevronLeft, ChevronRight } from 'lucide-react';
import { useId } from 'react';
import { Button } from '@/ui/Button/Button';
import { formatWhole } from '@/format/decimal';
import { PAGE_SIZES } from './financials-links';
import type { PageSize } from './financials-links';
import styles from './Pager.module.css';

export interface PagerProps {
  page: number;
  pageSize: PageSize;
  total: number;
  onPage: (page: number) => void;
  onPageSize: (pageSize: PageSize) => void;
}

/** "1–50 of 120" with Previous and Next, and 50 / 100 / 200 per page (FR-021). */
export function Pager({ page, pageSize, total, onPage, onPageSize }: PagerProps) {
  const id = useId();
  const pages = Math.max(1, Math.ceil(total / pageSize));
  const first = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const last = Math.min(total, page * pageSize);

  return (
    <nav className={styles.pager} aria-label="Pages">
      <div className={styles.size}>
        <label htmlFor={id}>Per page</label>
        <select
          id={id}
          className={styles.select}
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
      <p className={styles.range} aria-live="polite">
        {`${formatWhole(first)}–${formatWhole(last)} of ${formatWhole(total)}`}
      </p>
      <div className={styles.buttons}>
        <Button
          size="sm"
          icon={<ChevronLeft />}
          disabled={page <= 1}
          onClick={() => onPage(page - 1)}
        >
          Previous
        </Button>
        <Button size="sm" disabled={page >= pages} onClick={() => onPage(page + 1)}>
          Next
          <ChevronRight aria-hidden="true" className={styles.after} />
        </Button>
      </div>
    </nav>
  );
}
