import http from '@/util/http'
import { mapModuleSummary, type ModuleSummaryDTO } from './reading-contract'

export async function fetchModules() {
  const { payload } = await http.get<{ modules: ModuleSummaryDTO[] | null }>('/blog/modules')
  return (payload.modules ?? []).map(mapModuleSummary)
}
