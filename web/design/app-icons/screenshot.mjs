/**
 * 把 gallery.html 截成 preview/gallery.png（用于对外展示）。
 * 用法：node design/app-icons/screenshot.mjs
 */
import { chromium } from "playwright";
import { dirname, join } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const root = dirname(fileURLToPath(import.meta.url));
const browser = await chromium.launch();
const page = await browser.newPage({
  viewport: { width: 1440, height: 1000 },
  deviceScaleFactor: 2,
});
await page.goto(pathToFileURL(join(root, "gallery.html")).href);
await page.waitForLoadState("networkidle");
await page.evaluate(() => document.fonts.ready);
await page.screenshot({ path: join(root, "preview", "gallery.png"), fullPage: true });

await page.click("#themeToggle");
await page.waitForTimeout(350);
await page.screenshot({ path: join(root, "preview", "gallery-light.png"), fullPage: true });

await browser.close();
console.log("wrote preview/gallery.png and preview/gallery-light.png");
