// Local visual review only. Not included in the production Vite build.
import { createApp, h, nextTick } from 'vue';
import Desktop from '../src/Desktop.vue';
import Guacamole from '../src/vendor/guacamole.js';
import '@fontsource-variable/manrope';
import '../src/assets/main.css';
import '../src/assets/native-desktop.css';
document.documentElement.dataset.density = 'compact';
document.documentElement.dataset.theme = new URLSearchParams(location.search).get('theme') || 'light';
const encode = (...parts: (string | number)[]) => parts.map(p => `${Array.from(String(p)).length}.${p}`).join(',') + ';';
const workspacePreview = new URLSearchParams(location.search).has('workspace');
const mockConnections = ['台式电脑', 'MacBook Pro', '测试服务器'].map((name, i) => ({ id: `preview-${i}`, name, protocol: 'vnc', host: '127.0.0.1', port: 5900, credentialId: 'saved', settings: { clipboard: 'text' } }));
const deletedSessions: string[] = [];
(window as any).__deletedSessions = deletedSessions;
let sessionCounter = 0;
const nativeFetch = window.fetch.bind(window);
window.fetch = async (input, init) => {
  const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
  if (workspacePreview && url.includes('/api/v1/')) {
    const path = new URL(url, location.origin).pathname.split('/api/v1/')[1];
    let data: any = { items: [] };
    if (path === 'me') data = { uid: 'preview', name: '界面验收' };
    else if (path === 'connections') data = { items: mockConnections };
    else if (path === 'sessions' && init?.method === 'POST') data = { id: `design-preview-${++sessionCounter}` };
    else if (path.endsWith('/ticket')) data = { ticket: 'preview', webSocketPath: `ws://${location.host}/session-preview-tunnel` };
    else if (init?.method === 'DELETE') deletedSessions.push(path.split('/')[1]);
    else if (path.startsWith('sessions/')) data = { state: 'ACTIVE' };
    return new Response(JSON.stringify(data), { headers: { 'Content-Type': 'application/json' } });
  }
  if (url.includes('/api/v1/sessions/design-preview')) return new Response(JSON.stringify(url.endsWith('/ticket')
    ? { ticket: 'local-design-preview', webSocketPath: `ws://${location.host}/session-preview-tunnel` }
    : { state: 'ACTIVE' }), { headers: { 'Content-Type': 'application/json' } });
  return nativeFetch(input, init);
};
const previewParams = new URLSearchParams(location.search);
const displayRequests: { width: number; height: number; time: number }[] = [];
(window as any).__displayRequests = displayRequests;
const mouseRequests: string[][] = [];
(window as any).__mouseRequests = mouseRequests;
function renderFrame(width: number, height: number) {
  // Exercise the real PNG decode and Guacamole canvas path with text, corners,
  // and alternating physical-pixel lines, rather than only a flat rectangle.
  const frame = document.createElement('canvas');
  frame.width = width; frame.height = height;
  const ctx = frame.getContext('2d')!;
  ctx.fillStyle = '#e7edf4'; ctx.fillRect(0, 0, width, height);
  ctx.fillStyle = '#10283d'; ctx.font = '28px sans-serif';
  ctx.fillText('远程桌面清晰度 · Remote desktop 0123456789', 32, 72);
  ctx.font = '18px sans-serif';
  ctx.fillText(`${width} × ${height} · PNG · 1px test lines`, 32, 112);
  for (let x = 32; x < 288; x++) {
    ctx.fillStyle = x % 2 ? '#ffffff' : '#000000';
    ctx.fillRect(x, 160, 1, 48);
  }
  for (const [x, y] of [[0, 0], [width - 16, 0], [0, height - 16], [width - 16, height - 16]]) {
    ctx.fillStyle = '#c82040'; ctx.fillRect(x, y, 16, 16);
  }
  return encode('size', 0, width, height) + encode('png', 14, 0, 0, 0, frame.toDataURL('image/png').split(',')[1]) + encode('sync', Date.now());
}
const NativeSocket = window.WebSocket;
(window as any).WebSocket = class {
  static OPEN = 1; static CLOSED = 3; static CONNECTING = 0; static CLOSING = 2;
  parser = new Guacamole.Parser();
  readyState = 0; protocol = 'guacamole'; onopen: any; onmessage: any; onclose: any; onerror: any;
  constructor(url: string, protocols: string[]) {
    if (!url.includes('/session-preview-tunnel')) return new NativeSocket(url, protocols);
    setTimeout(() => { this.readyState = 1; this.onopen?.(new Event('open'));
      this.push(encode('', 'preview') + renderFrame(innerWidth, innerHeight));
    }, 30);
  }
  push(data: string) { this.onmessage?.({ data }); }
  send(raw: string) {
    this.parser.oninstruction = (op: string, args: string[]) => {
      if (op === '' && args[0] === 'ping') {
        setTimeout(() => this.push(encode('', ...args)), 0);
      } else if (op === 'size') {
        const [width, height] = args.map(Number);
        displayRequests.push({ width, height, time: Date.now() });
        if (previewParams.has('ignore-resize')) return;
        setTimeout(() => this.push(renderFrame(width, height)), Number(previewParams.get('resize-delay')) || 20);
      } else if (op === 'mouse') {
        mouseRequests.push(args);
      } else if (op === 'clipboard' || op === 'blob') {
        setTimeout(() => this.push(encode('ack', args[0], 'OK', 0)), 10);
      }
    };
    this.parser.receive(raw);
  }
  close() { this.readyState = 3; this.onclose?.(new Event('close')); }
};
if (workspacePreview) {
  const { default: RemoteSessionOverlay } = await import('../src/components/RemoteSessionOverlay.vue');
  const { backend, state, refreshWorkspace } = await import('../src/stores/app');
  const { ui } = await import('../src/stores/ui');
  const { default: ElementPlus } = await import('element-plus');
  await import('element-plus/dist/index.css');
  backend.user = { uid: 'preview', name: '界面验收', admin: true };
  await refreshWorkspace();
  state.settings.confirmDisconnect = true;
  state.settings.theme = new URLSearchParams(location.search).get('theme') === 'dark' ? 'dark' : 'light';
  createApp({ render: () => h('main', [h('button', { onClick: () => (window as any).__openPreview(1) }, '连接另一台电脑'), h(RemoteSessionOverlay)]) }).use(ElementPlus).mount('#app');
  (window as any).__openPreview = async (index: number) => {
    ui.remote.connection = mockConnections[index] as any;
    ui.remote.open = true;
    ui.remote.nonce++;
    await nextTick();
  };
  await (window as any).__openPreview(0);
} else createApp({ render: () => {
  const desktop = h(Desktop, { sessionId: 'design-preview', name: 'win', protocol: 'rdp', clipboard: true });
  return previewParams.has('overlay')
    ? h('section', { class: 'fixed inset-0 flex flex-col', 'aria-label': '模拟远程会话工作区' }, [
      h('header', { class: 'h-11 shrink-0 bg-surface px-3' }, '模拟会话标签栏'),
      h('div', { class: 'native-desktop relative min-h-0 min-w-0 flex-1 overflow-hidden' }, [desktop]),
    ])
    : h('div', { class: 'native-desktop', style: 'height:100dvh' }, [desktop]);
} }).mount('#app');
