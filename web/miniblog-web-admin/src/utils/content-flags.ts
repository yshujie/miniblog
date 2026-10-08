interface ContentEnvironment { VITE_CONTENT_REGISTER_ENABLED?: string; DEV?: boolean }
export function registrationEnabled(env: ContentEnvironment): boolean {
  if (env.VITE_CONTENT_REGISTER_ENABLED !== undefined) return env.VITE_CONTENT_REGISTER_ENABLED === 'true';
  return env.DEV === true;
}
export const contentRegistrationEnabled = registrationEnabled(import.meta.env);
