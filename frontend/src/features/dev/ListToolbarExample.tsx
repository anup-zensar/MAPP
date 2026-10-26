import { useState } from 'react';
import { Card } from '@/ui/Card/Card';
import { FilterChips } from '@/ui/FilterChips/FilterChips';
import { SearchField } from '@/ui/SearchField/SearchField';
import { SegmentedFilter } from '@/ui/SegmentedFilter/SegmentedFilter';
import styles from './ComponentGalleryPage.module.css';

export function ListToolbarExample() {
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState<'active' | 'inactive' | 'all'>('active');
  const [chips, setChips] = useState([
    { key: 'status', label: 'Status: Draft' },
    { key: 'year', label: 'Year: 2026' },
  ]);
  return (
    <Card title="SearchField, SegmentedFilter and FilterChips">
      <div className={styles.stack}>
        <SearchField
          label="Search models"
          placeholder="Search name"
          value={search}
          onChange={setSearch}
        />
        <SegmentedFilter
          label="Status"
          value={status}
          onChange={setStatus}
          options={[
            { value: 'active', label: 'Active', count: 5 },
            { value: 'inactive', label: 'Inactive', count: 1 },
            { value: 'all', label: 'All', count: 6 },
          ]}
        />
        <FilterChips
          chips={chips}
          onRemove={(key) => setChips(chips.filter((chip) => chip.key !== key))}
          onClearAll={() => setChips([])}
        />
      </div>
    </Card>
  );
}
