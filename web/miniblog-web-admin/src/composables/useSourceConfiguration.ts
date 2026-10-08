import { computed, reactive, ref } from 'vue';
import type { SourceUpdate, SyncConfig, SyncSource } from '@/types/notion-sync';
const states = ['draft', 'published', 'unpublished', 'archived'];
function normalizedConfig(value?: SyncConfig): SyncConfig {
  const config: SyncConfig = {
    title_property_id: (value?.title_property_id || '').trim(),
    state_property_id: (value?.state_property_id || '').trim(),
    topic_property_id: (value?.topic_property_id || '').trim(),
    tags_property_id: (value?.tags_property_id || '').trim(),
    state_option_ids: Object.fromEntries(states.map(state => [state, (value?.state_option_ids?.[state] || '').trim()]))
  };
  for (const key of ['description_property_id', 'difficulty_property_id', 'date_property_id'] as const) {
    const field = value?.[key]?.trim();
    if (field) config[key] = field;
  }
  return config;
}
export function useSourceConfiguration() {
  const source = ref<SyncSource>();
  const form = reactive({ label: '', module_code: '', enabled: false, config: normalizedConfig() });
  function open(value: SyncSource) {
    // Keep the revision and baseline from the edit session; health refreshes must not replace drafts.
    source.value = { ...value, config: normalizedConfig(value.config), catalog_bindings: (value.catalog_bindings || []).map(binding => ({ ...binding })) };
    Object.assign(form, { label: value.label, module_code: value.module_code, enabled: value.enabled, config: normalizedConfig(value.config) });
  }
  const update = computed<SourceUpdate | undefined>(() => {
    const baseline = source.value;
    if (!baseline) return undefined;
    const changes: SourceUpdate = {};
    if (form.label.trim() !== baseline.label.trim()) changes.label = form.label.trim();
    if (form.module_code.trim() !== baseline.module_code.trim()) changes.module_code = form.module_code.trim();
    if (form.enabled !== baseline.enabled) changes.enabled = form.enabled;
    const config = normalizedConfig(form.config);
    if (JSON.stringify(config) !== JSON.stringify(normalizedConfig(baseline.config))) changes.config = config;
    return Object.keys(changes).length ? { ...changes, expected_config_revision: baseline.config_revision } : undefined;
  });
  const affectsPublication = computed(() => Boolean(update.value && ('module_code' in update.value || 'config' in update.value)));
  return { source, form, open, update, affectsPublication };
}
