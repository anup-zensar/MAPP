import { useQuery } from '@tanstack/react-query';
import { api, call } from '@/api/client';

/** The demo people offered on the sign-in page; ApiError `notFound` when demo sign-in is off. */
export function useDemoPeople() {
  return useQuery({
    queryKey: ['demo-people'],
    queryFn: () => call(api.GET('/api/v1/auth/demo-people')),
  });
}
