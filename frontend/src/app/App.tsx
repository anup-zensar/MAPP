import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Tooltip } from 'radix-ui';
import { RouterProvider } from 'react-router/dom';
import { ToastProvider } from '@/components';
import { router } from './router';

/** The one cache of data from the API for the whole app (research R15). */
const queryClient = new QueryClient({
  defaultOptions: {
    // A failed request shows Try again at once, instead of retrying out of sight.
    queries: { retry: false },
  },
});

/** The web app: data from the API, tooltips and messages around the pages of the router. */
export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      {/* Tooltips appear after 200 ms, the delay of the sidebar's icon rail (plan "Layout"). */}
      <Tooltip.Provider delayDuration={200}>
        <ToastProvider>
          <RouterProvider router={router} />
        </ToastProvider>
      </Tooltip.Provider>
    </QueryClientProvider>
  );
}
