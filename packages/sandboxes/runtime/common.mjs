import { timingSafeEqual } from 'node:crypto';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';

export const exec = promisify(execFile);

export function settings(env = process.env) {
  const mode = env.SANDBOX_MODE;
  const port = Number(env.SANDBOX_PORT);
  const token = env.SANDBOX_TOKEN || '';
  if (!['browser', 'computer'].includes(mode)) throw new Error('invalid sandbox mode');
  if (!Number.isInteger(port) || port < 1024 || port > 65535) throw new Error('invalid sandbox port');
  if (token.length < 32 || token.length > 256 || !/^[A-Za-z0-9_-]+$/.test(token)) {
    throw new Error('SANDBOX_TOKEN must contain 32–256 URL-safe random characters');
  }
  return { mode, port, token };
}

// Browser pages must not use ambient localhost access to drive the sandbox.
// Clients send a bearer token explicitly; query-string tokens and browser origins are rejected.
export function authorized(req, token) {
  if (req.headers.origin || req.headers['sec-fetch-site']) return false;
  const received = Buffer.from(req.headers.authorization || '');
  const expected = Buffer.from(`Bearer ${token}`);
  return received.length === expected.length && timingSafeEqual(received, expected);
}

export function json(res, status, value) {
  res.writeHead(status, { 'content-type': 'application/json', 'cache-control': 'no-store', 'x-content-type-options': 'nosniff' });
  res.end(JSON.stringify(value));
}

export async function body(req) {
  if ((req.headers['content-type'] || '').split(';')[0] !== 'application/json') throw new Error('expected JSON');
  const parts = [];
  let size = 0;
  for await (const chunk of req) {
    size += chunk.length;
    if (size > 65536) throw new Error('request too large');
    parts.push(chunk);
  }
  return JSON.parse(Buffer.concat(parts).toString('utf8'));
}

// Child GUI processes do not need the API credential.
export function childEnvironment(env = process.env) {
  const result = { ...env };
  delete result.SANDBOX_TOKEN;
  return result;
}

export function command(file, args, options = {}) {
  return exec(file, args, { env: childEnvironment(), timeout: 10000, maxBuffer: 65536, ...options });
}
