import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import BootstrapPreviewList from './BootstrapPreviewList.vue';
import type { BootstrapCandidate, SyncItem } from '@/types/notion-sync';
function candidate(fields: Partial<BootstrapCandidate> = {}): BootstrapCandidate {
  return { page_id: 'page', source_id: 'source', title: 'Notion 标题', topic: '新主题', notion_state: 'published', new_page: false, match_method: 'known_reading_url', requires_legacy_alias: true, public_condition: 'public_url_missing', proposed_section_code: 'new-section', proposed_section_title: '新主题', placement_change: '目标主题章节直属', candidate_article_ids: ['9007199254740993'], title_hint_article_ids: [], article_id: '9007199254740993', local_title: '历史标题', local_state: 'unpublished', local_section_code: 'old-section', local_subsection_code: 'old-subsection', ...fields };
}
function open(items: SyncItem[]) { return mount(BootstrapPreviewList, { props: { items, sourceLabels: { source: 'Go 文档' }, directoryLabels: { 'old-subsection': 'Go / 旧主题 / 历史子章节', 'new-section': 'Go / 新主题' }}, global: { stubs: { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' }}}}); }
describe('saved historical baseline review', () => {
  it('shows source versus local status, exact historical identity, placement and public restrictions without a write action', () => {
    const view = open([{ item_id: 'audit', page_id: 'page', outcome: 'preview', after: { bootstrap_preview: candidate() }}]);
    expect(view.text()).toContain('Notion 标题'); expect(view.text()).toContain('历史标题'); expect(view.text()).toContain('来源期望状态已发布'); expect(view.text()).toContain('本站状态已下架');
    expect(view.text()).toContain('Go / 旧主题 / 历史子章节'); expect(view.text()).toContain('Go / 新主题'); expect(view.text()).toContain('目标主题章节直属');
    expect(view.text()).toContain('需要额外确认旧阅读链接'); expect(view.text()).toContain('来源没有公开阅读链接');
    expect(view.findAll('a').every(link => link.attributes('href') === '/article/edit/9007199254740993')).toBe(true);
    expect(view.text()).toContain('此页面不会写入 Notion 或直接接管文章'); expect(view.find('button').exists()).toBe(false); expect(view.find('pre').exists()).toBe(false); view.unmount();
  });
  it('separates same-title hints from trusted candidates and preserves unknown diagnostics', () => {
    const view = open([{ page_id: 'ambiguous', outcome: 'blocked', after: { bootstrap_preview: candidate({ article_id: undefined, match_method: 'ambiguous', title_hint_article_ids: ['9007199254740995'], reason: 'custom_diagnostic' }) }}]);
    expect(view.text()).toContain('多个候选或身份冲突，需要人工核对'); expect(view.text()).toContain('仅同名参考，不作为自动关联依据'); expect(view.text()).toContain('custom_diagnostic');
    expect(view.find('a[href="/article/edit/9007199254740995"]').exists()).toBe(true); view.unmount();
  });
  it('does not invent a complete comparison for old audit rows without a saved candidate', () => {
    const view = open([{ page_id: 'old', outcome: 'preview', reason: 'historical_match_requires_confirmation', after: { legacy: true }}]);
    expect(view.text()).toContain('旧审核记录没有完整对照清单'); expect(view.text()).toContain('等待审核当前博客状态'); expect(view.text()).not.toContain('来源期望状态'); view.unmount();
  });
});
