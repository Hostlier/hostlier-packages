import { spawn } from 'node:child_process';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { chromium } from 'playwright';
import { actionCommand } from './actions.mjs';
import { childEnvironment, command } from './common.mjs';

export async function computerRuntime() {
  const children = [];
  const requiredChildren = [];
  let ready = false;
  let browser;
  const start = (file, args, required = true) => {
    const child = spawn(file, args, { env: childEnvironment(), stdio: 'ignore', detached: true });
    children.push(child);
    if (required) requiredChildren.push(child);
    child.on('error', () => { if (required) ready = false; });
    child.on('exit', () => { if (required) ready = false; });
    return child;
  };
  const close = async () => {
    ready = false;
    await browser?.close().catch(() => {});
    for (const child of children.reverse()) {
      if (child.pid) { try { process.kill(-child.pid, 'SIGTERM'); } catch {} }
    }
  };
  try {
    const display = start('Xvfb', [':99', '-screen', '0', '1280x800x24', '-nolisten', 'tcp']);
    await waitForDisplay(display);
    const wm = start('openbox', ['--sm-disable']);
    await waitForWindowManager(wm);
    start('xterm', ['-geometry', '90x25+10+10', '-title', 'Sandbox terminal'], false);
    browser = await chromium.launch({
      headless: false, chromiumSandbox: true, env: childEnvironment(),
      args: ['--disable-dev-shm-usage', '--window-size=1000,700'],
    });
    browser.on('disconnected', () => { ready = false; });
    const page = await browser.newPage({ viewport: null });
    await page.goto('about:blank');
    ready = true;
    return { ready: () => ready && requiredChildren.every(child => child.pid && child.exitCode === null && child.signalCode === null), close, action, screenshot, width: 1280, height: 800 };
  } catch (error) {
    await close();
    throw error;
  }
}

async function waitForDisplay(child) {
  for (let attempt = 0; attempt < 100; attempt++) {
    if (child.exitCode !== null) throw new Error('display exited');
    try { await command('xdpyinfo', []); return; } catch {}
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  throw new Error('display not ready');
}

async function waitForWindowManager(child) {
  for (let attempt = 0; attempt < 100; attempt++) {
    if (child.exitCode !== null) throw new Error('window manager exited');
    const { stdout } = await command('xprop', ['-root', '_NET_SUPPORTING_WM_CHECK']);
    if (stdout.includes('window id # 0x')) return;
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  throw new Error('window manager not ready');
}

async function action(value) {
  await command('xdotool', actionCommand(value));
}

async function screenshot() {
  const dir = await mkdtemp('/tmp/hostlier-screen-');
  try {
    const file = `${dir}/screen.png`;
    await command('scrot', ['--silent', file]);
    return await readFile(file);
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
}
