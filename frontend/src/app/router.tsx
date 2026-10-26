import { createBrowserRouter } from 'react-router';
import { routes } from './routes';

/** The router of the running app; the address comes from the browser. */
export const router = createBrowserRouter(routes);
