import { describe, expect, it, vi } from 'vitest';
import request from '@/utils/request';
import { logout } from './auth';
vi.mock('@/utils/request', () => ({ default: vi.fn().mockResolvedValue({ message: 'ok' }) }));
describe('logout JSON contract', () => {
  it('sends a JSON object accepted by the existing Go controller', async () => {
    await logout(); expect(request).toHaveBeenCalledWith({ url: '/auth/logout', method: 'post', data: {}});
  });
});
