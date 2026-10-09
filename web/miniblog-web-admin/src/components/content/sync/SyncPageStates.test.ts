import { mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { describe, expect, it } from 'vitest';
import SyncPageStates from './SyncPageStates.vue';
import type { SyncPage } from '@/types/notion-sync';
const page = (override: Partial<SyncPage> = {}): SyncPage => ({ page_id: 'page', article_id: '9007199254740993', source_id: 'source', title: '来源页面', desired_state: 'published', status: 'ok', management_state: 'managed', page_url: 'https://example.invalid/page', publication_hold: false, ...override });
const open = (items: SyncPage[]) => mount(SyncPageStates, { props: { items, loading: false }, global: { plugins: [ElementPlus], stubs: { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' }}}});
describe('sync page state presentation', () => {
  it('does not infer publication or visibility from source intent or a reading URL', () => {
    const view = open([page({ local_state: undefined, effective_visibility: undefined }), page({ page_id: 'not-registered', local_state: null, effective_visibility: false })]);
    expect(view.text()).toContain('尚未提供'); expect(view.text()).toContain('尚无本站文章'); expect(view.text()).toContain('尚未核验'); expect(view.text()).toContain('前台不可见');
    expect(view.text()).not.toContain('前台可见');
    expect(view.findAll('a[href="/article/edit/9007199254740993"]').length).toBeGreaterThan(0);
    view.unmount();
  });
  it('shows restrictions and recent failures separately and never renders an unsafe reading URL', () => {
    const view = open([page({ local_state: 'published', effective_visibility: false, visibility_reason: 'needs_revalidation', last_error: 'transport_error', needs_revalidation: true, publication_hold: true, public_url: 'javascript:alert(1)' })]);
    expect(view.text()).toContain('已发布'); expect(view.text()).toContain('前台不可见'); expect(view.text()).toContain('等待重新核验'); expect(view.text()).toContain('本地紧急下架'); expect(view.text()).toContain('最近失败：读取来源失败');
    expect(view.findAll('a').every(link => !link.attributes('href')?.startsWith('javascript:'))).toBe(true);
    expect(view.findAll('a[target="_blank"]')).toHaveLength(0);
    view.unmount();
  });
});
