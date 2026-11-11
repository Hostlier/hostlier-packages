import http from 'node:http';
// Use the scoped environment token without printing it or placing it in process arguments.
const request = http.get({
  hostname: '127.0.0.1', port: process.env.SANDBOX_PORT, path: '/health', timeout: 1500,
  headers: { Authorization: `Bearer ${process.env.SANDBOX_TOKEN || ''}` },
}, response => {
  response.resume();
  process.exitCode = response.statusCode === 200 ? 0 : 1;
});
request.on('timeout', () => request.destroy());
request.on('error', () => { process.exitCode = 1; });
