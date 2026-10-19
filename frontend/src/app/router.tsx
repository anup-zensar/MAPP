import { createBrowserRouter } from 'react-router';
import type { RouteObject } from 'react-router';
import { ComponentGalleryPage } from '@/features/dev/ComponentGalleryPage';
import { SampleRecordsPage } from '@/features/dev/SampleRecordsPage';
import { HomePage } from '@/features/home/HomePage';
import { ComingSoonPage } from '@/features/placeholders/ComingSoonPage';
import { SignInPage } from '@/features/sign-in/SignInPage';
import { DevOnly } from './DevOnly';
import { COMING_SOON_SECTIONS } from './navigation';
import { NotFoundRoute } from './NotFoundRoute';
import { RequireSignIn } from './RequireSignIn';
import type { PageHandle } from './RequireSignIn';
import { RootLayout } from './RootLayout';
import { RouteErrorPage } from './RouteErrorPage';

/**
 * Every page of the web app and its address (plan "Router"). A data router without loaders or
 * actions: pages load their own data with TanStack Query. Tests build a memory router from these.
 */
export const routes: RouteObject[] = [
  {
    path: '/',
    element: <RootLayout />,
    // Any page that fails to draw shows the error page instead of a blank screen.
    errorElement: <RouteErrorPage />,
    children: [
      { path: '/sign-in', element: <SignInPage /> },
      {
        // Every page in here needs a signed-in person and is drawn inside the app layout. Each
        // names itself in `handle`, for the browser tab while the session is being checked.
        element: <RequireSignIn />,
        children: [
          { index: true, element: <HomePage />, handle: { title: 'Home' } satisfies PageHandle },
          ...COMING_SOON_SECTIONS.map((section) => ({
            path: section.path,
            element: <ComingSoonPage section={section} />,
            handle: { title: section.label } satisfies PageHandle,
          })),
          // The developer pages: not in the navigation, and only where demo sign-in is on — the
          // local and development environments (research R23, D-036).
          {
            path: '/dev/sample-records',
            element: (
              <DevOnly>
                <SampleRecordsPage />
              </DevOnly>
            ),
            handle: { title: 'Sample records' } satisfies PageHandle,
          },
          {
            path: '/dev/components',
            element: (
              <DevOnly>
                <ComponentGalleryPage />
              </DevOnly>
            ),
            handle: { title: 'Shared building blocks' } satisfies PageHandle,
          },
        ],
      },
      // Any other address: "Page not found", for signed-in people and visitors alike (FR-015).
      { path: '*', element: <NotFoundRoute /> },
    ],
  },
];

/** The router of the running app; the address comes from the browser. */
export const router = createBrowserRouter(routes);
