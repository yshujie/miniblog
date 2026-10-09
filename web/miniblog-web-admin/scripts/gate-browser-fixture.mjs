/* Gate-off application acceptance. Only compiled localhost assets leave the route interceptor. */
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import path from 'node:path';
import http from 'node:http';
import { createRequire } from 'node:module';
const load = createRequire(import.meta.url);
const { chromium } = load(process.env.MINIBLOG_PLAYWRIGHT_MODULE || 'playwright');
const directory = path.resolve(process.env.MINIBLOG_ADMIN_GATE_DIST || '/private/tmp/miniblog-admin-gate-off-dist');
const output = path.resolve(process.env.MINIBLOG_ADMIN_GATE_OUTPUT || 'test-results/gate-browser-fixture');
const moduleRow = { id: '1', code: 'go', title: 'Go 技术笔记', status: 1, sort: 1 };
const section = { code: 'base', module_code: 'go', title: '基础知识', status: 1, sort: 1 };
const article = { id: '9007199254740993', title: '关闭收录开关仍可查看的文章', author: '本机测试作者', tags: ['Go'], external_link: 'https://www.notion.so/fixture-readonly', content: '只读历史快照', module: moduleRow, section, status: 'Published', pos: 1, effective_visibility: true, publication_hold: { held: false }};
const requests = []; const pageErrors = []; const unknownRequests = []; const blockedExternal = []; const cases = []; const screenshots = []; let assetRequests = 0;
const server = http.createServer(async (request, response) => {
  try {
    assert(['GET', 'HEAD'].includes(request.method), 'The local static server accepts only reads');
    const relative = decodeURIComponent(new URL(request.url, 'http://localhost').pathname).replace(/^\/+/, '') || 'index.html';
    const filename = path.resolve(directory, relative); assert(filename.startsWith(`${directory}${path.sep}`), 'Path must stay within the isolated build');
    const content = await fs.readFile(filename); const ext = path.extname(filename);
    response.writeHead(200, { 'content-type': ({ '.html': 'text/html; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.css': 'text/css; charset=utf-8', '.svg': 'image/svg+xml', '.png': 'image/png', '.jpg': 'image/jpeg', '.jpeg': 'image/jpeg' })[ext] || 'application/octet-stream', 'cache-control': 'no-store' });
    response.end(request.method === 'HEAD' ? undefined : content); assetRequests += 1;
  } catch { response.writeHead(404); response.end('Local fixture asset unavailable'); }
});
let base;
const success = payload => ({ code: 'ok', message: '', payload });
async function fixture(route) {
  const request = route.request(); const url = new URL(request.url()); const method = request.method();
  if (!url.pathname.startsWith('/v1/')) {
    if (url.origin === base && ['GET', 'HEAD'].includes(method)) return route.continue();
    blockedExternal.push({ host: url.hostname, method, resource: request.resourceType() }); return route.abort();
  }
  const pathname = url.pathname.replace('/v1/admin', ''); const body = request.postDataJSON(); requests.push({ method, pathname, body });
  const reply = (payload, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(payload) });
  if (pathname === '/users/myinfo' && method === 'GET') return reply(success({ user: { roles: ['admin'], nickname: '门控本机账号', avatar: '', introduction: 'API fixture' }}));
  if (pathname === '/notion-sync/status' && method === 'GET') return reply(success({ enabled: false, paused: false, health: 'disabled', pending_count: 0, blocked_count: 0, error_count: 0, sources: [] }));
  if (pathname === '/modules' && method === 'GET') return reply(success({ modules: [moduleRow] }));
  if (pathname === '/sections/go' && method === 'GET') return reply(success({ sections: [section] }));
  if (pathname.startsWith('/subsections/') && method === 'GET') return reply(success({ subsections: [] }));
  if (pathname === '/sections/base' && method === 'PUT') { assert.deepEqual(Object.keys(body).sort(), ['sort', 'title']); section.title = body.title; section.sort = body.sort; return reply(success({ section })); }
  if (pathname === '/articles' && method === 'GET') return reply(success({ articles: [article], total: 1 }));
  if (pathname === `/articles/${article.id}` && method === 'GET') return reply(success({ article }));
  unknownRequests.push({ method, pathname }); return reply({ code: 'FixtureUnexpectedRequest', message: `Gate fixture rejected ${method} ${pathname}` }, 400);
}
async function settle(page) { await page.waitForFunction(() => !document.querySelector('.fade-transform-enter-active') && [...document.querySelectorAll('.el-loading-mask')].every(element => element.getClientRects().length === 0)); }
async function capture(page, name) { await settle(page); const bounds = await page.evaluate(() => ({ viewport: innerWidth, html: document.documentElement.scrollWidth, body: document.body.scrollWidth })); assert(bounds.html <= bounds.viewport + 1 && bounds.body <= bounds.viewport + 1); await page.screenshot({ path: path.join(output, name), fullPage: true, animations: 'disabled' }); screenshots.push(name); }
async function run() {
  await fs.mkdir(output, { recursive: true });
  await fs.access(path.join(directory, 'index.html'));
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve); });
  const address = server.address(); assert(address && typeof address === 'object'); base = `http://127.0.0.1:${address.port}`;
  const browser = await chromium.launch({ headless: true, executablePath: process.env.MINIBLOG_CHROMIUM_EXECUTABLE || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' });
  let failure;
  try {
    for (const width of [1440, 390]) {
      section.title = '基础知识'; const context = await browser.newContext({ viewport: { width, height: width === 390 ? 844 : 1000 }}); await context.route('**/*', fixture); await context.addInitScript(() => localStorage.setItem('Admin-Token', 'fixture-only-not-a-real-token')); const page = await context.newPage(); page.on('pageerror', error => pageErrors.push(error.message));
      try {
        await page.goto(`${base}/#/content/workbench?module_code=go&section_code=base&collect=1`); await page.getByRole('heading', { name: '基础知识', exact: true }).waitFor();
        assert.equal(await page.getByRole('button', { name: '收录文章', exact: true }).isDisabled(), true); assert.equal(await page.locator('.collect-sheet:visible').count(), 0); await capture(page, `${width}-workbench-disabled.png`); cases.push(`${width}/workbench/disabled/collect-query-does-not-open`);
        await page.getByRole('button', { name: '编辑资料', exact: true }).click(); const drawer = page.locator('.catalog-editor:visible'); await drawer.locator('[data-test="catalog-title"]').fill(`基础资料门控验证 ${width}`); assert.equal(await drawer.locator('[data-test="catalog-code"]').isDisabled(), true); await drawer.getByRole('button', { name: '保存资料', exact: true }).click(); await page.getByRole('heading', { name: `基础资料门控验证 ${width}`, exact: true }).waitFor();
        const update = requests.findLast(item => item.pathname === '/sections/base' && item.method === 'PUT'); assert.deepEqual(update.body, { title: `基础资料门控验证 ${width}`, sort: 1 }); await capture(page, `${width}-directory-edit-still-available.png`); cases.push(`${width}/directory/edit/confirmed-fixture-response/code-preserved`);
        await page.goto(`${base}/#/article/list`); await page.getByRole('heading', { name: '文章库', exact: true }).waitFor(); assert.equal(await page.getByRole('button', { name: '收录文章', exact: true }).isDisabled(), true); await page.locator('.article-count').filter({ hasText: '1' }).waitFor(); await capture(page, `${width}-article-library-disabled.png`); cases.push(`${width}/library/disabled/existing-list-available`);
        await page.goto(`${base}/#/article/edit/${article.id}`); await page.getByRole('heading', { name: article.title, exact: true }).waitFor(); assert.equal(await page.locator('[data-test="article-title"]').inputValue(), article.title); await capture(page, `${width}-existing-article-visible.png`); cases.push(`${width}/detail/read/string-id-preserved`);
        await page.goto(`${base}/#/article/create?module_code=go&section_code=base&collect=1`); await page.getByRole('heading', { name: '收录外部文章', exact: true }).waitFor(); const opener = page.getByRole('button', { name: '粘贴链接收录', exact: true }); assert.equal(await opener.isDisabled(), true); await opener.evaluate(button => button.click()); assert.equal(await page.locator('.collect-sheet:visible').count(), 0); await capture(page, `${width}-legacy-create-disabled.png`); cases.push(`${width}/legacy-create/disabled/no-drawer/no-register`);
      } catch (error) { await page.screenshot({ path: path.join(output, `${width}-failure.png`), fullPage: true }).catch(() => {}); throw error; } finally { await context.close(); }
    }
    assert.equal(requests.some(item => item.pathname === '/articles/register' || item.pathname === '/article-sources/preview'), false); assert.deepEqual(pageErrors, []); assert.deepEqual(unknownRequests, []);
  } catch (error) { failure = error; } finally { await browser.close(); await new Promise(resolve => server.close(resolve)); }
  const report = { passed: !failure, configuration: { apiRoot: '/v1', contentRegistrationEnabled: false, isolatedBuild: directory }, cases, screenshots, pageErrors, unknownRequests, blockedExternal, productionNetworkRequests: 0, apiRequests: requests.length, registerRequests: requests.filter(item => item.pathname === '/articles/register').length, staticAssetRequests: assetRequests, temporaryServerStopped: !server.listening, ...(failure ? { failure: failure.message } : {}), boundary: '本机编译应用 + API fixture；目录PUT也是内存替身，不涉及真实认证、数据库、Notion或生产。' };
  await fs.writeFile(path.join(output, 'report.json'), JSON.stringify(report, null, 2)); await fs.writeFile(path.join(output, 'api-requests.json'), JSON.stringify(requests, null, 2)); console.log(JSON.stringify(report, null, 2)); if (failure) throw failure;
}
run().catch(error => { if (server.listening) server.close(); console.error(error); process.exitCode = 1; });
