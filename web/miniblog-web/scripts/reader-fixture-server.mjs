import http from 'node:http';
import { createReaderFixture } from './reader-fixture-data.mjs';
const port = Number(process.env.MINIBLOG_READER_FIXTURE_PORT || 8771);
const fixture = createReaderFixture(`http://127.0.0.1:${port}`);
http.createServer((req, res) => {
  const reply = fixture.handle(req.url, req.method);
  if (!reply) { res.writeHead(404); res.end('Local reader fixture only'); return; }
  res.writeHead(reply.status, { 'content-type': reply.contentType, 'cache-control': 'no-store' });
  res.end(reply.body);
}).listen(port, '127.0.0.1', () => console.log(`Reader fixture: http://127.0.0.1:${port} (local example data)`));
