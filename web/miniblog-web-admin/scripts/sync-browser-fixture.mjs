/* Offline browser acceptance: localhost pages, intercepted API and iframe fixtures only. */
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import path from 'node:path';
import { createRequire } from 'node:module';
const require = createRequire(import.meta.url);
const { chromium } = require(process.env.MINIBLOG_PLAYWRIGHT_MODULE || 'playwright');
const adminURL = process.env.MINIBLOG_ADMIN_URL || 'http://127.0.0.1:5189';
const readerURL = process.env.MINIBLOG_READER_URL || 'http://127.0.0.1:5188';
for (const url of [adminURL, readerURL]) assert(['127.0.0.1', 'localhost'].includes(new URL(url).hostname));
const output = path.resolve(process.env.MINIBLOG_FIXTURE_OUTPUT || '/private/tmp/miniblog-sync-browser');
const requests = []; const errors = []; const unexpected = [];
const moduleRow = { id: '1', code: 'go', title: 'Go 技术笔记', status: 1, sort: 1 };
const section = { id: '2', code: 'base', title: '基础知识', module_code: 'go', status: 1, sort: 1 };
const article = { id: '9007199254741993', id_text: '9007199254741993', title: 'Notion 托管标题', author: '本地作者', tags: ['Go'], external_link: 'https://example.invalid/captured-original', reading_url: 'http://127.0.0.1:8099/frame', content: '保留的历史正文', module: moduleRow, section, status: 'Published', pos: 1, management: { mode: 'notion_sync', source_id: 'source', managed_fields: ['title', 'catalog', 'tags', 'link', 'status'] }, allowed_actions: ['view_source', 'edit_local_fields', 'reorder', 'hold'], publication_hold: { held: false }, effective_visibility: true };
const config = { title_property_id: 'title', state_property_id: 'state', topic_property_id: 'topic', tags_property_id: 'tags', state_option_ids: { draft: 'a', published: 'b', unpublished: 'c', archived: 'd' }};
const source = { source_id: 'source', label: 'Notion 来源', module_code: 'go', enabled: true, config_revision: 2, config, health: 'healthy', catalog_bindings: [{ id: '9007199254740993', source_id: 'source', option_id: 'topic-id', option_name: '基础主题', status: 'conflict', reason: '同名章节，需要确认绑定' }] };
const status = { enabled: true, paused: false, source_writes_paused: false, health: 'healthy', pending_count: 1, error_count: 1, sources: [source] };
const run = { run_id: 'fixture-run', mode: 'dry_run', status: 'completed', phase: 'completed', finished_at: '2026-10-08T00:00:00Z', counts: { seen: 2, created: 0, updated: 1, unchanged: 0, blocked: 1, failed: 0 }};
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
  if (pathname === '/users/myinfo') return reply(success({ user: { roles: ['admin'], nickname: '测试作者', avatar: '', introduction: '' }}));
  if (pathname === '/modules') return reply(success({ modules: [moduleRow] }));
  if (pathname === '/sections/go') return reply(success({ sections: [section] }));
  if (pathname.startsWith('/subsections/')) return reply(success({ subsections: [] }));
  if (pathname === '/notion-sync/status') return reply(success(status));
  if (pathname === '/notion-sync/sources') return reply(success(pageResult([source])));
  if (pathname === '/notion-sync/pages') return reply(success(pageResult([{ page_id: 'pending-page', title: '等待历史审核', source_id: 'source', desired_state: 'published', management_state: 'baseline_pending', publication_hold: false, page_url: '', publish_block_reason: '等待状态审核' }])));
  if (pathname === '/notion-sync/runs' && method === 'GET') return reply(success(pageResult([run])));
  if (pathname === '/notion-sync/runs' && method === 'POST') { assert(['dry_run', 'sync'].includes(data.mode)); run.mode = data.mode; if (data.mode === 'sync') { needsRevalidation = false; article.effective_visibility = !article.publication_hold.held; } return reply(success({ run_id: run.run_id }), 202); }
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
  if (pathname === `/articles/${article.id}/publication-hold`) { article.publication_hold = data; needsRevalidation = !data.held; article.effective_visibility = false; article.allowed_actions = ['view_source', 'edit_local_fields', 'reorder', data.held ? 'release_hold' : 'hold']; return reply(success({ article })); }
  const publicArticle = { id: article.id, title: article.title, module_code: 'go', section_code: 'base', external_link: article.external_link, reading_url: article.reading_url, author: article.author };
  if (pathname === '/blog/modules') return reply(success({ modules: [moduleRow] }));
  if (pathname === '/blog/articleDetail') return article.publication_hold.held || needsRevalidation ? reply({ code: 'not_found', message: '已下架' }, 404) : reply(success({ article_detail: publicArticle }));
  if (pathname === '/blog/moduleDetail') return reply(success({ module_detail: { ...moduleRow, sections: [{ ...section, articles: article.publication_hold.held ? [] : [publicArticle], subsections: [] }] }}));
  unexpected.push(`${method} ${pathname}`); return reply({ code: 'fixture_unknown', message: pathname }, 404);
}
await fs.mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, executablePath: process.env.MINIBLOG_CHROMIUM_EXECUTABLE || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' });
try {
  const adminContext = await browser.newContext({ viewport: { width: 1440, height: 1000 }}); await adminContext.route('**/*', fixture);
  await adminContext.addInitScript(() => localStorage.setItem('Admin-Token', 'fixture-only'));
  const page = await adminContext.newPage(); page.on('pageerror', error => errors.push(error.message));
  await page.goto(adminURL + '/#/content/sync'); await page.getByRole('heading', { name: 'Notion 自动同步' }).waitFor(); await page.getByText('等待历史审核', { exact: true }).waitFor();
  await page.screenshot({ path: path.join(output, 'sync-overview.png'), fullPage: true, animations: 'disabled' });
  await page.getByRole('button', { name: '运行预览', exact: true }).click(); await page.getByRole('heading', { name: '同步运行详情' }).waitFor(); await page.getByText('等待基线审核', { exact: true }).waitFor(); await page.getByText('新空主题 A', { exact: true }).waitFor(); await page.getByText('新空主题 B：同名章节冲突', { exact: true }).waitFor(); assert.equal(await page.getByText('主题目录', { exact: true }).count(), 2);
  assert.equal(requests.filter(item => item.method === 'POST' && item.pathname === '/notion-sync/runs').length, 1);
  await page.screenshot({ path: path.join(output, 'sync-run-items.png'), fullPage: true, animations: 'disabled' });
  await page.locator('.el-drawer__close-btn').click();
  await page.getByRole('button', { name: '配置', exact: true }).click(); const dialog = page.getByRole('dialog', { name: '来源配置' }); await dialog.waitFor();
  await dialog.locator('.el-form-item').filter({ hasText: '名称' }).locator('input').fill('保留的新配置名称'); await dialog.getByRole('button', { name: '保存配置', exact: true }).click(); await dialog.getByText('源配置已变化，请刷新').waitFor();
  assert.equal(await dialog.locator('.el-form-item').filter({ hasText: '名称' }).locator('input').inputValue(), '保留的新配置名称'); await dialog.getByRole('button', { name: '关闭', exact: true }).click();
  await page.getByRole('button', { name: '调整绑定', exact: true }).click(); const bind = page.getByRole('dialog', { name: '调整主题绑定' }); await bind.locator('.el-select').click(); await page.getByRole('option', { name: '基础知识', exact: true }).click(); await bind.getByRole('button', { name: '保存绑定', exact: true }).click(); await bind.waitFor({ state: 'hidden' });
  await page.goto(adminURL + `/#/article/edit/${article.id}`); await page.getByText('Notion 同步管理', { exact: true }).waitFor();
  const titleInput = page.locator('.el-form-item').filter({ hasText: /^标题/ }).locator('input'); assert.equal(await titleInput.isDisabled(), true);
  const authorInput = page.locator('.el-form-item').filter({ hasText: '作者（可选）' }).locator('input'); await authorInput.fill('浏览器本地作者');
  await page.getByRole('button', { name: '保存本地信息', exact: true }).click(); await page.getByText('本地作者保存暂时失败', { exact: true }).waitFor(); assert.equal(await authorInput.inputValue(), '浏览器本地作者');
  await page.getByRole('button', { name: '保存本地信息', exact: true }).click(); await page.getByText('本地作者信息已保存', { exact: true }).waitFor();
  await page.screenshot({ path: path.join(output, 'managed-article.png'), fullPage: true, animations: 'disabled' });
  const readerContext = await browser.newContext({ viewport: { width: 390, height: 844 }}); await readerContext.route('**/*', fixture);
  const reader = await readerContext.newPage(); reader.on('pageerror', error => errors.push(error.message)); await reader.clock.install();
  await reader.goto(readerURL + `/blog/removed-module/article/${article.id}?from=old#part`); await reader.locator('iframe').waitFor();
  assert.equal(new URL(reader.url()).pathname, `/blog/go/article/${article.id}`); assert.equal(new URL(reader.url()).search, '?from=old'); assert.equal(new URL(reader.url()).hash, '#part');
  assert.equal(await reader.locator('iframe').getAttribute('src'), article.reading_url); await reader.evaluate(() => { window.fixtureFrame = document.querySelector('iframe'); });
  article.title = '自动同步后的新标题'; await reader.clock.runFor(60000); await reader.getByRole('heading', { name: article.title, exact: true }).waitFor();
  assert.equal(await reader.evaluate(() => window.fixtureFrame === document.querySelector('iframe')), true);
  await reader.screenshot({ path: path.join(output, 'reader-refreshed-mobile.png'), fullPage: true, animations: 'disabled' });
  await page.getByRole('button', { name: '紧急下架', exact: true }).click(); const hold = page.getByRole('dialog', { name: '紧急下架' }); await hold.locator('input').fill('浏览器验收下架'); await hold.getByRole('button', { name: /确定|确认/ }).click(); await page.getByText('本地下架状态已更新', { exact: true }).waitFor();
  await reader.clock.runFor(60000); await reader.getByRole('heading', { name: '文章不可用', exact: true }).waitFor(); assert.equal(await reader.locator('iframe').count(), 0);
  await reader.screenshot({ path: path.join(output, 'reader-held-mobile.png'), fullPage: true, animations: 'disabled' });
  await page.getByRole('button', { name: '解除本地下架', exact: true }).click(); const release = page.getByRole('dialog', { name: '解除本地下架' }); await release.getByRole('button', { name: /确定|确认/ }).click(); await page.getByRole('button', { name: '紧急下架', exact: true }).waitFor();
  await reader.getByRole('button', { name: '重试', exact: true }).click(); await reader.getByRole('heading', { name: '文章不可用', exact: true }).waitFor(); assert.equal(await reader.locator('iframe').count(), 0);
  await page.goto(adminURL + '/#/content/sync'); await page.getByRole('button', { name: '手动同步', exact: true }).click(); const confirmSync = page.getByRole('dialog', { name: '手动同步' }); await confirmSync.getByRole('button', { name: /确定|确认/ }).click(); await page.getByRole('heading', { name: '同步运行详情' }).waitFor();
  await reader.getByRole('button', { name: '重试', exact: true }).click(); await reader.locator('iframe').waitFor();
  await page.locator('.el-drawer__close-btn').click(); await page.getByRole('heading', { name: '同步运行详情' }).waitFor({ state: 'hidden' }); await page.waitForFunction(() => !document.querySelector('.el-message'));
  await page.setViewportSize({ width: 390, height: 844 }); await page.goto(adminURL + '/#/content/sync'); await page.getByRole('heading', { name: 'Notion 自动同步' }).waitFor();
  await page.screenshot({ path: path.join(output, 'sync-small-screen.png'), fullPage: true, animations: 'disabled' });
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1), true);
  assert.deepEqual(errors, []); assert.deepEqual(unexpected, []);
  const report = { passed: true, cases: ['sync-health-and-baseline', 'dry-run-and-paged-items', 'distinct-empty-topic-plans', 'config-conflict-retains-input', 'catalog-binding', 'managed-author-PATCH-and-failure', 'historic-moved-URL', 'reading-url', 'mobile-periodic-refresh-stable-iframe', 'hold-removes-iframe', 'release-requires-revalidation', 'admin-small-screen'], requests: requests.length, productionNetworkRequests: 0, screenshotDirectory: output };
  await fs.writeFile(path.join(output, 'report.json'), JSON.stringify(report, null, 2)); console.log(JSON.stringify(report, null, 2));
} catch (error) { console.error(error, { errors, unexpected, requests }); throw error; } finally { await browser.close(); }
