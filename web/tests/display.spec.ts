// Local mock verifies browser negotiation and request ordering, not Windows rendering.
import { test, expect, type Page } from '@playwright/test';
test.use({ channel: 'chrome', deviceScaleFactor: 2, viewport: { width: 1200, height: 800 } });
const preview = 'http://127.0.0.1:18081/app/remote-desktop/tests/session-ui-preview.html';
async function settings(page: Page) {
  await page.getByRole('button', { name: '显示桌面工具栏', exact: true }).click();
  await page.getByRole('button', { name: '会话设置', exact: true }).click();
}
async function selectResolution(page: Page, label: string) {
  await page.getByRole('combobox', { name: '分辨率', exact: true }).click();
  await page.getByRole('option', { name: label, exact: true }).click();
}
test('Retina renders native pixels and stages content DPI until explicit reconnect', async ({ page }) => {
  await page.goto(preview);
  await expect.poll(() => page.evaluate(() => (window as any).__displayRequests.length)).toBe(1);
  await expect.poll(() => page.evaluate(() => (window as any).__displayRequests[0].width)).toBe(2400);
  await settings(page);
  await page.getByRole('combobox', { name: '画面缩放', exact: true }).click();
  await page.getByRole('option', { name: '像素原始尺寸', exact: true }).click();
  await expect.poll(() => page.locator('.remote-canvas > div').evaluate((el) => el.getBoundingClientRect().width)).toBe(1200);
  await page.getByRole('radio', { name: '大', exact: true }).click();
  await expect(page.getByRole('button', { name: '重新连接并应用', exact: true })).toBeEnabled();
  await page.getByRole('combobox', { name: '画质模式', exact: true }).click();
  await page.getByRole('option', { name: '清晰优先', exact: true }).click();
  await expect(page.locator('.session-settings-footer')).toContainText('画质模式、内容大小待应用');
  await expect(page.getByRole('button', { name: '重新连接并应用', exact: true })).toHaveCount(1);
  await page.waitForTimeout(1700);
  expect(await page.evaluate(() => (window as any).__displayRequests.length)).toBe(1);
  expect(await page.evaluate(() => localStorage.getItem('remote-desktop.display-scale.rdp'))).toBe('1.5');
  await page.screenshot({ path: '../build/display-retina-settings.png' });
});
test('rapid resolution changes wait for acknowledgement and retain only latest selection', async ({ page }) => {
  await page.goto(preview + '?resize-delay=2500');
  await expect.poll(() => page.evaluate(() => (window as any).__displayRequests.length)).toBe(1);
  await settings(page);
  await selectResolution(page, '1920 × 1080');
  await page.waitForTimeout(700);
  await selectResolution(page, '2560 × 1440');
  await page.waitForTimeout(700);
  expect(await page.evaluate(() => (window as any).__displayRequests.length)).toBe(1);
  await expect.poll(() => page.evaluate(() => (window as any).__displayRequests.length)).toBe(2);
  expect(await page.evaluate(() => (window as any).__displayRequests.map((r: any) => r.width))).toEqual([2400, 2560]);
  await expect(page.getByLabel('实际分辨率')).toHaveText('2560 × 1440');
});
test('unacknowledged resize stops without resend storm and offers reconnect', async ({ page }) => {
  await page.goto(preview + '?ignore-resize');
  await settings(page);
  await expect(page.getByLabel('实际分辨率')).toContainText('远端未确认', { timeout: 12000 });
  await expect(page.getByRole('button', { name: '重新连接并应用', exact: true })).toBeEnabled();
  await selectResolution(page, '1920 × 1080');
  await page.waitForTimeout(2000);
  expect(await page.evaluate(() => (window as any).__displayRequests.length)).toBe(1);
});

test('overlay fit has no unscaled overflow and fill covers both dimensions with correct pointer mapping', async ({ page }) => {
  const failures: string[] = [];
  page.on('pageerror', error => failures.push(error.message));
  await page.goto(preview + '?overlay');
  await expect.poll(() => page.evaluate(() => (window as any).__displayRequests.length)).toBe(1);
  await settings(page);
  await selectResolution(page, '1920 × 1080');
  await expect(page.getByLabel('实际分辨率')).toHaveText('1920 × 1080');
  const metrics = () => page.locator('.remote-canvas').evaluate(el => {
    const frame = el.querySelector('.remote-surface')!.getBoundingClientRect();
    return { width: el.clientWidth, height: el.clientHeight, scrollWidth: el.scrollWidth, scrollHeight: el.scrollHeight, frameWidth: frame.width, frameHeight: frame.height };
  });
  let m = await metrics();
  expect(m.width).toBe(1200);
  expect(m.height).toBe(756);
  expect(m.scrollWidth).toBe(m.width);
  expect(m.scrollHeight).toBe(m.height);
  expect(m.frameWidth).toBe(1200);
  expect(m.frameHeight).toBe(675);
  await page.getByRole('combobox', { name: '画面缩放', exact: true }).click();
  await page.getByRole('option', { name: '铺满窗口（拉伸）', exact: true }).click();
  await expect.poll(async () => (await metrics()).frameHeight).toBe(756);
  m = await metrics();
  expect(m.frameWidth).toBe(m.width);
  expect(m.frameHeight).toBe(m.height);
  expect(m.scrollWidth).toBe(m.width);
  expect(m.scrollHeight).toBe(m.height);
  await page.getByRole('button', { name: '关闭会话设置' }).click();
  await page.locator('.remote-surface').click({ position: { x: 600, y: 378 } });
  await expect.poll(() => page.evaluate(() => (window as any).__mouseRequests.at(-1)?.slice(0, 2))).toEqual(['960', '540']);
  await page.getByRole('button', { name: '显示桌面工具栏', exact: true }).click();
  await page.getByRole('button', { name: '全屏', exact: true }).click();
  await expect.poll(() => page.evaluate(() => document.fullscreenElement?.className)).toBe('desktop-panel');
  await expect.poll(async () => { const v = await metrics(); return Math.abs(v.frameHeight - v.height); }).toBe(0);
  m = await metrics();
  expect(m.frameWidth).toBe(m.width);
  expect(m.frameHeight).toBe(m.height);
  await page.evaluate(() => document.exitFullscreen());
  expect(failures).toEqual([]);
});

test('Retina PNG detail retains alternating one-physical-pixel lines in the actual browser screenshot', async ({ page }) => {
  await page.goto(preview + '?overlay');
  await expect.poll(() => page.locator('.remote-surface > div').evaluate(el => el.style.width)).toBe('2400px');
  // Wait for PNG decoding and drawing, independently of the size notification.
  await expect.poll(() => page.locator('.remote-surface canvas').first().evaluate((el: HTMLCanvasElement) => el.getContext('2d')!.getImageData(32, 180, 1, 1).data[3])).toBe(255);
  const screenshot = await page.screenshot({ path: '../build/display-pixel-fidelity.png' });
  const row = await page.evaluate(async encoded => {
    const image = new Image();
    image.src = 'data:image/png;base64,' + encoded;
    await image.decode();
    const canvas = document.createElement('canvas');
    canvas.width = image.width; canvas.height = image.height;
    const ctx = canvas.getContext('2d')!;
    ctx.drawImage(image, 0, 0);
    // The production-shaped overlay has a 44 CSS-pixel header, at DPR 2.
    const rgba = ctx.getImageData(32, 44 * 2 + 180, 256, 1).data;
    return Array.from({ length: 256 }, (_, i) => rgba[i * 4]);
  }, screenshot.toString('base64'));
  expect(row).toEqual(Array.from({ length: 256 }, (_, i) => i % 2 ? 255 : 0));
});

test('every fixed resolution fits inside the production-shaped overlay without scrollbars', async ({ page }) => {
  await page.goto(preview + '?overlay');
  await settings(page);
  for (const [width, height] of [[1280, 720], [1600, 900], [1920, 1080], [1920, 1200], [2560, 1440], [3840, 2160]]) {
    await selectResolution(page, `${width} × ${height}`);
    await expect(page.getByLabel('实际分辨率')).toHaveText(`${width} × ${height}`);
    const m = await page.locator('.remote-canvas').evaluate(el => {
      const frame = el.querySelector('.remote-surface')!.getBoundingClientRect();
      return [el.clientWidth, el.clientHeight, el.scrollWidth, el.scrollHeight, frame.width, frame.height];
    });
    expect(m[2]).toBe(m[0]); expect(m[3]).toBe(m[1]);
    expect(m[4]).toBeLessThanOrEqual(m[0] + 0.01);
    expect(m[5]).toBeLessThanOrEqual(m[1] + 0.01);
    expect(Math.abs(m[4] / m[5] - width / height)).toBeLessThan(0.001);
  }
  await page.getByRole('combobox', { name: '画面缩放', exact: true }).click();
  await page.getByRole('option', { name: '像素原始尺寸', exact: true }).click();
  await expect.poll(() => page.locator('.remote-surface').evaluate(el => el.clientWidth)).toBe(1920);
  expect(await page.locator('.remote-canvas').evaluate(el => el.scrollWidth)).toBe(1920);
});
