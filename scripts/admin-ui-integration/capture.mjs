/* Final viewport captures use real loopback APIs and synthetic local data. */
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
const output = path.resolve(process.env.MINIBLOG_ADMIN_CAPTURE_OUTPUT || '/tmp/miniblog-admin-review');
await fs.mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, executablePath: process.env.MINIBLOG_CHROMIUM_EXECUTABLE || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' });
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }});
const errors = []; const blocked = []; const requests = []; const screenshots = [];
await context.route('**/*', route => {
  const request = route.request(); const url = new URL(request.url());
  const permitted = url.origin === base && (request.method() === 'GET' || (request.method() === 'POST' && url.pathname === '/v1/auth/login') || (request.method() === 'PATCH' && url.pathname === '/v1/admin/articles/' + metadata.managed_id + '/local-fields'));
  if (permitted) return route.continue(); blocked.push({ method: request.method(), origin: url.origin, path: url.pathname }); return route.abort();
});
const page = await context.newPage(); page.setDefaultTimeout(15000); page.on('pageerror', error => errors.push(error.message));
page.on('response', response => { const url = new URL(response.url()); if (url.pathname.startsWith('/v1/')) requests.push({ path: url.pathname, method: response.request().method(), status: response.status() }); });
async function go(route) { await page.goto(base + '/#' + route); await page.locator('main.app-main').waitFor(); }
async function capture(name) { await page.waitForLoadState('networkidle'); await page.waitForFunction(() => !document.querySelector('#nprogress') && [...document.querySelectorAll('.el-loading-mask')].every(element => element.getClientRects().length === 0 || getComputedStyle(element).opacity === '0') && !document.querySelector('.el-skeleton')); await page.evaluate(() => { document.activeElement?.blur(); window.scrollTo(0, 0); }); await page.waitForFunction(() => scrollY === 0); assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)); await page.screenshot({ path: path.join(output, name + '.png'), fullPage: false, animations: 'disabled' }); screenshots.push(name + '.png'); }
let toast;
try {
  await page.goto(base + '/#/home'); await page.locator('input[autocomplete=username]').fill('fixtureadmin'); await page.locator('input[autocomplete=current-password]').fill('LocalFixture12'); await page.getByRole('button', { name: '登录', exact: true }).click(); await page.getByRole('heading', { name: '工作概览', exact: true }).waitFor();
  await page.locator('.overview-grid').waitFor(); await capture('desktop-overview');
  await go('/content/workbench?module_code=m1&section_code=s1'); await page.locator('.context-panel').getByRole('heading', { name: '接口章节', exact: true }).waitFor(); await page.locator('[data-test=health-summary]').waitFor(); await capture('desktop-catalog');
  await go('/article/list'); await page.getByRole('link', { name: '真实API托管文章', exact: true }).first().waitFor(); await capture('desktop-library');
  await go('/article/edit/' + metadata.managed_id); await page.locator('[data-test=source-title]').waitFor(); await capture('desktop-managed');
  await go('/content/sync?tab=pages'); await page.getByRole('link', { name: '真实API托管文章', exact: true }).first().waitFor(); await capture('desktop-sync-pages');
  await page.setViewportSize({ width: 390, height: 844 }); await go('/article/list'); await page.locator('.article-card').first().waitFor(); await capture('mobile-library');
  await go('/content/sync?tab=overview'); await page.getByRole('heading', { name: '自动同步已开启', exact: true }).waitFor(); await capture('mobile-sync');
  await go('/article/edit/' + metadata.managed_id); await page.locator('[data-test=source-title]').waitFor(); await page.locator('[data-test=article-author]').fill('最终本机提示验收 ' + Date.now().toString(36));
  const response = page.waitForResponse(item => item.request().method() === 'PATCH' && new URL(item.url()).pathname.endsWith('/local-fields'));
  await page.getByRole('button', { name: '保存本地信息', exact: true }).click(); assert.equal((await response).status(), 200);
  await page.getByText('本地作者信息已保存', { exact: true }).waitFor();
  await page.waitForFunction(() => { const element = document.querySelector('.el-message'); if (!element) return false; const box = element.getBoundingClientRect(); return box.y >= 8 && box.y + box.height <= innerHeight; });
  toast = await page.locator('.el-message').evaluate(element => { const box = element.getBoundingClientRect(); const style = getComputedStyle(element); return { position: style.position, display: style.display, x: box.x, y: box.y, width: box.width, height: box.height, viewportWidth: innerWidth, viewportHeight: innerHeight }; });
  assert.equal(toast.position, 'fixed'); assert(toast.x >= 0 && toast.x + toast.width <= toast.viewportWidth && toast.y >= 0 && toast.y + toast.height <= toast.viewportHeight && toast.height >= 24); await page.screenshot({ path: path.join(output, 'mobile-success-toast.png'), fullPage: false, animations: 'disabled' }); screenshots.push('mobile-success-toast.png');
  assert.deepEqual(errors, []); assert.deepEqual(blocked, []);
} catch (error) { console.error(String(error)); process.exitCode = 1; }
finally { await fs.writeFile(path.join(output, 'report.json'), JSON.stringify({ passed: !process.exitCode, screenshots, toast, errors, blocked, requests, realGoAPI: true, realMySQL: true, realNotion: false, productionAccess: false, tokenRecorded: false, capture: 'viewport; ordinary pages scroll to top; modal/toast retain viewport' }, null, 2)); await browser.close(); console.log(JSON.stringify({ passed: !process.exitCode, screenshots: screenshots.length, output })); }
