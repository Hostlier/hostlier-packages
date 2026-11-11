import http from 'node:http';
import { mkdir } from 'node:fs/promises';
import { settings, authorized, json, body } from './common.mjs';
import { actionCommand } from './actions.mjs';
import { browserRuntime, attachPlaywright } from './browser.mjs';
import { computerRuntime } from './computer.mjs';

// Fail closed if the image, secret, Chromium sandbox or desktop cannot start.
async function main() {
  const config = settings();
  await mkdir('/tmp/home', { recursive: true, mode: 0o700 });
  const runtime = await (config.mode === 'browser' ? browserRuntime() : computerRuntime());
  let serial = Promise.resolve();
  const enqueue = task => {
    const result = serial.then(task);
    serial = result.catch(() => {});
    return result;
  };
  const server = http.createServer({ requestTimeout: 15000, headersTimeout: 10000, maxHeaderSize: 16384 }, async (req, res) => {
    if (!authorized(req, config.token)) return json(res, 401, { error: 'unauthorized' });
    if (!runtime.ready()) return json(res, 503, { error: 'sandbox not ready' });
    if (req.method === 'GET' && req.url === '/health') return json(res, 200, { ready: true, mode: config.mode });
    if (req.method === 'GET' && req.url === '/v1/info') {
      return json(res, 200, {
        mode: config.mode, playwrightVersion: '1.65.0',
        ...(config.mode === 'browser' ? { playwrightPath: '/playwright' } : {
          width: runtime.width, height: runtime.height,
          actions: ['move', 'click', 'type', 'key', 'scroll'], screenshotPath: '/v1/screenshot', actionPath: '/v1/actions',
        }),
      });
    }
    try {
      if (config.mode === 'computer' && req.method === 'GET' && req.url === '/v1/screenshot') {
        const png = await enqueue(() => runtime.screenshot());
        res.writeHead(200, { 'content-type': 'image/png', 'cache-control': 'no-store', 'x-content-type-options': 'nosniff' });
        return res.end(png);
      }
      if (config.mode === 'computer' && req.method === 'POST' && req.url === '/v1/actions') {
        let value;
        try { value = await body(req); actionCommand(value); } catch { return json(res, 400, { error: 'invalid desktop action' }); }
        await enqueue(() => runtime.action(value));
        return json(res, 200, { ok: true });
      }
      return json(res, 404, { error: 'not found' });
    } catch {
      // Do not expose command stderr, URLs, typed content or credentials in diagnostics.
      return json(res, 500, { error: 'sandbox action failed' });
    }
  });
  server.maxConnections = 64;
  const disconnect = config.mode === 'browser' ? attachPlaywright(server, runtime, config.token) : () => {};
  let closing = false;
  const close = async () => {
    if (closing) return;
    closing = true;
    server.close();
    server.closeAllConnections();
    disconnect();
    const deadline = setTimeout(() => process.exit(1), 15000);
    deadline.unref();
    await runtime.close();
    process.exit(0);
  };
  process.on('SIGTERM', close);
  process.on('SIGINT', close);
  server.on('error', () => { void runtime.close().finally(() => process.exit(1)); });
  server.listen(config.port, '127.0.0.1', () => console.log(`${config.mode} sandbox ready`));
}

main().catch(() => { console.error('Sandbox startup failed. Check image, token and host namespace policy.'); process.exit(1); });
