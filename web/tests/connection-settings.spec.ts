import { test, expect } from '@playwright/test';

test.use({ channel: 'chrome' });
const preview = process.env.RD_UI_PREVIEW_URL || 'http://127.0.0.1:18081/app/remote-desktop/tests/session-ui-preview.html';

test('failed sessions keep picture settings clickable and apply them on reconnect', async ({ page }) => {
  await page.goto(preview + '?workspace&failed');
  await expect(page.getByRole('heading', { name: '远程连接未完成' })).toBeVisible();
  await page.getByRole('button', { name: '会话设置', exact: true }).click();
  const pictureMode = page.getByRole('combobox', { name: '画质模式', exact: true });
  // A real click must pass hit testing against the connection progress layer.
  await pictureMode.click();
  await page.getByRole('option', { name: '清晰优先', exact: true }).click();
  await expect(pictureMode).toContainText('清晰优先');
  await page.getByRole('button', { name: '重新连接并应用', exact: true }).click();
  await expect.poll(() => page.evaluate(() => (window as any).__deletedSessions)).toContain('design-preview-1');
  await page.getByRole('button', { name: '会话设置', exact: true }).click();
  await expect(page.getByRole('combobox', { name: '画质模式', exact: true })).toContainText('清晰优先');
});


test('RDP wallpaper defaults on and changes are applied on reconnect', async ({ page }) => {
  await page.goto(preview + '?workspace&rdp');
  await page.getByRole('button', { name: '会话设置', exact: true }).click();
  const wallpaper = page.getByRole('switch', { name: '桌面壁纸' });
  await expect(wallpaper).toBeChecked();
  await expect.poll(() => page.evaluate(() => (window as any).__createdSessions.at(-1)?.wallpaper)).toBe(true);
  await wallpaper.uncheck();
  await expect(page.getByText('桌面壁纸待应用', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: '重新连接并应用', exact: true }).click();
  await expect.poll(() => page.evaluate(() => (window as any).__createdSessions.at(-1)?.wallpaper)).toBe(false);
  await page.getByRole('button', { name: '会话设置', exact: true }).click();
  await expect(page.getByRole('switch', { name: '桌面壁纸' })).not.toBeChecked();
  await page.getByRole('switch', { name: '桌面壁纸' }).check();
  await page.getByRole('button', { name: '重新连接并应用', exact: true }).click();
  await expect.poll(() => page.evaluate(() => (window as any).__createdSessions.at(-1)?.wallpaper)).toBe(true);
});
