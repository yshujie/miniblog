/* Local fixture service for manual Chrome acceptance. No real account or API is used. */
import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { Buffer } from 'node:buffer';
import { createSyncFixture } from './sync-fixture-data.mjs';
const adminRoot = path.resolve(process.env.MINIBLOG_ADMIN_FIXTURE_DIST || '/private/tmp/miniblog-rollout-browser/admin');
const readerRoot = path.resolve(process.env.MINIBLOG_READER_FIXTURE_DIST || '/private/tmp/miniblog-rollout-browser/reader');
let state = createSyncFixture();
const mime = { '.html': 'text/html; charset=utf-8', '.js': 'text/javascript', '.css': 'text/css', '.json': 'application/json', '.svg': 'image/svg+xml', '.png': 'image/png', '.jpg': 'image/jpeg', '.ico': 'image/x-icon', '.woff': 'font/woff', '.woff2': 'font/woff2' };
const policy = "default-src 'self' data: blob:; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self' http://127.0.0.1:5188 http://127.0.0.1:5189; frame-src 'self' http://127.0.0.1:8099;";
async function staticPage(response, pathname, root, admin) {
  const file = path.resolve(root, '.' + decodeURIComponent(pathname));
  if (file !== root && !file.startsWith(root + path.sep)) { response.writeHead(403); response.end('Forbidden'); return; }
  const extension = path.extname(file); const selected = extension ? file : path.join(root, 'index.html');
  try {
    let body = await readFile(selected);
    if (selected.endsWith('index.html') && admin) body = Buffer.from(body.toString().replace('<head>', '<head><script>localStorage.setItem("Admin-Token", "fixture-only")</script>'));
    response.writeHead(200, { 'content-type': mime[path.extname(selected)] || 'application/octet-stream', 'cache-control': 'no-store', 'content-security-policy': policy, 'x-content-type-options': 'nosniff' }); response.end(body);
  } catch {
    response.writeHead(404, { 'content-type': 'text/plain; charset=utf-8' }); response.end('Local fixture build not found. Build both frontends with localhost API roots into /private/tmp/miniblog-rollout-browser.');
  }
}
function listen(port, root, admin) {
  const server = createServer(async (request, response) => {
    const url = new URL(request.url || '/', `http://127.0.0.1:${port}`);
    try {
      if (url.pathname === '/fixture-info') {
        response.writeHead(200, { 'content-type': 'application/json' }); response.end(JSON.stringify({ fixtureOnly: true, service: admin ? 'admin' : 'reader', externalRequests: 0, requests: state.requests })); return;
      }
      if (url.pathname === '/fixture-reset' && request.method === 'POST') {
        state = createSyncFixture(); response.writeHead(200, { 'content-type': 'application/json' }); response.end(JSON.stringify({ reset: true })); return;
      }
      const chunks = []; for await (const chunk of request) chunks.push(chunk);
      const body = Buffer.concat(chunks).toString(); const data = body ? JSON.parse(body) : null;
      await state.fixture({
        request: () => ({ url: () => url.href, method: () => request.method, postDataJSON: () => data }),
        abort: () => { response.writeHead(403); response.end('External fixture requests are forbidden'); },
        fulfill: ({ status = 200, contentType, headers = {}, body = '' }) => { response.writeHead(status, { ...(contentType ? { 'content-type': contentType } : {}), ...headers }); response.end(body); },
        continue: () => staticPage(response, url.pathname, root, admin)
      });
    } catch (cause) {
      response.writeHead(500, { 'content-type': 'application/json' }); response.end(JSON.stringify({ code: 'fixture_failure', message: String(cause) }));
    }
  });
  server.listen(port, '127.0.0.1', () => console.log(`Fixture ${admin ? 'admin' : 'reader'}: http://127.0.0.1:${port}`));
  return server;
}
const servers = [listen(5189, adminRoot, true), listen(5188, readerRoot, false), listen(8099, readerRoot, false)];
for (const signal of ['SIGINT', 'SIGTERM']) process.on(signal, () => { for (const server of servers) server.close(); });
