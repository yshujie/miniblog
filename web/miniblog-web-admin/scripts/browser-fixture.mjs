/* Local browser acceptance: every API request is intercepted; no account or database is used. */
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import path from 'node:path';
import { existsSync } from 'node:fs';
import { createRequire } from 'node:module';
const loadPlaywright = createRequire(import.meta.url);
const { chromium } = loadPlaywright(process.env.MINIBLOG_PLAYWRIGHT_MODULE || 'playwright');
const base = process.env.MINIBLOG_ADMIN_URL || 'http://127.0.0.1:8001';
const gateBase = process.env.MINIBLOG_ADMIN_GATE_URL || '';
const output = path.resolve('test-results/browser-fixture');
for (const value of [base, gateBase].filter(Boolean)) {
  assert(['127.0.0.1', 'localhost'].includes(new URL(value).hostname), 'Fixture pages must run on localhost');
}
const moduleRow = { id: 1, code: 'go', title: 'Go 技术笔记', status: 1, sort: 1 };
const sections = [
  { code: 'base', module_code: 'go', title: '基础知识', status: 1, sort: 1 },
  { code: 'advanced', module_code: 'go', title: '进阶实践', status: 1, sort: 2 }
];
const subsections = [{ code: 'runtime', section_code: 'base', title: '运行时', status: 1, sort: 1 }];
const archivedLink = 'https://example.feishu.cn/docx/archived-fixture';
const articles = [{ id: '9007199254740993', id_text: '9007199254740993', title: '已归档的原有文章', author: '原作者', tags: [], external_link: archivedLink, content: '', module: moduleRow, section: sections[0], subsection: subsections[0], status: 'Deleted', pos: 1 }];
const requests = []; const errors = []; let nextID = 9007199254741000n; let uncertainAttempts = 0;
function success(payload) { return { code: 'ok', message: '', payload }; }
async function fixture(route) {
  const request = route.request(); const url = new URL(request.url());
  if (!url.pathname.startsWith('/v1/')) {
    if (!['127.0.0.1', 'localhost'].includes(url.hostname)) return route.abort();
    return route.continue();
  }
  const method = request.method(); if (method === 'OPTIONS') return route.fulfill({ status: 204, headers: { 'access-control-allow-origin': '*', 'access-control-allow-methods': '*', 'access-control-allow-headers': '*' }}); const pathname = url.pathname.replace('/v1/admin', ''); const body = request.postDataJSON();
  requests.push({ method, pathname, body, key: request.headers()['idempotency-key'] });
  const reply = (payload, status = 200) => route.fulfill({ status, contentType: 'application/json', headers: { 'access-control-allow-origin': '*' }, body: JSON.stringify(payload) });
  if (pathname === '/users/myinfo') return reply(success({ user: { roles: ['admin'], nickname: '联调作者', avatar: '', introduction: '本地测试账号' }}));
  if (pathname === '/notion-sync/status') return reply(success({ enabled: false, paused: false, health: 'disabled', pending_count: 0, error_count: 0, sources: [] }));
  if (pathname === '/modules') return reply(success({ modules: [moduleRow] }));
  if (pathname === '/sections/go') return reply(success({ sections }));
  if (pathname.startsWith('/subsections/')) return reply(success({ subsections: pathname.endsWith('/base') ? subsections : [] }));
  if (pathname === '/articles' && method === 'GET') {
    const filters = url.searchParams;
    const matching = articles.filter(article => (!filters.get('module_code') || article.module.code === filters.get('module_code')) && (!filters.get('section_code') || article.section.code === filters.get('section_code')) && (!filters.get('subsection_code') || article.subsection?.code === filters.get('subsection_code')) && (filters.get('direct_only') !== 'true' || !article.subsection?.code) && (!filters.get('title') || article.title.includes(filters.get('title'))) && (filters.get('status') ? article.status === filters.get('status') : true)).sort((a, b) => a.pos - b.pos);
    const page = Number(filters.get('page') || 1); const limit = Number(filters.get('limit') || 20);
    return reply(success({ articles: matching.slice((page - 1) * limit, page * limit), total: matching.length }));
  }
  if (pathname === '/article-sources/preview') {
    if (body.external_link.includes('slow-notion')) await new Promise(resolve => setTimeout(resolve, 1000));
    const existing = articles.find(article => article.external_link === body.external_link);
    return reply(success({ provider: body.external_link.includes('notion') ? 'notion' : 'feishu', canonical_url: body.external_link, title: body.external_link.includes('slow-notion') ? '迟到的 Notion 标题' : '', metadata_status: body.external_link.includes('notion') ? 'resolved' : 'manual_required', existing_article: existing }));
  }
  if (pathname === '/articles/register') {
    if (body.external_link.includes('gate-failure')) return reply({ code: 'ContentRegistrationUnavailable', message: 'fixture migration pending' }, 503);
    if (body.external_link.includes('uncertain') && uncertainAttempts++ === 0) return reply({ code: 'InternalError', message: 'fixture temporary failure' }, 500);
    const existing = articles.find(article => article.external_link === body.external_link);
    if (existing) return reply(success({ outcome: 'already_registered', article: existing }));
    const article = { ...body, id: String(nextID++), module: moduleRow, section: sections.find(item => item.code === body.section_code), subsection: subsections.find(item => item.code === body.subsection_code), content: '', pos: articles.length + 1, status: body.publish ? 'Published' : 'Draft' };
    articles.push(article); return reply(success({ outcome: 'created', article }));
  }
  const match = pathname.match(/^\/articles\/(\d+)(?:\/(restore|archive|publish|unpublish|move))?$/);
  if (match) {
    const article = articles.find(item => item.id === match[1]); assert(article, `Unknown fixture ID ${match[1]}`);
    if (method === 'PUT') {
      if (match[2] === 'restore') article.status = 'Draft';
      else if (match[2] === 'archive') article.status = 'Deleted';
      else if (match[2] === 'publish') article.status = 'Published';
      else if (match[2] === 'unpublish') article.status = 'Unpublished';
      else if (match[2] === 'move') { article.section = sections.find(item => item.code === body.section_code); article.subsection = subsections.find(item => item.code === body.subsection_code); } else Object.assign(article, { title: body.title, author: body.author, tags: body.tags });
    }
    return reply(success({ article }));
  }
  return reply({ code: 'FixtureUnknownRequest', message: `${method} ${pathname}` }, 404);
}
async function prepare(browser, url) {
  const context = await browser.newContext({ viewport: { width: 1440, height: 1100 }});
  await context.route('**/*', fixture);
  await context.addInitScript(() => { localStorage.setItem('Admin-Token', 'fixture-only-not-a-real-token'); });
  const page = await context.newPage(); page.on('pageerror', error => errors.push(error.message));
  await page.goto(`${url}/#/home`);
  try { await page.getByRole('button', { name: '收录文章', exact: true }).waitFor({ timeout: 15000 }); } catch (error) { await page.screenshot({ path: path.join(output, 'failure.png'), fullPage: true, animations: 'disabled' }); console.error({ url: page.url(), text: await page.locator('body').innerText(), errors, requests }); throw error; }
  return { context, page };
}
async function fill(page, link, title) {
  await page.locator('input[data-test="collect-link"]').fill(link);
  await page.locator('input[data-test="collect-title"]').fill(title);
}
async function value(page, name) { return page.locator(`input[data-test="collect-${name}"]`).inputValue(); }
async function snapshot(page, filename) {
  await page.waitForFunction(() => [...document.querySelectorAll('.el-alert')].every(element => getComputedStyle(element).opacity === '1') && !document.querySelector('.fade-transform-enter-active'));
  await page.screenshot({ path: path.join(output, filename), fullPage: true, animations: 'disabled' });
}
(async () => {
  await fs.mkdir(output, { recursive: true });
  const executablePath = process.env.MINIBLOG_CHROMIUM_EXECUTABLE || [chromium.executablePath(), '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'].find(existsSync);
  assert(executablePath, 'Provide MINIBLOG_CHROMIUM_EXECUTABLE or install a local Chromium browser');
  const browser = await chromium.launch({ headless: true, executablePath });
  try {
    const { context, page } = await prepare(browser, base);
    await page.goto(`${base}/#/content/workbench?module_code=go&section_code=base&subsection_code=runtime&collect=1`);
    await page.getByRole('heading', { name: '收录外部文章' }).waitFor();
    assert.equal(await value(page, 'author'), '联调作者');
    await fill(page, 'https://example.feishu.cn/docx/first-fixture', '连续收录第一篇');
    await page.locator('[data-test="collect-continue"]').click();
    await page.getByText('文章已发布', { exact: true }).waitFor();
    assert.equal(await value(page, 'link'), ''); assert.equal(await value(page, 'title'), ''); assert.equal(await value(page, 'author'), '联调作者');
    const first = requests.find(item => item.pathname === '/articles/register'); assert.equal(first.body.section_code, 'base'); assert.equal(first.body.subsection_code, 'runtime'); assert.deepEqual(first.body.tags, []);
    await snapshot(page, '01-publish-and-continue.png');
    await fill(page, 'https://www.notion.so/slow-notion-fixture', '用户手改标题');
    await page.getByRole('button', { name: '重新获取', exact: true }).click();
    await page.getByText('已取得标题建议').waitFor(); assert.equal(await value(page, 'title'), '用户手改标题');
    await snapshot(page, '02-late-title-keeps-manual.png');
    await page.locator('input[data-test="collect-author"]').fill('');
    await page.locator('[data-test="collect-continue"]').click();
    await page.waitForFunction(() => document.querySelector('input[data-test="collect-link"]')?.value === '');
    assert.equal(requests.find(item => item.pathname === '/articles/register' && item.body.title === '用户手改标题').body.author, '');
    assert.equal(await value(page, 'author'), '');
    await fill(page, 'https://example.feishu.cn/docx/gate-failure-fixture', '失败后保留标题');
    await page.locator('[data-test="collect-publish"]').click(); await page.locator('[data-test="collect-error"]').filter({ hasText: '本次输入已保留' }).waitFor();
    assert.equal(await value(page, 'title'), '失败后保留标题'); assert.equal(await page.locator('input[data-test="collect-title"]').isEnabled(), true);
    await snapshot(page, '03-failure-retains-input.png');
    await fill(page, 'https://example.feishu.cn/docx/uncertain-fixture', '原样重试标题');
    await page.locator('[data-test="collect-continue"]').click(); await page.getByRole('button', { name: '原样重试', exact: true }).waitFor();
    assert.equal(await value(page, 'title'), '原样重试标题'); assert.equal(await page.locator('input[data-test="collect-title"]').isDisabled(), true);
    await page.getByRole('button', { name: '原样重试', exact: true }).click();
    await page.waitForFunction(() => document.querySelector('input[data-test="collect-link"]')?.value === '');
    const attempts = requests.filter(item => item.pathname === '/articles/register' && item.body.external_link.includes('uncertain')); assert.equal(attempts.length, 2); assert.equal(attempts[0].key, attempts[1].key); assert.deepEqual(attempts[0].body, attempts[1].body);
    await fill(page, archivedLink, '不应覆盖现有标题');
    await page.getByRole('button', { name: '重新获取', exact: true }).click(); await page.locator('[data-test="collect-duplicate"]').waitFor();
    await page.locator('[data-test="collect-publish"]').click(); await page.locator('[data-test="collect-error"]').filter({ hasText: '未修改现有文章' }).waitFor();
    assert.equal(articles[0].title, '已归档的原有文章'); assert.equal(articles[0].status, 'Deleted'); assert.equal(await value(page, 'title'), '不应覆盖现有标题');
    await page.getByRole('button', { name: '恢复为草稿', exact: true }).click();
    await page.locator('[data-test="collect-duplicate"]').filter({ hasText: '（草稿）' }).waitFor(); assert.equal(articles[0].status, 'Draft');
    await snapshot(page, '04-duplicate-explicit-restore.png');
    await context.close();
    if (gateBase) {
      const gate = await prepare(browser, gateBase);
      assert.equal(await gate.page.getByRole('button', { name: '收录文章', exact: true }).isDisabled(), true);
      assert.equal(await gate.page.locator('.sidebar-container').getByText('目录与收录', { exact: true }).count(), 1);
      await gate.page.goto(`${gateBase}/#/content/workbench`); await gate.page.getByText('收录功能暂未启用，目录和已有文章仍可管理。', { exact: true }).waitFor();
      assert.equal(await gate.page.getByRole('button', { name: '收录文章', exact: true }).isDisabled(), true);
      await snapshot(gate.page, '05-production-gate-disabled.png'); await gate.context.close();
    }
    assert.deepEqual(errors, []);
    const report = { passed: true, cases: ['context/defaults/continue', 'late-title/manual', 'optional-author/empty', '503/input-retention', 'uncertain/frozen-retry', 'duplicate/explicit-restore', ...(gateBase ? ['production/registration-gate'] : [])], apiRequests: requests.length, productionNetworkRequests: 0, screenshotDirectory: output };
    await fs.writeFile(path.join(output, 'report.json'), JSON.stringify(report, null, 2)); console.log(JSON.stringify(report, null, 2));
  } finally { await browser.close(); }
})().catch(error => { console.error(error); process.exitCode = 1; });
