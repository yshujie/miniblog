/* New admin UI -> real local Go controller/biz/store -> disposable MySQL. */
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import path from 'node:path';
import { createRequire } from 'node:module';
const require = createRequire(import.meta.url);
const { chromium } = require(process.env.MINIBLOG_PLAYWRIGHT_MODULE || 'playwright');
const metadata = JSON.parse(await fs.readFile(process.env.MINIBLOG_ADMIN_READY || '/tmp/miniblog-admin-ui-real-20261009/ready.json', 'utf8'));
const base = metadata.url;
assert.equal(new URL(base).hostname, '127.0.0.1');
assert.match(metadata.database, /^miniblog_refactor_test_admin_ui_/);
const output = path.resolve(process.env.MINIBLOG_ADMIN_REAL_OUTPUT || '/tmp/miniblog-admin-ui-real-browser');
await fs.mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, executablePath: process.env.MINIBLOG_CHROMIUM_EXECUTABLE || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' });
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }});
const blocked = []; const errors = []; const requests = []; const results = [];
await context.route('**/*', route => { if (new URL(route.request().url()).origin === base) return route.continue(); blocked.push(new URL(route.request().url()).origin); return route.abort(); });
const page = await context.newPage(); page.setDefaultTimeout(15000);
page.on('pageerror', error => errors.push(error.message));
page.on('response', response => { const url = new URL(response.url()); if (url.pathname.startsWith('/v1/')) requests.push({ method: response.request().method(), path: url.pathname, status: response.status(), ...(url.pathname.startsWith('/v1/auth/') ? {} : { query: url.search, body: response.request().postDataJSON() }) }); });
const exact = async response => JSON.parse((await response.text()).replace(/"id":(\d{15,})/g, '"id":"$1"'));
const apiResponse = (method, pathname) => page.waitForResponse(response => response.request().method() === method && new URL(response.url()).pathname === pathname);
async function go(route) { await page.goto(base + '/#' + route); await page.locator('main.app-main').waitFor(); }
async function screenshot(name) { await page.screenshot({ path: path.join(output, name + '.png'), fullPage: true, animations: 'disabled' }); }
const resume = process.env.MINIBLOG_ADMIN_REAL_RESUME === '1';
if (resume) { const previous = JSON.parse(await fs.readFile(path.join(output, 'report.json'), 'utf8')); assert.equal(previous.database, metadata.database); results.push(...previous.cases.filter(item => item.passed)); requests.push(...previous.apiRequests); await fs.copyFile(path.join(output, 'report.json'), path.join(output, 'pre-resume-report.json')); }
async function check(name, work) { if (results.some(item => item.name === name && item.passed)) return; try { await work(); results.push({ name, passed: true }); } catch (error) { results.push({ name, passed: false, error: String(error) }); await screenshot('failure-' + name); throw error; } }
const unique = Date.now().toString(36);
let collectedID;
try {
  if (resume) { await page.goto(base + '/#/content/sync?tab=pages#state'); await page.getByRole('heading', { name: '登录内容工作台' }).waitFor(); await page.locator('input[autocomplete=username]').fill('fixtureadmin'); await page.locator('input[autocomplete=current-password]').fill('LocalFixture12'); await page.getByRole('button', { name: '登录', exact: true }).click(); await page.locator('main.app-main').waitFor(); }
  await check('login-cold-deep-link-query-hash-large-id', async () => {
    await page.goto(base + '/#/article/edit/' + metadata.manual_large_id + '?from=real#context');
    await page.getByRole('heading', { name: '登录内容工作台' }).waitFor();
    await page.locator('input[autocomplete=username]').fill('fixtureadmin');
    await page.locator('input[autocomplete=current-password]').fill('LocalFixture12');
    const login = apiResponse('POST', '/v1/auth/login'); await page.getByRole('button', { name: '登录', exact: true }).click(); assert.equal((await login).status(), 200);
    await page.locator('[data-test=article-title]').waitFor();
    assert.equal(new URL(page.url()).hash, '#/article/edit/' + metadata.manual_large_id + '?from=real#context');
    await screenshot('01-real-manual-large-id');
  });
  await check('manual-save-and-dirty-navigation', async () => {
    await page.locator('[data-test=article-title]').fill('真实接口界面修改 ' + unique); await page.locator('[data-test=article-author]').fill('UI接口作者');
    const saved = apiResponse('PUT', '/v1/admin/articles/' + metadata.manual_large_id); await page.getByRole('button', { name: '保存资料', exact: true }).click(); assert.equal((await saved).status(), 200);
    await page.getByText('文章资料已保存，发布状态保持', { exact: true }).waitFor(); assert.match(await page.locator('.publication-card').innerText(), /已发布/);
    await page.locator('[data-test=article-author]').fill('尚未保存的作者'); await page.locator('.admin-rail').getByRole('link', { name: '文章库', exact: true }).click();
    const confirm = page.getByRole('dialog', { name: '未保存修改' }); await confirm.getByRole('button', { name: '继续编辑', exact: true }).click();
    assert.equal(await page.locator('[data-test=article-author]').inputValue(), '尚未保存的作者');
    await page.locator('[data-test=article-author]').fill('UI接口作者');
  });
  await check('manual-state-transitions', async () => {
    let response = apiResponse('PUT', '/v1/admin/articles/' + metadata.manual_large_id + '/unpublish'); await page.getByRole('button', { name: '下架文章', exact: true }).click(); assert.equal((await response).status(), 200); await page.getByRole('button', { name: '发布文章', exact: true }).waitFor();
    response = apiResponse('PUT', '/v1/admin/articles/' + metadata.manual_large_id + '/publish'); await page.getByRole('button', { name: '发布文章', exact: true }).click(); assert.equal((await response).status(), 200); await page.getByRole('button', { name: '下架文章', exact: true }).waitFor();
  });
  await check('catalog-context-create-subsection', async () => {
    await go('/content/workbench?module_code=m1&section_code=s1'); await page.getByRole('button', { name: '新增子章节', exact: true }).waitFor(); await page.getByRole('button', { name: '新增子章节', exact: true }).click();
    const drawer = page.getByRole('dialog', { name: '新增子章节' }); await drawer.locator('[data-test=catalog-title]').fill('UI接口子章节 ' + unique); await drawer.locator('[data-test=catalog-code]').fill('ui-' + unique);
    const created = apiResponse('POST', '/v1/admin/subsections'); await drawer.getByRole('button', { name: '创建目录', exact: true }).click(); assert.equal((await created).status(), 200); await drawer.waitFor({ state: 'hidden' });
    await page.locator('.context-panel').getByRole('heading', { name: 'UI接口子章节 ' + unique, exact: true }).waitFor(); await screenshot('02-real-catalog');
  });
  await check('collect-continue-and-duplicate-does-not-overwrite', async () => {
    await go('/content/workbench?module_code=m1&section_code=s1&subsection_code=sub1&collect=1'); const drawer = page.getByRole('dialog', { name: '收录外部文章' }); await drawer.waitFor();
    await drawer.locator('[data-test=collect-link]').fill('https://example.invalid/ui/' + unique); await drawer.locator('[data-test=collect-title]').fill('真实接口收录 ' + unique);
    const originalAuthor = await drawer.locator('[data-test=collect-author]').inputValue();
    let response = apiResponse('POST', '/v1/admin/articles/register'); await drawer.locator('[data-test=collect-continue]').click(); const first = await exact(await response); assert.equal(first.code, 'ok'); assert.equal(first.payload.outcome, 'created'); collectedID = first.payload.article.id_text || first.payload.article.id;
    await page.waitForFunction(() => document.querySelector('[data-test=collect-link]')?.value === ''); assert.equal(await drawer.locator('[data-test=collect-author]').inputValue(), originalAuthor); assert.equal(await drawer.locator('[data-test=collect-link]').isEnabled(), true);
    await drawer.locator('[data-test=collect-link]').fill('https://example.invalid/ui/' + unique); await drawer.locator('[data-test=collect-title]').fill('不应覆盖原有标题');
    response = apiResponse('POST', '/v1/admin/articles/register'); await drawer.locator('[data-test=collect-publish]').click(); const duplicate = await exact(await response); assert.equal(duplicate.payload.outcome, 'already_registered'); assert.equal(duplicate.payload.article.title, '真实接口收录 ' + unique); assert.equal(duplicate.payload.article.id_text || duplicate.payload.article.id, collectedID);
    await screenshot('03-real-duplicate'); await drawer.getByRole('button', { name: '关闭', exact: true }).click(); const confirmation = page.getByRole('dialog', { name: '关闭收录' }); await confirmation.getByRole('button', { name: '确定', exact: true }).click(); await drawer.waitFor({ state: 'hidden' });
  });
  await check('article-library-real-server-filter', async () => {
    await go('/article/list'); await page.getByRole('heading', { name: '文章库', exact: true }).waitFor(); await page.getByRole('textbox', { name: '搜索文章标题' }).fill('真实接口收录 ' + unique);
    const response = page.waitForResponse(value => value.request().method() === 'GET' && new URL(value.url()).pathname === '/v1/admin/articles' && new URL(value.url()).searchParams.get('title') === '真实接口收录 ' + unique); await page.getByRole('button', { name: '查询', exact: true }).click(); const data = await exact(await response); assert.equal(data.payload.total, 1); assert.equal(data.payload.articles[0].id_text || data.payload.articles[0].id, collectedID); await screenshot('04-real-library');
  });
  await check('managed-readonly-and-local-author-patch', async () => {
    await go('/article/edit/' + metadata.managed_id); await page.locator('[data-test=source-title]').waitFor(); assert.equal(await page.locator('[data-test=article-title]').count(), 0); assert.equal(await page.getByRole('button', { name: '发布文章', exact: true }).count(), 0);
    await page.locator('[data-test=article-author]').fill('真实UI本地作者'); const response = apiResponse('PATCH', '/v1/admin/articles/' + metadata.managed_id + '/local-fields'); await page.getByRole('button', { name: '保存本地信息', exact: true }).click(); const saved = await response; assert.equal(saved.status(), 200); assert.deepEqual(saved.request().postDataJSON(), { author: '真实UI本地作者' }); await page.getByText('本地作者信息已保存', { exact: true }).waitFor(); await screenshot('05-real-managed-detail');
  });
  await check('local-hold-release-awaits-revalidation', async () => {
    await page.getByRole('button', { name: '紧急下架', exact: true }).click(); const hold = page.getByRole('dialog', { name: '紧急下架' }); await hold.locator('textarea').fill('本机接口验收'); let response = apiResponse('PUT', '/v1/admin/articles/' + metadata.managed_id + '/publication-hold'); await hold.getByRole('button', { name: '确认下架', exact: true }).click(); assert.equal((await response).status(), 200); await hold.waitFor({ state: 'hidden' });
    assert.equal((await context.request.get(base + '/v1/blog/articleDetail?article_id=' + metadata.managed_id)).status(), 404);
    await page.getByRole('button', { name: '解除本地下架', exact: true }).click(); const release = page.getByRole('dialog', { name: '解除本地下架' }); response = apiResponse('PUT', '/v1/admin/articles/' + metadata.managed_id + '/publication-hold'); await release.getByRole('button', { name: '确认解除', exact: true }).click(); assert.equal((await response).status(), 200); await release.waitFor({ state: 'hidden' });
    assert.equal((await context.request.get(base + '/v1/blog/articleDetail?article_id=' + metadata.managed_id)).status(), 404);
  });
  await check('sync-cold-tab-and-readonly-preview', async () => {
    await go('/content/sync?tab=pages#state'); await page.getByRole('tab', { name: '页面状态', selected: true }).waitFor(); assert.equal(new URL(page.url()).hash, '#/content/sync?tab=pages#state');
    const response = apiResponse('POST', '/v1/admin/notion-sync/runs'); await page.getByRole('button', { name: '运行预览', exact: true }).click(); const created = await response; assert.equal(created.status(), 202); assert.deepEqual(created.request().postDataJSON(), { mode: 'dry_run' });
    const run = page.getByRole('dialog', { name: '同步运行详情' }); await run.waitFor(); await run.getByText(/预览不会写入文章或目录/).waitFor(); await run.locator('.el-tag').filter({ hasText: /^完成$/ }).waitFor({ timeout: 45000 }); assert.equal((await context.request.get(base + '/v1/blog/articleDetail?article_id=' + metadata.managed_id)).status(), 404); await screenshot('06-real-dry-run'); await run.locator('.el-drawer__close-btn').click(); await run.waitFor({ state: 'hidden' });
  });
  await check('fresh-sync-revalidates-released-article', async () => {
    await go('/content/sync?tab=overview'); await page.getByRole('button', { name: '手动同步', exact: true }).click();
    const confirm = page.getByRole('dialog', { name: '手动同步' }); const response = apiResponse('POST', '/v1/admin/notion-sync/runs'); await confirm.getByRole('button', { name: /确定|确认/ }).click(); const created = await response; assert.equal(created.status(), 202); assert.deepEqual(created.request().postDataJSON(), { mode: 'sync' });
    const run = page.getByRole('dialog', { name: '同步运行详情' }); await run.waitFor(); await run.locator('.el-tag').filter({ hasText: /^完成$/ }).waitFor({ timeout: 45000 });
    const visible = await context.request.get(base + '/v1/blog/articleDetail?article_id=' + metadata.managed_id); assert.equal(visible.status(), 200); const data = await exact(visible); assert.equal(data.code, 'ok'); await screenshot('07-real-fresh-sync'); await run.locator('.el-drawer__close-btn').click(); await run.waitFor({ state: 'hidden' });
    await go('/content/sync?tab=pages'); await page.getByRole('link', { name: '真实API托管文章', exact: true }).first().waitFor(); await screenshot('08-real-revalidated-pages');
  });
  await check('mobile-navigation-and-real-logout', async () => {
    await page.setViewportSize({ width: 390, height: 844 }); await go('/article/list'); await page.getByRole('heading', { name: '文章库', exact: true }).waitFor(); assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true); await screenshot('09-real-mobile-library');
    await page.getByRole('button', { name: '打开导航', exact: true }).click(); const navigation = page.getByRole('dialog', { name: '内容管理导航' }); await navigation.waitFor(); await navigation.getByRole('button', { name: '查看我的信息', exact: true }).click(); const account = page.getByRole('dialog', { name: '我的信息' }); await account.waitFor(); await account.getByText('接口验收作者', { exact: true }).first().waitFor();
    const response = apiResponse('POST', '/v1/auth/logout'); await account.getByRole('button', { name: '退出登录', exact: true }).click(); const loggedOut = await response; assert.equal(loggedOut.status(), 200); assert.deepEqual(loggedOut.request().postDataJSON(), {}); await page.getByRole('heading', { name: '登录内容工作台' }).waitFor(); assert.equal(await page.evaluate(() => localStorage.getItem('Admin-Token')), null); await screenshot('10-real-login');
  });
  assert.deepEqual(errors, []); assert.deepEqual(blocked, []);
} catch (error) { console.error(String(error)); process.exitCode = 1; }
finally {
  await fs.writeFile(path.join(output, 'report.json'), JSON.stringify({ passed: results.length === 11 && results.every(result => result.passed) && !errors.length && !blocked.length, cases: results, apiRequests: requests, errors, blockedExternalRequests: blocked, scope: metadata.scope, database: metadata.database, realGoAPI: true, realMySQL: true, realNotion: false, productionRouterVerified: false, productionAccess: false, tokenRecorded: false, screenshotDirectory: output }, null, 2));
  console.log(JSON.stringify({ passed: !process.exitCode, cases: results.map(({ name, passed }) => ({ name, passed })), output })); await browser.close();
}
