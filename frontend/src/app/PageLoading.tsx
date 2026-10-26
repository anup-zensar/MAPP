import { Skeleton } from '@/ui/Skeleton/Skeleton';

/** Shown while a page's code is on its way: the loading rows a page shows while its data loads. */
export function PageLoading() {
  return <Skeleton lines={4} height="18px" />;
}
