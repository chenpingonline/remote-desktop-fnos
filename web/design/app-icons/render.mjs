/**
 * 渲染图标主稿：把 concepts/*.svg 以 1024×1024 透明背景输出到 build/master/
 * 用法：node design/app-icons/render.mjs
 */
import { chromium } from "playwright";
import { readFileSync, readdirSync, mkdirSync } from "node:fs";
import { dirname, join, basename } from "node:path";
import { fileURLToPath } from "node:url";

const root = dirname(fileURLToPath(import.meta.url));
const masterDir = join(root, "build", "master");
mkdirSync(masterDir, { recursive: true });

const files = readdirSync(join(root, "concepts"))
  .filter((f) => f.endsWith(".svg"))
  .sort();

const browser = await chromium.launch();
const page = await browser.newPage({
  viewport: { width: 512, height: 512 },
  deviceScaleFactor: 2,
});

for (const file of files) {
  const id = basename(file, ".svg");
  const svg = readFileSync(join(root, "concepts", file), "utf8");
  await page.setContent(
    `<style>html,body{margin:0;background:transparent}svg{display:block;width:512px;height:512px}</style>${svg}`,
  );
  await page.screenshot({
    path: join(masterDir, `${id}.png`),
    omitBackground: true,
  });
  console.log("rendered", id);
}

await browser.close();
