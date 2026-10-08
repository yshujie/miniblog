import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useModuleStore } from '@/stores/module'

export function useReadingNavigation() {
  const router = useRouter()
  const store = useModuleStore()
  onMounted(() => { void store.loadModules().catch(() => {}) })

  // All three entry points go through the page's single loading coordinator.
  const openModule = (code: string) => router.push({ name: 'BlogModule', params: { module: code } })
  async function openFirstModule() {
    const modules = await store.loadModules()
    if (modules.length) await openModule(modules[0].code)
  }
  const reloadModules = () => { void store.loadModules(true).catch(() => {}) }
  return { store, openModule, openFirstModule, reloadModules }
}
