export class ApiError extends Error {
  constructor(message: string, public code = 'request_failed', public status = 0) { super(message); this.name = 'ApiError'; }
}
export function errorMessage(error: unknown, fallback = '请求失败'): string { return error instanceof Error ? error.message : fallback; }
export function isUncertain(error: unknown): boolean { return !(error instanceof ApiError) || error.status === 0 || error.status >= 500; }
