import { onScopeDispose } from 'vue';

// Callers own requests and their cancellation; this helper only owns the visible lifecycle.
export function useVisiblePolling(options: { refresh: () => void; suspend: () => void; resume?: () => void }) {
  let alive = true;
  let interval: ReturnType<typeof setInterval> | undefined;
  let resumeTimer: ReturnType<typeof setTimeout> | undefined;
  const visible = () => alive && document.visibilityState !== 'hidden';
  function start() {
    if (interval === undefined) interval = setInterval(() => { if (visible()) options.refresh(); }, 15000);
  }
  function suspend() {
    if (interval !== undefined) clearInterval(interval);
    if (resumeTimer !== undefined) clearTimeout(resumeTimer);
    interval = undefined; resumeTimer = undefined; options.suspend();
  }
  function resume() {
    if (!visible()) return;
    start();
    if (resumeTimer === undefined) {
      resumeTimer = setTimeout(() => {
        resumeTimer = undefined;
        if (visible()) { (options.resume || options.refresh)(); }
      }, 50);
    }
  }
  function visibilityChanged() { if (visible()) resume(); else suspend(); }
  document.addEventListener('visibilitychange', visibilityChanged);
  window.addEventListener('focus', resume);
  if (visible()) start();
  onScopeDispose(() => {
    alive = false; suspend();
    document.removeEventListener('visibilitychange', visibilityChanged);
    window.removeEventListener('focus', resume);
  });
  return { visible };
}
