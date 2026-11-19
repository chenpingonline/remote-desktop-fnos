import { expect, test } from "@playwright/test";

test("copy in A and paste immediately after switching to B", async ({ page }) => {
  const origin = process.env.RD_PREVIEW_ORIGIN || "http://127.0.0.1:18081";
  await page.goto(origin + "/app/remote-desktop/tests/session-ui-preview.html?workspace&rdp&files");
  const a = page.getByRole("tab", { name: "台式电脑，已连接" });
  const b = page.getByRole("tab", { name: "MacBook Pro，已连接" });
  await expect(a).toBeVisible();
  await page.getByRole("button", { name: "新建连接" }).click();
  await page.getByRole("button", { name: "连接另一台电脑" }).click();
  await expect(b).toBeVisible();
  await a.click();
  await page.locator(".remote-canvas:visible").click();
  await page.keyboard.press("Control+c");
  await b.click();
  await page.keyboard.press("Control+v");
  await expect.poll(() => page.evaluate(() => (window as any).__fileTrace))
    .toEqual(["copy", "install", "paste"]);
});
