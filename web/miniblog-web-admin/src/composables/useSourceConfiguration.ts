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
function cloneSource(value: SyncSource): SyncSource { return { ...value, config: normalizedConfig(value.config), catalog_bindings: (value.catalog_bindings || []).map(binding => ({ ...binding })) }; }
export function useSourceConfiguration() {
  const source = ref<SyncSource>(); const latest = ref<SyncSource>();
  const form = reactive({ label: '', module_code: '', enabled: false, config: normalizedConfig() });
  function open(value: SyncSource) {
    // Keep the revision and baseline from the edit session; health refreshes must not replace drafts.
    source.value = cloneSource(value); latest.value = undefined;
    Object.assign(form, { label: value.label, module_code: value.module_code, enabled: value.enabled, config: normalizedConfig(value.config) });
  }
  function compareLatest(value: SyncSource) { if (value.source_id === source.value?.source_id) latest.value = cloneSource(value); }
  function acceptLatest() { if (latest.value?.source_id === source.value?.source_id) source.value = cloneSource(latest.value!); }
  const needsReview = computed(() => Boolean(latest.value && latest.value.config_revision !== source.value?.config_revision));
  const comparison = computed(() => {
    const baseline = source.value; const remote = latest.value;
    if (!baseline || !remote) return [];
    const rows = [
      { label: '名称', baseline: baseline.label, draft: form.label, latest: remote.label },
      { label: '模块', baseline: baseline.module_code, draft: form.module_code, latest: remote.module_code },
      { label: '来源同步', baseline: baseline.enabled ? '启用' : '停用', draft: form.enabled ? '启用' : '停用', latest: remote.enabled ? '启用' : '停用' }
    ];
    const fields = { title_property_id: '标题属性', state_property_id: '状态属性', topic_property_id: '主题属性', tags_property_id: '标签属性', description_property_id: '简介属性', difficulty_property_id: '难度属性', date_property_id: '日期属性' } as const;
    for (const [field, label] of Object.entries(fields)) {
      const key = field as keyof typeof fields;
      rows.push({ label, baseline: baseline.config[key] || '', draft: form.config[key] || '', latest: remote.config[key] || '' });
    }
    const labels = { draft: '草稿', published: '发布', unpublished: '下架', archived: '归档' };
    for (const state of states) rows.push({ label: `${labels[state as keyof typeof labels]}选项`, baseline: baseline.config.state_option_ids[state] || '', draft: form.config.state_option_ids[state] || '', latest: remote.config.state_option_ids[state] || '' });
    return rows;
  });
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
  return { source, latest, form, open, compareLatest, acceptLatest, comparison, needsReview, update, affectsPublication };
}
