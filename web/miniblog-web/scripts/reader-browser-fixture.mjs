// Offline browser acceptance for the public reader UI. Never calls production services.
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import path from 'node:path';
import { createRequire } from 'node:module';
import { createReaderFixture } from './reader-fixture-data.mjs';
const require = createRequire(import.meta.url);
const { chromium } = require(process.env.MINIBLOG_PLAYWRIGHT_MODULE || 'playwright');
const readerURL = process.env.MINIBLOG_READER_URL || 'http://127.0.0.1:8770';
assert(['127.0.0.1', 'localhost'].includes(new URL(readerURL).hostname), 'Reader URL must be localhost');
const output = path.resolve(process.env.MINIBLOG_READER_OUTPUT || '/private/tmp/miniblog-reader-ui');
await fs.mkdir(output, { recursive: true });
const checks = []; const errors = []; const blockedNetwork = []; const shortScreenMeasurements = [];
const browser = await chromium.launch({ headless: true, ...(process.env.MINIBLOG_CHROMIUM_EXECUTABLE ? { executablePath: process.env.MINIBLOG_CHROMIUM_EXECUTABLE } : {}) });
async function session(width = 1440, height = 1000) {
  const context = await browser.newContext({ viewport: { width, height }, reducedMotion: 'reduce' });
  const fixture = createReaderFixture();
  await context.route('**/*', async route => {
    const url = new URL(route.request().url());
    if (!['localhost', '127.0.0.1'].includes(url.hostname)) { blockedNetwork.push(url.origin); await route.abort(); return; }
    const reply = fixture.handle(url.href, route.request().method());
    if (!reply) { await route.continue(); return; }
    if (fixture.state.delayMS && url.pathname.includes('/v1/')) await new Promise(resolve => setTimeout(resolve, fixture.state.delayMS));
    if (reply.pending) { await new Promise(resolve => setTimeout(resolve, 11500)); await route.abort().catch(() => {}); return; }
    await route.fulfill({ status: reply.status, contentType: reply.contentType, body: reply.body });
  });
  const page = await context.newPage();
  page.on('pageerror', error => errors.push(error.message));
  return { context, page, fixture };
}
const visit = (page, route) => page.goto(readerURL + route);
async function check(name, action) { await action(); checks.push(name); console.log('PASS', name); }
async function shot(page, name, fullPage = true) {
  await page.screenshot({ path: path.join(output, name + '.png'), fullPage, animations: 'disabled' });
}
async function noOverflow(page) {
  assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), 'Horizontal overflow');
}
try {
  const { page, fixture, context } = await session();
  await check('home-real-public-catalog', async () => {
    await visit(page, '/');
    await page.getByRole('heading', { name: '从一个主题开始', exact: true }).waitFor();
    await page.locator('.topic-column').first().waitFor();
    await page.locator('.chapter-preview a').first().waitFor();
    assert.equal(fixture.requests.filter(r => r.pathname === '/v1/blog/articleDetail').length, 0);
    await noOverflow(page); await shot(page, 'desktop-home');
  });
  await check('topic-overview-without-body-or-first-redirect', async () => {
    const before = fixture.requests.filter(r => r.pathname === '/v1/blog/articleDetail').length;
    await page.locator('.site-nav').getByRole('link', { name: 'Golang', exact: true }).click();
    await page.waitForURL('**/topics/go');
    await page.getByRole('heading', { name: 'Golang', exact: true }).waitFor();
    assert.equal(new URL(page.url()).pathname, '/topics/go');
    assert.equal(await page.locator('iframe').count(), 0);
    assert.equal(fixture.requests.filter(r => r.pathname === '/v1/blog/articleDetail').length, before);
    await noOverflow(page); await shot(page, 'desktop-topic');
  });
  await check('title-search-and-chapter-location', async () => {
    const input = page.getByRole('searchbox');
    await input.fill('  HELLO  ');
    assert.equal(await page.locator('.article-row').count(), 1);
    await input.fill('不存在的标题');
    await page.getByText('没有找到匹配的文章', { exact: false }).waitFor();
    await shot(page, 'desktop-search-empty');
    await page.getByRole('button', { name: '清空搜索', exact: true }).last().click();
    assert.equal(await input.inputValue(), '');
    await visit(page, '/topics/go?chapter=lifecycle');
    await page.locator('.chapter-index [aria-current]').waitFor();
    assert((await page.locator('.chapter-index [aria-current]').textContent()).includes('生命周期'));
    assert.equal(await page.locator('iframe').count(), 0);
  });
  await check('legacy-first-article-and-subsection-order', async () => {
    await visit(page, '/blog/go'); await page.locator('iframe').waitFor();
    assert.equal(new URL(page.url()).pathname, '/blog/go/article/9007199254740993');
    assert.equal(await page.locator('iframe').getAttribute('src'), fixture.articles[0].reading_url);
    assert.equal(await page.locator('iframe').getAttribute('referrerpolicy'), 'no-referrer');
    await shot(page, 'desktop-reader');
  });
  await check('stable-frame-collapse-resize-and-refresh', async () => {
    await page.evaluate(() => { window.fixtureFrame = document.querySelector('iframe'); });
    await page.getByRole('button', { name: '收起文章目录', exact: true }).click();
    assert(await page.evaluate(() => window.fixtureFrame === document.querySelector('iframe')));
    await page.getByRole('button', { name: '展开文章目录', exact: true }).click();
    await page.setViewportSize({ width: 390, height: 844 });
    assert(await page.evaluate(() => window.fixtureFrame === document.querySelector('iframe')));
    await noOverflow(page);
    await page.clock.install();
    fixture.articles[0].title = '更新后的文章标题';
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    await page.clock.runFor(60);
    await page.getByRole('heading', { name: '更新后的文章标题', exact: true }).waitFor();
    assert(await page.evaluate(() => window.fixtureFrame === document.querySelector('iframe')));
  });
  await check('mobile-panels-focus-escape-and-selection', async () => {
    await page.getByRole('button', { name: '打开文章目录', exact: true }).click();
    const directory = page.getByRole('dialog', { name: '文章目录', exact: true });
    await directory.waitFor(); await shot(page, 'mobile-directory', false);
    await page.keyboard.press('Escape'); await directory.waitFor({ state: 'hidden' });
    assert.equal(await page.evaluate(() => document.activeElement?.getAttribute('aria-label')), '打开文章目录');
    await page.getByRole('button', { name: '打开主题选择', exact: true }).click();
    const topics = page.getByRole('dialog', { name: '浏览技术主题', exact: true });
    await topics.waitFor(); await shot(page, 'mobile-topics', false);
    assert.equal(await directory.isVisible(), false);
    await topics.getByRole('link', { name: /领域驱动设计/ }).click();
    await page.waitForURL('**/topics/ddd');
    await page.getByRole('heading', { name: '领域驱动设计', exact: true }).waitFor();
    await topics.waitFor({ state: 'hidden' });
    assert.equal(new URL(page.url()).pathname, '/topics/ddd');
    await shot(page, 'mobile-topic');
  });
  await check('history-migration-preserves-query-and-hash', async () => {
    await visit(page, '/blog/removed-module/article/9007199254740993?from=old#part');
    await page.locator('iframe').waitFor();
    const url = new URL(page.url()); assert.equal(url.pathname, '/blog/go/article/9007199254740993');
    assert.equal(url.search, '?from=old'); assert.equal(url.hash, '#part');
    assert.equal(fixture.requests.filter(r => r.pathname === '/v1/blog/moduleDetail' && r.query.module_code === 'removed-module').length, 0);
  });
  await check('refresh-error-retains-frame-authoritative404-removes', async () => {
    fixture.state.articleError = true;
    await page.evaluate(() => window.dispatchEvent(new Event('focus'))); await page.clock.runFor(60);
    await page.getByText('更新失败', { exact: false }).waitFor(); assert.equal(await page.locator('iframe').count(), 1);
    await shot(page, 'mobile-refresh-error');
    fixture.state.articleError = false; fixture.state.held = true;
    await page.evaluate(() => window.dispatchEvent(new Event('focus'))); await page.clock.runFor(60);
    await page.getByRole('heading', { name: '文章不可用', exact: true }).waitFor();
    assert.equal(await page.locator('iframe').count(), 0); await shot(page, 'mobile-not-found');
  });
  await context.close();
  await check('new-page-scroll-reset-and-history-position', async () => {
    const s = await session(); await visit(s.page, '/'); await s.page.locator('.chapter-preview a').first().waitFor();
    await s.page.evaluate(() => window.scrollTo(0, 300)); const homeY = await s.page.evaluate(() => scrollY); assert(homeY > 0);
    await s.page.locator('.topic-column h3 a').first().click(); await s.page.waitForURL('**/topics/go');
    await s.page.locator('.article-row').first().waitFor(); await s.page.waitForFunction(() => scrollY === 0);
    await s.page.goBack(); await s.page.waitForURL(readerURL + '/');
    await s.page.waitForFunction(expected => Math.abs(scrollY - expected) <= 1, homeY);
    await s.context.close();
  });
  await check('directory-selection-and-previous-next' , async () => {
    const s = await session(390, 844);
    await visit(s.page, '/blog/go/article/9007199254740993'); await s.page.locator('iframe').waitFor();
    await s.page.locator('.link-message').waitFor({ state: 'hidden' }); await shot(s.page, 'mobile-reader', false);
    await s.page.getByRole('button', { name: '打开文章目录', exact: true }).click();
    const dialog = s.page.getByRole('dialog', { name: '文章目录', exact: true }); await dialog.waitFor();
    await dialog.getByRole('searchbox').fill('类型');
    await dialog.getByRole('button', { name: 'Go 的类型、变量与作用域', exact: true }).click();
    await s.page.waitForURL('**/blog/go/article/9007199254740994'); await dialog.waitFor({ state: 'hidden' });
    await s.page.waitForFunction(() => document.querySelector('iframe')?.src.endsWith('/9007199254740994'));
    await s.page.locator('.reading-pagination .next').click();
    await s.page.waitForURL('**/blog/go/article/9007199254740995');
    await s.page.waitForFunction(() => document.querySelector('iframe')?.src.endsWith('/9007199254740995'));
    await s.page.locator('.reading-pagination .previous').click();
    await s.page.waitForURL('**/blog/go/article/9007199254740994');
    await s.page.waitForFunction(() => document.querySelector('iframe')?.src.endsWith('/9007199254740994'));
    await s.context.close();
  });
  await check('new-route-hard-refresh-and-browser-history', async () => {
    const s = await session(); await visit(s.page, '/topics/go?chapter=lifecycle');
    await s.page.locator('.article-row').first().waitFor(); await s.page.reload();
    await s.page.locator('.article-row').first().waitFor(); assert.equal(await s.page.locator('iframe').count(), 0);
    await s.page.getByRole('link', { name: /从第一篇开始/ }).click();
    await s.page.waitForURL('**/blog/go/article/9007199254740993'); await s.page.locator('iframe').waitFor();
    await s.page.goBack(); await s.page.waitForURL('**/topics/go?chapter=lifecycle');
    await s.page.locator('.article-row').first().waitFor(); assert.equal(await s.page.locator('iframe').count(), 0);
    await s.page.goForward(); await s.page.waitForURL('**/blog/go/article/9007199254740993');
    await s.page.locator('iframe').waitFor(); await s.context.close();
  });
  await check('unknown-dynamic-topic-and-independent-preview-error', async () => {
    const s = await session(320, 844);
    s.fixture.modules.push({ id: '5', code: 'unknown', title: '一个来自接口的新主题名称', sections: [] });
    s.fixture.state.detailError = true;
    await visit(s.page, '/'); await s.page.getByRole('heading', { name: '从一个主题开始', exact: true }).waitFor();
    const unknown = s.page.locator('.topic-column').filter({ hasText: '一个来自接口的新主题名称' });
    await unknown.scrollIntoViewIfNeeded(); await unknown.getByText('章节预览暂时无法加载。', { exact: true }).waitFor();
    assert.equal(await unknown.getByRole('link', { name: '浏览全部文章', exact: true }).isEnabled(), true);
    s.fixture.state.detailError = false;
    await unknown.getByRole('button', { name: '重试章节预览', exact: true }).click();
    await unknown.getByText('暂无已发布文章', { exact: true }).waitFor(); await noOverflow(s.page);
    await unknown.getByRole('link', { name: '浏览全部文章', exact: true }).click();
    await s.page.waitForURL('**/topics/unknown'); await s.page.getByRole('heading', { name: '一个来自接口的新主题名称', exact: true }).waitFor();
    await noOverflow(s.page); await shot(s.page, 'mobile-unknown-topic'); await s.context.close();
  });
  for (const width of [320, 390, 649, 650, 651, 768, 899, 900, 901, 1024, 1440]) {
    await check('responsive-' + width, async () => {
      const s = await session(width, width < 651 ? 844 : 1000);
      for (const route of ['/', '/topics/go', '/blog/go/article/9007199254740997']) {
        await visit(s.page, route);
        if (route.startsWith('/blog')) await s.page.locator('iframe').waitFor();
        else await s.page.getByRole('heading').first().waitFor();
        await noOverflow(s.page);
        if (route.startsWith('/blog')) {
          const box = await s.page.locator('iframe').boundingBox(); assert(box && box.height >= 180, 'Reader frame clipped');
        }
      }
      if (width === 390) { await shot(s.page, 'mobile-reader-long-title'); await visit(s.page, '/'); await shot(s.page, 'mobile-home'); }
      await s.context.close();
    });
  }
  for (const [width, height] of [[390, 500], [844, 390], [1024, 500]]) {
    await check(`short-screen-${width}x${height}`, async () => {
      const s = await session(width, height); s.fixture.state.waiting = true;
      await s.page.goto(readerURL + '/blog/go/article/9007199254740997', { waitUntil: 'domcontentloaded' });
      await s.page.locator('iframe').waitFor(); await noOverflow(s.page);
      const box = await s.page.locator('iframe').boundingBox();
      assert(box && box.height >= 90 && box.y + box.height <= height, 'Short screen reader frame clipped');
      const original = await s.page.locator('.original-link').boundingBox();
      assert(original && original.height >= 44 && original.y + original.height <= height, 'Original link inaccessible');
      shortScreenMeasurements.push({ width, height, frameHeight: box.height });
      await shot(s.page, `reader-short-${width}x${height}`, false); await s.context.close();
    });
  }
  for (const scenario of ['empty-topic' , 'topic404', 'article404', 'error', 'missing-link', 'waiting', 'global404', 'home-empty', 'home-error']) {
    await check('state-' + scenario, async () => {
      const s = await session(390, 844); let route = '/blog/go/article/9007199254740993'; let title;
      if (scenario === 'empty-topic') { route = '/topics/empty'; title = '暂无已发布文章'; }
      if (scenario === 'topic404') { route = '/topics/missing'; title = '主题不可用'; }
      if (scenario === 'article404') { route = '/blog/go/article/404'; title = '文章不可用'; }
      if (scenario === 'error') { s.fixture.state.articleError = true; title = '加载失败'; }
      if (scenario === 'missing-link') { s.fixture.articles[0].reading_url = ''; s.fixture.articles[0].external_link = ''; }
      if (scenario === 'waiting') { s.fixture.state.waiting = true; }
      if (scenario === 'global404') route = '/not-a-real-page';
      if (scenario === 'home-empty') { route = '/'; s.fixture.state.list = 'empty'; }
      if (scenario === 'home-error') { route = '/'; s.fixture.state.list = 'error'; }
      await visit(s.page, route);
      if (title) await s.page.getByRole('heading', { name: title, exact: true }).waitFor();
      if (scenario === 'missing-link') await s.page.getByText('文章链接不可用', { exact: false }).waitFor();
      if (scenario === 'waiting') await s.page.getByText('如无法显示，请打开原文', { exact: false }).waitFor();
      if (scenario === 'global404') await s.page.getByText('404', { exact: true }).waitFor();
      if (scenario === 'home-empty') await s.page.getByRole('heading', { name: '暂无阅读模块', exact: true }).waitFor();
      if (scenario === 'home-error') await s.page.getByRole('heading', { name: '主题暂时无法加载', exact: true }).waitFor();
      await noOverflow(s.page); await shot(s.page, 'mobile-state-' + scenario); await s.context.close();
    });
  }
  for (const [route, title, name] of [
    ['/', '正在加载阅读主题…', 'home'],
    ['/topics/go', '正在加载主题目录…', 'topic'],
    ['/blog/go/article/9007199254740993', '正在加载文章…', 'reader'],
  ]) {
    await check('loading-' + name, async () => {
      const s = await session(390, 844); s.fixture.state.delayMS = 1500;
      await visit(s.page, route); await s.page.getByRole('heading', { name: title, exact: true }).waitFor();
      await noOverflow(s.page); await shot(s.page, 'mobile-loading-' + name); await s.context.close();
    });
  }
  assert.deepEqual(errors, []); assert.deepEqual(blockedNetwork, []);
  const report = { passed: true, checks, shortScreenMeasurements, pageErrors: errors, productionNetworkRequests: 0, screenshots: output, evidence: 'Local public API and document substitutes; real Notion/device acceptance remains separate.' };
  await fs.writeFile(path.join(output, 'report.json'), JSON.stringify(report, null, 2));
  console.log(JSON.stringify(report, null, 2));
} catch (error) {
  await fs.writeFile(path.join(output, 'failure.json'), JSON.stringify({ passed: false, checks, errors, blockedNetwork, error: String(error) }, null, 2));
  throw error;
} finally { await browser.close(); }
