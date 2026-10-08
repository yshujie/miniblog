import { expect, it } from 'vitest';
import { registrationEnabled } from './content-flags';
it('requires explicit production enablement and honors a local rollback switch', () => {
  expect(registrationEnabled({ DEV: false })).toBe(false);
  expect(registrationEnabled({ DEV: false, VITE_CONTENT_REGISTER_ENABLED: 'true' })).toBe(true);
  expect(registrationEnabled({ DEV: true })).toBe(true);
  expect(registrationEnabled({ DEV: true, VITE_CONTENT_REGISTER_ENABLED: 'false' })).toBe(false);
});
