import test from 'node:test';
import assert from 'node:assert/strict';
import { settings, authorized, childEnvironment, body } from '../common.mjs';
import { actionCommand } from '../actions.mjs';
import { Readable } from 'node:stream';

const token = 'a'.repeat(48);

test('authentication rejects query tokens, browser origins and incorrect credentials', () => {
  assert.equal(authorized({ headers: { authorization: `Bearer ${token}` } }, token), true);
  for (const headers of [{}, { authorization: 'Bearer wrong' }, { authorization: `Bearer ${token}`, origin: 'https://attacker.example' }, { authorization: `Bearer ${token}`, 'sec-fetch-site': 'cross-site' }]) {
    assert.equal(authorized({ headers }, token), false);
  }
  assert.equal(childEnvironment({ SANDBOX_TOKEN: token, DISPLAY: ':99' }).SANDBOX_TOKEN, undefined);
});

test('invalid mode, port and secrets prevent startup', () => {
  const valid = { SANDBOX_MODE: 'browser', SANDBOX_PORT: '9222', SANDBOX_TOKEN: token };
  assert.equal(settings(valid).port, 9222);
  for (const change of [{ SANDBOX_TOKEN: '' }, { SANDBOX_TOKEN: 'short' }, { SANDBOX_TOKEN: ' '.repeat(48) }, { SANDBOX_PORT: '80' }, { SANDBOX_MODE: 'other' }]) {
    assert.throws(() => settings({ ...valid, ...change }));
  }
});

test('desktop controls enforce screen bounds and literal argv', () => {
  assert.deepEqual(actionCommand({ action: 'click', x: 10, y: 20 }), ['mousemove', '--sync', '10', '20', 'click', '1']);
  assert.deepEqual(actionCommand({ action: 'type', text: '$(touch /tmp/test); --help' }).slice(-2), ['--', '$(touch /tmp/test); --help']);
  assert.deepEqual(actionCommand({ action: 'key', key: 'ctrl+l' }), ['key', '--clearmodifiers', 'ctrl+l']);
  for (const value of [null, [], { action: 'move', x: -1, y: 0 }, { action: 'move', x: 1280, y: 0 }, { action: 'move', x: 0, y: 800 }, { action: 'key', key: 'Return;exec' }, { action: 'scroll', direction: 'down', amount: 1000 }, { action: 'type', text: 'x', command: 'exec' }]) {
    assert.throws(() => actionCommand(value));
  }
});

test('JSON action requests have a finite body budget', async () => {
  const req = Readable.from([Buffer.from('{"action":"key","key":"Return"}')]);
  req.headers = { 'content-type': 'application/json' };
  assert.equal((await body(req)).key, 'Return');
  const large = Readable.from([Buffer.alloc(65537)]);
  large.headers = req.headers;
  await assert.rejects(body(large));
});
