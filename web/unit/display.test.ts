import { test } from 'node:test';
import assert from 'node:assert/strict';
import { displaySize, fixedResolution, sessionDisplay, nativeDisplayScale, performanceDisplay, loadRenderMode, saveRenderMode } from '../src/display.ts';

test('Retina requests device pixels and matching Windows DPI, while normal screens stay 1:1', () => {
  assert.deepEqual(displaySize(1512, 900, 1, 1), { width: 1512, height: 900, dpi: 96 });
  assert.deepEqual(displaySize(1512, 900, 1, 2), { width: 3024, height: 1800, dpi: 192 });
  assert.deepEqual(displaySize(1280, 800, 1, 1.5), { width: 1920, height: 1200, dpi: 144 });
});
test('framebuffer and DPI limits preserve the aspect ratio', () => {
  assert.deepEqual(displaySize(5120, 2880, 1, 2), { width: 3840, height: 2160, dpi: 96 });
  const tall = displaySize(1800, 3000, 1, 2);
  assert.equal(tall.height, 2160);
  assert.equal(tall.width, 1296);
});
test('content size adjusts Windows DPI before reducing framebuffer detail at the DPI limit', () => {
  assert.deepEqual(displaySize(1500, 900, 0.5, 1), { width: 3000, height: 1800, dpi: 96 });
  assert.deepEqual(displaySize(1500, 900, 1.5, 1), { width: 1500, height: 900, dpi: 144 });
  assert.deepEqual(displaySize(1500, 900, 1.5, 2), { width: 2000, height: 1200, dpi: 192 });
  assert.deepEqual(displaySize(1500, 900, 0.75, 2), { width: 3000, height: 1800, dpi: 144 });
  assert.deepEqual(displaySize(1500, 900, NaN, NaN), displaySize(1500, 900, 1, 1));
  const small = displaySize(390, 700, 2, 2);
  assert.equal(small.width, 390);
  assert.equal(small.dpi, 192);
});
test('fixed framebuffer still negotiates matching DPI and rejects arbitrary stored sizes', () => {
  assert.deepEqual(sessionDisplay(1920, 1080, 1, '3840x2160', 2), { width: 3840, height: 2160, dpi: 192 });
  assert.deepEqual(fixedResolution('1920x1080'), { width: 1920, height: 1080, dpi: 96 });
  for (const mode of ['auto', '99999x99999', 'NaNx1080']) assert.equal(fixedResolution(mode), undefined);
});
test('native-size display maps each remote pixel to one physical screen pixel', () => {
  assert.equal(nativeDisplayScale(1, 2), 0.5);
  assert.equal(nativeDisplayScale(1.5, 2), 0.75);
  assert.equal(nativeDisplayScale(1, 1), 1);
});

test('smooth mode caps automatic high-DPI pixels while preserving explicit resolutions', () => {
  assert.deepEqual(performanceDisplay(1920, 1080, 1, 'auto', 'smooth', 2), { width: 1920, height: 1080, dpi: 96 });
  assert.deepEqual(performanceDisplay(1920, 1080, 1, '3840x2160', 'smooth', 2), { width: 3840, height: 2160, dpi: 192 });
  for (const profile of ['clear', 'balanced']) {
    assert.deepEqual(performanceDisplay(1512, 900, 1, 'auto', profile, 2), sessionDisplay(1512, 900, 1, 'auto', 2));
  }
  const portrait = performanceDisplay(1200, 1800, 1, 'auto', 'smooth', 2);
  assert.ok(portrait.width <= 1920 && portrait.height <= 1080);
  assert.ok(Math.abs(portrait.width / portrait.height - 1200 / 1800) < .002);
  assert.deepEqual(performanceDisplay(800, 600, 1, 'auto', 'smooth', 1), sessionDisplay(800, 600, 1, 'auto', 1));
});

test('render preferences default safely and stay isolated by protocol', () => {
  const values = new Map<string, string>();
  const previous = Object.getOwnPropertyDescriptor(globalThis, 'localStorage');
  Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => values.set(key, value),
  } });
  try {
    assert.equal(loadRenderMode('rdp'), 'balanced');
    saveRenderMode('rdp', 'smooth');
    assert.equal(loadRenderMode('rdp'), 'smooth');
    assert.equal(loadRenderMode('vnc'), 'balanced');
    saveRenderMode('rdp', 'invalid');
    assert.equal(loadRenderMode('rdp'), 'smooth');
    values.set('remote-desktop.render-mode.rdp', 'corrupt');
    assert.equal(loadRenderMode('rdp'), 'balanced');
  } finally {
    if (previous) Object.defineProperty(globalThis, 'localStorage', previous);
    else Reflect.deleteProperty(globalThis, 'localStorage');
  }
});
