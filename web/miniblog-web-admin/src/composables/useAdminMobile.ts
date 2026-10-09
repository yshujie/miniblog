import { onScopeDispose, ref } from 'vue';
export function useAdminMobile() {
  const query = window.matchMedia('(max-width: 780px)');
  const mobile = ref(query.matches);
  const update = () => { mobile.value = query.matches; };
  query.addEventListener('change', update);
  onScopeDispose(() => query.removeEventListener('change', update));
  return mobile;
}
