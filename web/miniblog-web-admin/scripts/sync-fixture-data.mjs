import assert from 'node:assert/strict';
import { Buffer } from 'node:buffer';
// All fixture state is in memory; this handler never calls external services.
export function createSyncFixture() {
  const requests = []; const errors = []; const unexpected = [];
  const moduleRow = { id: '1', code: 'go', title: 'Go 技术笔记', status: 1, sort: 1 };
  const section = { id: '2', code: 'base', title: '基础知识', module_code: 'go', status: 1, sort: 1 };
  const article = { id: '9007199254741993', id_text: '9007199254741993', title: 'Notion 托管标题', author: '本地作者', tags: ['Go'], external_link: 'https://example.invalid/captured-original', reading_url: 'http://127.0.0.1:8099/frame', content: '保留的历史正文', module: moduleRow, section, status: 'Published', pos: 1, management: { mode: 'notion_sync', source_id: 'source', managed_fields: ['title', 'catalog', 'tags', 'link', 'status'] }, allowed_actions: ['view_source', 'edit_local_fields', 'reorder', 'hold'], publication_hold: { held: false }, effective_visibility: true };
  const avatar = 'data:image/svg+xml;base64,' + Buffer.from('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40"><rect width="40" height="40" fill="#409eff"/><circle cx="20" cy="14" r="7" fill="white"/><path d="M7 38c0-16 26-16 26 0" fill="white"/></svg>').toString('base64');
  const config = { title_property_id: 'title', state_property_id: 'state', topic_property_id: 'topic', tags_property_id: 'tags', state_option_ids: { draft: 'a', published: 'b', unpublished: 'c', archived: 'd' }};
  const source = { source_id: 'source', label: 'Notion 来源', module_code: 'go', enabled: true, config_revision: 2, config, health: 'healthy', catalog_bindings: [{ id: '9007199254740993', source_id: 'source', option_id: 'topic-id', option_name: '基础主题', status: 'conflict', reason: '同名章节，需要确认绑定' }] };
  const status = { enabled: true, paused: false, source_writes_paused: false, health: 'healthy', pending_count: 1, blocked_count: 0, error_count: 0, sources: [source] };
  const run = { run_id: 'fixture-run', mode: 'dry_run', status: 'completed', phase: 'completed', finished_at: '2026-10-08T00:00:00Z', counts: { seen: 2, created: 0, updated: 1, unchanged: 0, blocked: 1, failed: 0, pending: 1, frozen: 0 }};
  const bootstrapRun = { ...run, run_id: 'bootstrap-preview-fixture', mode: 'bootstrap_preview' };
  const bootstrapCandidate = { page_id: 'historical-page', source_id: 'source', title: 'Notion 来源标题', topic: '基础主题', notion_state: '', new_page: false, match_method: 'known_reading_url', requires_legacy_alias: true, public_condition: 'public_url_available', proposed_section_code: 'base', proposed_section_title: '基础知识', placement_change: '目标主题章节直属', candidate_article_ids: [article.id], title_hint_article_ids: ['9007199254742993'], article_id: article.id, local_state: 'published', local_title: article.title, local_section_code: 'base', local_subsection_code: 'legacy-subsection', reason: 'historical_match_requires_confirmation' };
  let failAuthor = true; let needsRevalidation = false;
  const success = payload => ({ code: 'ok', message: '', payload });
  const pageResult = items => ({ items, total: items.length, page: 1, limit: 20 });
  async function fixture(route) {
    const req = route.request(); const url = new URL(req.url());
    if (!['127.0.0.1', 'localhost'].includes(url.hostname)) { unexpected.push(url.origin); return route.abort(); }
    const reply = (payload, code = 200) => route.fulfill({ status: code, contentType: 'application/json', headers: { 'access-control-allow-origin': '*', 'access-control-allow-headers': '*', 'access-control-allow-methods': '*' }, body: JSON.stringify(payload) });
    if (req.method() === 'OPTIONS') return route.fulfill({ status: 204, headers: { 'access-control-allow-origin': '*', 'access-control-allow-headers': '*', 'access-control-allow-methods': '*' }});
    if (url.pathname === '/frame') return route.fulfill({ contentType: 'text/html; charset=utf-8', body: '<meta charset="utf-8"><h1>本地文章正文演示</h1><p>此页面不连接 Notion。</p>' });
    const apiPath = url.pathname.replace(/^\/api\/v1/, '/v1');
    if (!apiPath.startsWith('/v1/')) return route.continue();
    const pathname = apiPath.replace('/v1/admin', '').replace('/v1', ''); const method = req.method(); const data = req.postDataJSON(); requests.push({ pathname, method, data });
    if (pathname === '/users/myinfo') return reply(success({ user: { roles: ['admin'], nickname: '测试作者', avatar, introduction: '' }}));
    if (pathname === '/modules') return reply(success({ modules: [moduleRow] }));
    if (pathname === '/sections/go') return reply(success({ sections: [section] }));
    if (pathname.startsWith('/subsections/')) return reply(success({ subsections: [] }));
    if (pathname === '/notion-sync/status') return reply(success(status));
    if (pathname === '/notion-sync/sources') return reply(success(pageResult([source])));
    if (pathname === '/notion-sync/pages') return reply(success(pageResult([{ page_id: 'pending-page', title: '等待历史审核', source_id: 'source', desired_state: '', local_state: null, needs_revalidation: false, effective_visibility: false, visibility_reason: 'historical_match_requires_confirmation', management_state: 'baseline_pending', publication_hold: false, page_url: '' }, { page_id: 'managed-page', article_id: article.id, title: article.title, source_id: 'source', desired_state: 'published', local_state: 'published', needs_revalidation: needsRevalidation, effective_visibility: article.effective_visibility, visibility_reason: article.publication_hold.held ? 'publication_hold' : needsRevalidation ? 'needs_revalidation' : '', management_state: 'managed', publication_hold: article.publication_hold.held, page_url: '' }])));
    if (pathname === '/notion-sync/runs' && method === 'GET') return reply(success(pageResult([run, bootstrapRun])));
    if (pathname === '/notion-sync/runs' && method === 'POST') { assert(['dry_run', 'sync'].includes(data.mode)); run.mode = data.mode; if (data.mode === 'sync') { needsRevalidation = false; article.effective_visibility = !article.publication_hold.held; status.blocked_count = article.publication_hold.held ? 1 : 0; } return reply(success({ run_id: run.run_id }), 202); }
    if (pathname === '/notion-sync/runs/bootstrap-preview-fixture') return reply(success(bootstrapRun));
    if (pathname === '/notion-sync/runs/bootstrap-preview-fixture/items') return reply(success(pageResult([{ item_id: 'bootstrap:historical-page', page_id: bootstrapCandidate.page_id, outcome: 'bootstrap_preview', reason: bootstrapCandidate.reason, after: { bootstrap_preview: bootstrapCandidate }}])));
    if (pathname === '/notion-sync/runs/fixture-run') return reply(success(run));
    if (pathname === '/notion-sync/runs/fixture-run/items') return reply(success(pageResult([{ item_id: 'article:pending-page', page_id: 'pending-page', outcome: 'blocked', reason: '等待基线审核', before: { title: '旧标题' }, after: { title: '新标题' }}, { item_id: 'catalog:source:one', page_id: '', outcome: 'created', reason: '新空主题 A', after: { option_id: 'empty-a', section_code: 'empty-a' }}, { item_id: 'catalog:source:two', page_id: '', outcome: 'blocked', reason: '新空主题 B：同名章节冲突', after: { option_id: 'empty-b' }}])));
    if (pathname === '/notion-sync/control') { Object.assign(status, data); return reply(success(status)); }
    if (pathname === '/notion-sync/sources/source' && method === 'PATCH') return reply({ code: 'conflict', message: '源配置已变化，请刷新' }, 409);
    if (pathname.startsWith('/notion-sync/catalog-bindings/')) { assert.equal(data.section_code, 'base'); Object.assign(source.catalog_bindings[0], { section_code: data.section_code, status: 'bound', reason: '' }); return reply(success(source.catalog_bindings[0])); }
    if (pathname === '/articles') return reply(success({ articles: [article], total: 1 }));
    if (pathname === `/articles/${article.id}`) return reply(success({ article }));
    if (pathname === `/articles/${article.id}/local-fields`) {
      assert.equal(method, 'PATCH'); assert.deepEqual(Object.keys(data), ['author']);
      if (failAuthor) { failAuthor = false; return reply({ code: 'failed', message: '本地作者保存暂时失败' }, 503); }
      article.author = data.author; return reply(success({ article }));
    }
    if (pathname === `/articles/${article.id}/publication-hold`) { article.publication_hold = data; needsRevalidation = !data.held; article.effective_visibility = false; status.blocked_count = 1; article.allowed_actions = ['view_source', 'edit_local_fields', 'reorder', data.held ? 'release_hold' : 'hold']; return reply(success({ article })); }
    const publicArticle = { id: article.id, title: article.title, module_code: 'go', section_code: 'base', external_link: article.external_link, reading_url: article.reading_url, author: article.author };
    if (pathname === '/blog/modules') return reply(success({ modules: [moduleRow] }));
    if (pathname === '/blog/articleDetail') return article.publication_hold.held || needsRevalidation ? reply({ code: 'not_found', message: '已下架' }, 404) : reply(success({ article_detail: publicArticle }));
    if (pathname === '/blog/moduleDetail') return reply(success({ module_detail: { ...moduleRow, sections: [{ ...section, articles: article.publication_hold.held ? [] : [publicArticle], subsections: [] }] }}));
    unexpected.push(`${method} ${pathname}`); return reply({ code: 'fixture_unknown', message: pathname }, 404);
  }
  return { fixture, requests, errors, unexpected, article, run, source, status };
}
