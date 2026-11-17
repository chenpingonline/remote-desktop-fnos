// Frontend-only theme checks with an isolated, empty mocked workspace.
import { test, expect } from "@playwright/test";

test.use({
  baseURL: process.env.RD_APPEARANCE_URL || "http://127.0.0.1:18080/app/remote-desktop/",
  channel: "chrome",
  colorScheme: "light",
});

test("palettes apply in both themes and previews match the rendered interface", async ({ page }) => {
  await page.route("**/api/v1/**", (route) => route.fulfill({
    json: new URL(route.request().url()).pathname.endsWith("/me")
      ? { uid: "appearance-qa", name: "界面验收", admin: false, csrfToken: "qa" }
      : { items: [] },
  }));
  await page.goto("#/settings");
  await page.getByRole("tab", { name: "外观", exact: true }).click();
  await expect(page.getByRole("button", { name: "天蓝", exact: true })).toHaveAttribute("aria-pressed", "true");
  const backgrounds = new Set<string>();
  for (const theme of ["light", "dark"]) {
    await page.getByRole("button", { name: theme === "light" ? "浅色 明亮、清晰" : "深色 专注、低光" }).click();
    for (const palette of ["native", "warm", "cool"]) {
      const preview = page.locator(`.palette-choice-preview[data-palette="${palette}"]`);
      const button = preview.locator("..");
      await expect(button).toBeEnabled();
      await button.click();
      await expect(button).toHaveAttribute("aria-pressed", "true");
      await expect(page.locator("html")).toHaveAttribute("data-palette", palette);
      await expect(preview).toHaveAttribute("data-theme", theme);
      const colors = await preview.evaluate((node) => {
        const root = getComputedStyle(document.documentElement);
        const card = getComputedStyle(node);
        const read = (style: CSSStyleDeclaration, key: string) => style.getPropertyValue(key).trim();
        const luminance = (hex: string) => {
          const rgb = hex.slice(1).match(/../g)!.map((x) => parseInt(x, 16) / 255)
            .map((x) => x <= 0.04045 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4);
          return rgb[0] * 0.2126 + rgb[1] * 0.7152 + rgb[2] * 0.0722;
        };
        const bg = luminance(read(root, "--surface"));
        return {
          actual: ["--bg", "--surface", "--border"].map((key) => read(root, key)),
          preview: ["--bg", "--surface", "--border"].map((key) => read(card, key)),
          contrast: ["--text", "--text-2", "--text-3"].map((key) => {
            const fg = luminance(read(root, key));
            return (Math.max(bg, fg) + 0.05) / (Math.min(bg, fg) + 0.05);
          }),
        };
      });
      expect(colors.preview).toEqual(colors.actual);
      colors.contrast.forEach((ratio) => expect(ratio).toBeGreaterThanOrEqual(4.5));
      backgrounds.add(colors.actual[0]);
    }
  }
  expect(backgrounds.size).toBe(6);
  await page.getByRole("button", { name: "跟随系统 自动切换" }).click();
  await page.emulateMedia({ colorScheme: "light" });
  await expect(page.locator(".palette-choice-preview").first()).toHaveAttribute("data-theme", "light");
  await page.emulateMedia({ colorScheme: "dark" });
  await expect(page.locator(".palette-choice-preview").first()).toHaveAttribute("data-theme", "dark");
  await expect(page.locator("html")).toHaveAttribute("data-palette", "cool");
  await page.getByRole("button", { name: "丁香", exact: true }).click();
  await page.reload();
  await page.getByRole("tab", { name: "外观", exact: true }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.locator("html")).toHaveAttribute("data-palette", "cool");
  await expect(page.getByRole("button", { name: "丁香", exact: true })).toHaveAttribute("aria-pressed", "true");
  await expect(page.getByRole("button", { name: "冷雾 冷调雾蓝" })).toHaveAttribute("aria-pressed", "true");
  await page.screenshot({ path: "../build/qa-palettes-dark-cool.png", fullPage: true });
});

test("tooltips use rounded containers without arrows in both themes", async ({ page }) => {
  await page.route("**/api/v1/**", route => route.fulfill({
    json: new URL(route.request().url()).pathname.endsWith("/me")
      ? { uid: "appearance-qa", name: "界面验收", admin: false, csrfToken: "qa" }
      : { items: [] },
  }));
  await page.goto('#/connections');
  for (const theme of ['light', 'dark']) {
    await page.evaluate(value => { document.documentElement.dataset.theme = value; }, theme);
    await page.getByRole('button', { name: '网格视图', exact: true }).hover();
    const tooltip = page.locator('.el-popper.is-dark').filter({ hasText: '网格视图' });
    await expect(tooltip).toBeVisible();
    await expect(tooltip.locator('.el-popper__arrow')).toHaveCount(0);
    expect(await tooltip.evaluate(node => parseFloat(getComputedStyle(node).borderRadius))).toBeGreaterThan(0);
    await page.mouse.move(0, 0);
    await expect(tooltip).toBeHidden();
  }
});
