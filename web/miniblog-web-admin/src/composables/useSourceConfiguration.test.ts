import { describe, expect, it } from 'vitest';
import { useSourceConfiguration } from './useSourceConfiguration';
import type { SyncSource } from '@/types/notion-sync';
function source(): SyncSource {
  return { source_id: 'source', label: 'Go 文档', module_code: 'go', enabled: true, config_revision: 7, health: 'healthy', catalog_bindings: [], config: {
    title_property_id: 'title', state_property_id: 'state', topic_property_id: 'topic', tags_property_id: 'tags',
    state_option_ids: { draft: 'd', published: 'p', unpublished: 'u', archived: 'a' }
  }};
}
describe('source configuration edit session', () => {
  it('does not submit an unchanged form, including equivalent optional blanks and whitespace', () => {
    const editor = useSourceConfiguration(); editor.open(source());
    editor.form.label = ' Go 文档 '; editor.form.config.description_property_id = ' '; editor.form.config.state_option_ids.published = ' p ';
    expect(editor.update.value).toBeUndefined(); expect(editor.affectsPublication.value).toBe(false);
  });
  it('sends label or enabled changes alone without resubmitting mappings that could hide articles', () => {
    const editor = useSourceConfiguration(); editor.open(source()); editor.form.label = '新名称';
    expect(editor.update.value).toEqual({ label: '新名称', expected_config_revision: 7 }); expect(editor.affectsPublication.value).toBe(false);
    editor.form.label = 'Go 文档'; editor.form.enabled = false;
    expect(editor.update.value).toEqual({ enabled: false, expected_config_revision: 7 }); expect(editor.affectsPublication.value).toBe(false);
  });
  it('marks real mapping and module changes as public-impacting, and clears the impact after undo', () => {
    const editor = useSourceConfiguration(); editor.open(source()); editor.form.config.state_option_ids.published = 'new-option';
    expect(editor.update.value).toEqual({ config: { ...source().config, state_option_ids: { ...source().config.state_option_ids, published: 'new-option' }}, expected_config_revision: 7 });
    expect(editor.affectsPublication.value).toBe(true);
    editor.form.config.state_option_ids.published = 'p'; editor.form.module_code = 'python';
    expect(editor.update.value).toEqual({ module_code: 'python', expected_config_revision: 7 });
    editor.form.module_code = 'go'; expect(editor.update.value).toBeUndefined(); expect(editor.affectsPublication.value).toBe(false);
  });
  it('keeps unsaved inputs and the original revision when refreshed source data changes', () => {
    const incoming = source(); const editor = useSourceConfiguration(); editor.open(incoming); editor.form.config.topic_property_id = 'my-unsaved-topic';
    incoming.label = '远端修改'; incoming.config_revision = 8; incoming.config.topic_property_id = 'remote-topic'; incoming.config.state_option_ids.published = 'remote-option';
    expect(editor.form.label).toBe('Go 文档'); expect(editor.form.config.topic_property_id).toBe('my-unsaved-topic');
    expect(editor.update.value?.expected_config_revision).toBe(7); expect(editor.update.value?.config?.state_option_ids.published).toBe('p');
    expect(editor.source.value?.config.topic_property_id).toBe('topic');
  });
});
