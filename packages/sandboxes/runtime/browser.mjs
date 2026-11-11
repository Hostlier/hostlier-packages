import http from 'node:http';
import { chromium } from 'playwright';
import { authorized, childEnvironment } from './common.mjs';

export async function browserRuntime() {
  // Port zero and Playwright's random backend path keep the internal endpoint independent
  // of the application listener. The authenticated gateway has the declared stable port.
  const browser = await chromium.launchServer({
    host: '127.0.0.1', port: 0, chromiumSandbox: true,
    args: ['--disable-dev-shm-usage'], env: childEnvironment(),
  });
  let ready = true;
  browser.on('close', () => { ready = false; });
  return {
    ready: () => ready,
    endpoint: new URL(browser.wsEndpoint()),
    close: () => browser.close(),
  };
}

// Forward the WebSocket handshake and bytes without interpreting Playwright's protocol.
// The upstream location is selected by this process, never from client input.
export function attachPlaywright(server, runtime, token) {
  const connections = new Set();
  server.on('upgrade', (req, socket, head) => {
    if (!authorized(req, token)) return reject(socket, 401);
    if (req.url !== '/playwright' || !runtime.ready()) return reject(socket, 404);
    const endpoint = runtime.endpoint;
    const headers = { ...req.headers, host: endpoint.host };
    delete headers.authorization;
    const upstream = http.request({
      hostname: '127.0.0.1', port: endpoint.port, path: endpoint.pathname,
      method: 'GET', headers, timeout: 10000,
    });
    connections.add(socket);
    socket.on('error', () => upstream.destroy());
    socket.on('close', () => { connections.delete(socket); upstream.destroy(); });
    upstream.on('timeout', () => upstream.destroy());
    upstream.on('error', () => socket.destroy());
    upstream.on('response', () => { upstream.destroy(); reject(socket, 502); });
    upstream.on('upgrade', (res, peer, peerHead) => {
      peer.setTimeout(0);
      peer.on('error', () => socket.destroy());
      peer.on('close', () => socket.destroy());
      socket.on('close', () => peer.destroy());
      socket.write(`HTTP/1.1 101 Switching Protocols\r\n${res.rawHeaders.reduce((text, item, i) => text + (i % 2 ? `${item}\r\n` : `${item}: `), '')}\r\n`);
      if (peerHead.length) socket.write(peerHead);
      if (head.length) peer.write(head);
      socket.pipe(peer).pipe(socket);
    });
    upstream.end();
  });
  return () => { for (const socket of connections) socket.destroy(); };
}

function reject(socket, status) {
  socket.end(`HTTP/1.1 ${status} Rejected\r\nConnection: close\r\nContent-Length: 0\r\n\r\n`);
}
