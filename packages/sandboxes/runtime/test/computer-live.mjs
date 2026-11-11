import assert from 'node:assert/strict';
const base = `http://127.0.0.1:${process.env.SANDBOX_PORT}`;
const headers = { Authorization: `Bearer ${process.env.SANDBOX_TOKEN}` };
assert.equal((await fetch(`${base}/health`)).status, 401);
const info = await (await fetch(`${base}/v1/info`, { headers })).json();
assert.equal(info.width, 1280);
const before = Buffer.from(await (await fetch(`${base}/v1/screenshot`, { headers })).arrayBuffer());
assert.equal(before.subarray(1, 4).toString(), 'PNG');
assert.equal(before.readUInt32BE(16), 1280);
assert.equal(before.readUInt32BE(20), 800);
for (const action of [{ action: 'click', x: 400, y: 150 }, { action: 'key', key: 'ctrl+l' }, { action: 'type', text: 'data:text/html,<h1>Computer sandbox verified</h1>' }, { action: 'key', key: 'Return' }, { action: 'scroll', direction: 'down', amount: 1 }]) {
  const res = await fetch(`${base}/v1/actions`, { method: 'POST', headers: { ...headers, 'Content-Type': 'application/json' }, body: JSON.stringify(action) });
  assert.equal(res.status, 200);
}
const invalid = await fetch(`${base}/v1/actions`, { method: 'POST', headers: { ...headers, 'Content-Type': 'application/json' }, body: JSON.stringify({ action: 'click', x: -1, y: 0 }) });
assert.equal(invalid.status, 400);
const after = Buffer.from(await (await fetch(`${base}/v1/screenshot`, { headers })).arrayBuffer());
assert.notDeepEqual(before, after);
console.log('computer: authentication, PNG screenshot, mouse, typing, keyboard and scroll passed');
