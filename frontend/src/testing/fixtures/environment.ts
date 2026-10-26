import type { components } from '@/api/generated/schema';

type Environment = components['schemas']['Environment'];

export const ENVIRONMENT_LOCAL: Environment = { environment: 'local', demoSignIn: true };
export const ENVIRONMENT_PRODUCTION: Environment = { environment: 'production', demoSignIn: false };
