import assert from 'node:assert/strict';
import { chromium } from 'playwright';
const base = `http://127.0.0.1:${process.env.SANDBOX_PORT}`;
const headers = { Authorization: `Bearer ${process.env.SANDBOX_TOKEN}` };
assert.equal((await fetch(`${base}/health`)).status, 401);
assert.equal((await fetch(`${base}/health`, { headers: { ...headers, Origin: 'https://example.com' } })).status, 401);
await assert.rejects(chromium.connect(base.replace('http:', 'ws:') + '/playwright', { timeout: 3000 }));
const browser = await chromium.connect(base.replace('http:', 'ws:') + '/playwright', { headers });
try {
  const first = await browser.newContext();
  const second = await browser.newContext();
  await first.addCookies([{ name: 'private', value: 'first-context', domain: 'example.com', path: '/' }]);
  assert.equal((await second.cookies()).length, 0);
  const page = await first.newPage();
  await page.goto('data:text/html,<title>Sandbox ready</title><button onclick="this.textContent=\'Clicked\'">Click me</button>');
  assert.equal(await page.title(), 'Sandbox ready');
  await page.getByRole('button').click();
  assert.equal(await page.getByRole('button').textContent(), 'Clicked');
  const png = await page.screenshot();
  assert.equal(png.subarray(1, 4).toString(), 'PNG');
  await first.close();
  await second.close();
} finally { await browser.close(); }
console.log('browser: authentication, Playwright actions, screenshots and context isolation passed');
