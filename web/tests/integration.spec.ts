// Uses a real Go backend with isolated QA data; optional synthetic RDP/guacd.
import { test, expect } from "@playwright/test";
import fs from "node:fs";
async function settleVisuals(page: import("@playwright/test").Page) {
  await page.evaluate(async () => {
    await new Promise(requestAnimationFrame);
    await new Promise(requestAnimationFrame);
    await Promise.all(
      document
        .getAnimations()
        .filter((a) => a.effect?.getTiming().iterations !== Infinity)
        .map((a) => a.finished.catch(() => {})),
    );
  });
}
const tokenPath = process.env.RD_QA_TOKEN_FILE;
test.skip(
  !tokenPath,
  "Start isolated local QA server and supply RD_QA_TOKEN_FILE",
);
test("new frontend persists backend objects and connects through certificate consent and WebSocket", async ({
  page,
}) => {
  const faults: string[] = [];
  page.on("pageerror", (e) => faults.push(e.message));
  await page.goto("#/connections");
  await page
    .getByLabel("开发访问令牌")
    .fill(fs.readFileSync(tokenPath!, "utf8"));
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(
    page.getByRole("link", { name: "我的连接", exact: false }).first(),
  ).toBeVisible();
  await expect(page.getByPlaceholder("搜索名称、地址或标签")).toBeVisible();
  await page
    .getByRole("button", { name: "新建连接", exact: true })
    .first()
    .click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  await dialog
    .getByRole("textbox", { name: "连接名称", exact: true })
    .fill("界面接入验收电脑");
  await dialog
    .getByRole("textbox", { name: "主机地址", exact: true })
    .fill("127.0.0.1");
  await dialog
    .getByRole("spinbutton", { name: "端口", exact: true })
    .fill("3390");
  await dialog
    .getByRole("switch", { name: "启用文本剪贴板" })
    .locator("..")
    .click();
  await dialog.getByRole("button", { name: "创建连接", exact: true }).click();
  await expect(dialog).not.toBeVisible();
  const card = page
    .locator(".connection-card")
    .filter({ hasText: "界面接入验收电脑" });
  await expect(card).toBeVisible();
  await page.reload();
  await expect(card).toBeVisible();
  const base = "/app/remote-desktop/api/v1";
  const me = await (
    await page.request.get(base + "/me", {
      headers: { "X-Remote-Desktop-Origin": "http://127.0.0.1:18080" },
    })
  ).json();
  const headers = {
    Origin: "http://127.0.0.1:18080",
    "X-Remote-Desktop-Origin": "http://127.0.0.1:18080",
    "X-CSRF-Token": me.csrfToken,
  };
  const connections = await (
    await page.request.get(base + "/connections")
  ).json();
  const connection = connections.items.find(
    (c: any) => c.name === "界面接入验收电脑",
  );
  expect(connection.protocol).toBe("rdp");
  expect(connection.settings.clipboard).toBe("text");
  await card.getByRole("button", { name: "快速连接", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "确认远程电脑证书" }),
  ).toBeVisible();
  await settleVisuals(page);
  await page.screenshot({
    path: new URL("../../build/relay-qa/certificate.png", import.meta.url)
      .pathname,
  });
  await page
    .getByRole("button", { name: "信任此电脑并连接", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "远程登录", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("textbox", { name: "远程用户名", exact: true })
    .fill("synthetic-user");
  await page.getByLabel("远程密码", { exact: true }).fill("synthetic-password");
  await page
    .getByRole("dialog", { name: "远程登录" })
    .getByRole("button", { name: "连接", exact: true })
    .click();
  await expect(page.locator(".desktop-tools .status")).toHaveText("已连接");
  await expect(page.getByLabel("远程密码")).toHaveCount(0);
  await page
    .getByRole("button", { name: "显示桌面工具栏", exact: true })
    .click();
  await page.getByRole("button", { name: "会话设置", exact: true }).click();
  await expect(page.getByRole("dialog", { name: "会话设置" })).toBeVisible();
  await settleVisuals(page);
  await page.screenshot({
    path: new URL("../../build/relay-qa/desktop.png", import.meta.url).pathname,
  });
  // Managing another page must preserve the live desktop.
  await page.getByRole("button", { name: "返回管理", exact: true }).click();
  await expect(page.locator(".relay-remote")).not.toBeVisible();
  await page
    .getByRole("button", { name: "返回远程会话 · 1", exact: true })
    .click();
  await expect(page.locator(".desktop-tools .status")).toHaveText("已连接");
  await page
    .getByRole("button", { name: "关闭远程工作区", exact: true })
    .click();
  await page.getByRole("button", { name: "断开并关闭", exact: true }).click();
  await expect(page.locator(".relay-remote")).not.toBeVisible();
  await page.goto("#/credentials");
  await page.getByRole("button", { name: "新建凭据", exact: true }).click();
  const credentialDialog = page.getByRole("dialog", { name: "新建凭据" });
  await credentialDialog
    .getByRole("textbox", { name: "凭据名称" })
    .fill("界面接入验收凭据");
  await credentialDialog
    .getByRole("textbox", { name: "用户名", exact: true })
    .fill("synthetic-user");
  await credentialDialog
    .locator("input[type=password]")
    .fill("unique-qa-password-no-local-storage");
  await credentialDialog
    .getByRole("button", { name: "保存凭据", exact: true })
    .click();
  await expect(credentialDialog).not.toBeVisible();
  await expect(
    page.locator(".credential-card").filter({ hasText: "界面接入验收凭据" }),
  ).toBeVisible();
  const storage = await page.evaluate(() => JSON.stringify(localStorage));
  expect(storage).not.toContain("unique-qa-password-no-local-storage");
  expect(storage).not.toContain("synthetic-password");
  await page.goto("#/settings");
  await page.getByRole("tab", { name: "外观", exact: true }).click();
  await page
    .getByRole("button", { name: "深色 专注、低光", exact: false })
    .click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await settleVisuals(page);
  await page.screenshot({
    path: new URL("../../build/relay-qa/settings-dark.png", import.meta.url)
      .pathname,
  });
  await page.setViewportSize({ width: 390, height: 750 });
  await page.goto("#/connections");
  await expect(card).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBeTruthy();
  await settleVisuals(page);
  await page.screenshot({
    path: new URL(
      "../../build/relay-qa/connections-mobile.png",
      import.meta.url,
    ).pathname,
  });
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.goto("#/settings");
  await page.getByRole("tab", { name: "安全与证书" }).click();
  await page.getByRole("button", { name: "查看与撤销" }).click();
  await expect(
    page.getByRole("heading", { name: "证书与信任", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "撤销信任", exact: true }).click();
  await page.getByRole("button", { name: "确认撤销", exact: true }).click();
  await expect(
    page.getByText("已撤销信任。下次连接将重新检查证书。"),
  ).toBeVisible();
  expect(faults).toEqual([]);
  // Only the isolated QA objects created by this test are removed.
  const credentials = await (
    await page.request.get(base + "/credentials")
  ).json();
  for (const c of credentials.items.filter(
    (c: any) => c.name === "界面接入验收凭据",
  ))
    await page.request.delete(base + "/credentials/" + c.id, { headers });
  await page.request.delete(base + "/connections/" + connection.id, {
    headers,
  });
});

test("group and WOL use server persistence; an edit conflict keeps the form open", async ({
  page,
}) => {
  await page.goto("#/connections");
  await page
    .getByLabel("开发访问令牌")
    .fill(fs.readFileSync(tokenPath!, "utf8"));
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await page.getByRole("button", { name: "新建分组", exact: true }).click();
  const groupDialog = page.getByRole("dialog", { name: "新建分组" });
  await groupDialog.getByLabel("分组名称").fill("接入验收分组");
  await groupDialog
    .getByRole("button", { name: "创建分组", exact: true })
    .click();
  await expect(groupDialog).not.toBeVisible();
  await page
    .getByRole("button", { name: "新建连接", exact: true })
    .first()
    .click();
  const dialog = page.getByRole("dialog", { name: "新建连接" });
  await dialog.getByLabel("连接名称", { exact: true }).fill("接入验收唤醒设备");
  await dialog.getByLabel("主机地址", { exact: true }).fill("192.0.2.10");
  await dialog
    .getByRole("switch", { name: "启用网络唤醒" })
    .locator("..")
    .click();
  await dialog.getByLabel("WOL MAC 地址").fill("02:11:22:33:44:55");
  await dialog.getByRole("button", { name: "创建连接", exact: true }).click();
  await expect(dialog).not.toBeVisible();
  const base = "/app/remote-desktop/api/v1",
    origin = "http://127.0.0.1:18080";
  const me = await (
    await page.request.get(base + "/me", {
      headers: { "X-Remote-Desktop-Origin": origin },
    })
  ).json();
  const headers = {
    Origin: origin,
    "X-Remote-Desktop-Origin": origin,
    "X-CSRF-Token": me.csrfToken,
  };
  const connections = await (
    await page.request.get(base + "/connections")
  ).json();
  const c = connections.items.find((c: any) => c.name === "接入验收唤醒设备");
  expect(c.groupId).toBeTruthy();
  expect(c.wol).toEqual({
    enabled: true,
    mac: "02:11:22:33:44:55",
    broadcast: "",
  });
  await page.goto("#/wol");
  await expect(
    page.getByRole("heading", { name: "接入验收唤醒设备", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "向 接入验收唤醒设备 发送关机指令" }),
  ).toBeDisabled();
  await page.reload();
  await expect(
    page.getByRole("heading", { name: "接入验收唤醒设备", exact: true }),
  ).toBeVisible();
  await page.goto("#/connections");
  const card = page
    .locator(".connection-card")
    .filter({ hasText: "接入验收唤醒设备" });
  await card.getByRole("button", { name: "更多操作" }).click();
  await page.getByRole("menuitem", { name: "编辑连接" }).click();
  const edit = page.getByRole("dialog", { name: "编辑连接" });
  await expect(edit).toBeVisible();
  const concurrent = await page.request.patch(base + "/connections/" + c.id, {
    headers,
    data: { ...c, name: "其他窗口更新后的设备" },
  });
  expect(concurrent.ok()).toBeTruthy();
  await edit.getByLabel("连接名称", { exact: true }).fill("不应覆盖其他窗口");
  await edit.getByRole("button", { name: "保存修改", exact: true }).click();
  await expect(page.getByText("内容已改变，请刷新后重试")).toBeVisible();
  await expect(edit).toBeVisible();
  const current = await (
    await page.request.get(base + "/connections/" + c.id)
  ).json();
  expect(current.name).toBe("其他窗口更新后的设备");
  await page.request.delete(base + "/connections/" + c.id, { headers });
  await page.request.delete(base + "/groups/" + c.groupId, { headers });
});

test("TCP probe updates card and list, survives polling, clears old latency on failure", async ({
  page,
}) => {
  const net = await import("node:net");
  const listener = net.createServer((socket) => socket.end());
  await new Promise<void>((resolve) =>
    listener.listen(0, "127.0.0.1", resolve),
  );
  let listening = true;
  const port = (listener.address() as import("node:net").AddressInfo).port;
  try {
    await page.goto("#/connections");
    await page
      .getByLabel("开发访问令牌")
      .fill(fs.readFileSync(tokenPath!, "utf8"));
    await page.getByRole("button", { name: "登录", exact: true }).click();
    await expect(page.getByPlaceholder("搜索名称、地址或标签")).toBeVisible();
    const base = "/app/remote-desktop/api/v1",
      origin = "http://127.0.0.1:18080";
    const me = await (
      await page.request.get(base + "/me", {
        headers: { "X-Remote-Desktop-Origin": origin },
      })
    ).json();
    const headers = {
      Origin: origin,
      "X-Remote-Desktop-Origin": origin,
      "X-CSRF-Token": me.csrfToken,
    };
    const response = await page.request.post(base + "/connections", {
      headers,
      data: {
        name: "TCP状态验收",
        protocol: "rdp",
        targetOs: "windows",
        host: "127.0.0.1",
        port,
      },
    });
    expect(response.ok()).toBeTruthy();
    const c = await response.json();
    try {
      await page.reload();
      const card = page
        .locator(".connection-card")
        .filter({ hasText: "TCP状态验收" });
      await expect(card.getByRole("status")).toHaveText("未检测");
      const probe = async () => {
        await card.getByRole("button", { name: "更多操作" }).click();
        await page.getByRole("menuitem", { name: "检查 TCP 连通性" }).click();
      };
      const succeeded = page.waitForResponse(
        (r) => r.url().endsWith("/" + c.id + "/probe") && r.status() === 200,
      );
      await probe();
      const result = await (await succeeded).json();
      expect(result.reachable).toBe(true);
      expect(result.latencyMs).toBeGreaterThanOrEqual(0);
      await expect(card.getByRole("status")).toHaveText("端口可达");
      await expect(card).toContainText(/\d+\.\d+ ms/);
      const previous = await card
        .locator("[title]")
        .filter({ hasText: /ms/ })
        .innerText();
      const limited = page.waitForResponse(
        (r) => r.url().endsWith("/" + c.id + "/probe") && r.status() === 429,
      );
      await probe();
      await limited;
      await expect(card.getByRole("status")).toHaveText("端口可达");
      await expect(card).toContainText(previous);
      // The real background refresh must not overwrite this page's probe cache.
      await page.waitForResponse(
        (r) =>
          r.request().method() === "GET" &&
          new URL(r.url()).pathname === base + "/connections",
        { timeout: 15000 },
      );
      await expect(card.getByRole("status")).toHaveText("端口可达");
      await expect(card).toContainText(previous);
      await listener.close();
      listening = false;
      const failed = page.waitForResponse(
        (r) => r.url().endsWith("/" + c.id + "/probe") && r.status() === 422,
      );
      await probe();
      await failed;
      await expect(card.getByRole("status")).toHaveText("端口不可达");
      await expect(card).toContainText("检测失败");
      await expect(card).not.toContainText(/\d+\.\d+ ms/);
      await settleVisuals(page);
      await page.screenshot({
        path: new URL(
          "../../build/relay-qa/probe-unreachable.png",
          import.meta.url,
        ).pathname,
      });
      await page.getByRole("button", { name: "列表视图" }).click();
      const row = page.getByRole("row").filter({ hasText: "TCP状态验收" });
      await expect(row).toContainText("端口不可达");
      await expect(row).toContainText("检测失败");
      // Changing the actual target invalidates cached measurements.
      const saved = await page.request.patch(base + "/connections/" + c.id, {
        headers,
        data: { ...c, port: port === 65535 ? port - 1 : port + 1 },
      });
      expect(saved.ok()).toBeTruthy();
      await page.goto("#/connections");
      await page.getByRole("button", { name: "刷新连接列表" }).click();
      await expect(
        page.getByRole("button", { name: "刷新连接列表" }),
      ).toBeEnabled();
      await page.goto("#/connections");
      await page.getByRole("button", {name:"网格视图"}).click();
      await expect(card.getByRole("status")).toHaveText("未检测");
      await expect(card).toContainText("未测延迟");
    } finally {
      await page.request.delete(base + "/connections/" + c.id, { headers });
    }
  } finally {
    if (listening) listener.close();
  }
});

test("connection progress follows certificate, login, channel and rendered first frame; cancellation ignores late sessions", async ({
  page,
}) => {
  let certificateCompletedAt = 0;
  page.on("response", (r) => {
    if (new URL(r.url()).pathname.endsWith("/certificate"))
      certificateCompletedAt = Date.now();
  });
  const control = "http://127.0.0.1:3392";
  await page.request.post(control + "/reset");
  await page.request.post(control + "/hold-handshake");
  await page.request.post(control + "/hold-frame");
  await page.goto("#/connections");
  await page
    .getByLabel("开发访问令牌")
    .fill(fs.readFileSync(tokenPath!, "utf8"));
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(page.getByPlaceholder("搜索名称、地址或标签")).toBeVisible();
  const base = "/app/remote-desktop/api/v1",
    origin = "http://127.0.0.1:18080";
  const me = await (
    await page.request.get(base + "/me", {
      headers: { "X-Remote-Desktop-Origin": origin },
    })
  ).json();
  const headers = {
    Origin: origin,
    "X-Remote-Desktop-Origin": origin,
    "X-CSRF-Token": me.csrfToken,
  };
  const response = await page.request.post(base + "/connections", {
    headers,
    data: {
      name: "连接进度验收",
      protocol: "rdp",
      targetOs: "windows",
      host: "127.0.0.1",
      port: 3390,
    },
  });
  const c = await response.json();
  let releaseCertificate = () => {},
    releaseSession = () => {};
  try {
    await page.reload();
    const card = page
      .locator(".connection-card")
      .filter({ hasText: "连接进度验收" });
    const progress = page.getByRole("region", { name: "连接进度" });
    const certificateGate = new Promise<void>(
      (resolve) => (releaseCertificate = resolve),
    );
    await page.route(
      "**/connections/" + c.id + "/certificate",
      async (route) => {
        await certificateGate;
        await route.continue();
      },
    );
    await card.getByRole("button", { name: "快速连接" }).click();
    await expect(progress.getByRole("heading")).toHaveText("正在检查远程证书");
    await expect(progress.locator('[data-step="0"]')).toHaveAttribute(
      "aria-current",
      "step",
    );
    releaseCertificate();
    await expect(
      page.getByRole("heading", { name: "确认远程电脑证书" }),
    ).toBeVisible();
    await page.getByRole("button", { name: "取消", exact: true }).click();
    await expect(progress.getByRole("heading")).toHaveText("等待确认远程证书");
    await progress
      .getByRole("button", { name: "确认远程证书", exact: true })
      .click();
    await page.getByRole("button", { name: "信任此电脑并连接" }).click();
    await expect(
      page.getByRole("heading", { name: "远程登录", exact: true }),
    ).toBeVisible();
    await page
      .getByRole("dialog", { name: "远程登录" })
      .getByRole("button", { name: "取消", exact: true })
      .click();
    await expect(progress.getByRole("heading")).toHaveText("等待输入登录信息");
    await progress.getByRole("button", { name: "输入登录信息" }).click();
    await page.getByLabel("远程用户名", { exact: true }).fill("synthetic-user");
    await page
      .getByLabel("远程密码", { exact: true })
      .fill("synthetic-password");
    await page
      .getByRole("dialog", { name: "远程登录" })
      .getByRole("button", { name: "连接", exact: true })
      .click();
    await expect(progress.getByRole("heading")).toHaveText("正在连接远程电脑");
    await expect(progress.locator('[data-step="2"]')).toHaveAttribute(
      "aria-current",
      "step",
    );
    await page.request.post(control + "/release-handshake");
    await expect(progress.getByRole("heading")).toHaveText("正在等待远程画面");
    await expect(progress.locator('[data-step="3"]')).toHaveAttribute(
      "aria-current",
      "step",
    );
    // No sync/frame has been sent: an open channel alone must not hide progress.
    const waiting = await (await page.request.get(base + "/sessions")).json();
    expect(waiting.items.find((s: any) => s.connectionId === c.id).state).toBe(
      "CONNECTING",
    );
    await settleVisuals(page);
    await page.screenshot({
      path: new URL(
        "../../build/relay-qa/progress-waiting-frame.png",
        import.meta.url,
      ).pathname,
    });
    await page.setViewportSize({ width: 390, height: 650 });
    await expect(progress).toBeVisible();
    await expect(
      progress.getByRole("button", { name: "返回管理" }),
    ).toBeInViewport();
    await page.screenshot({
      path: new URL("../../build/relay-qa/progress-mobile.png", import.meta.url)
        .pathname,
    });
    await page.request.post(control + "/release-frame");
    await expect(progress).not.toBeVisible();
    await expect(page.getByLabel("远程桌面画布")).toBeVisible();
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.getByRole("button", { name: "关闭远程工作区" }).click();
    await page.getByRole("button", { name: "断开并关闭" }).click();
    // A second attempt uses saved credentials and holds the real POST response.
    const saved = await page.request.patch(base + "/connections/" + c.id, {
      headers,
      data: {
        ...c,
        credential: {
          username: "synthetic-user",
          password: "synthetic-password",
          domain: "",
        },
      },
    });
    expect(saved.ok()).toBeTruthy();
    await page.goto("#/connections");
    await page.getByRole("button", { name: "刷新连接列表" }).click();
    await expect(
      page.getByRole("button", { name: "刷新连接列表" }),
    ).toBeEnabled();
    await page.goto("#/connections");
    await expect
      .poll(() => Date.now() - certificateCompletedAt)
      .toBeGreaterThan(1100);
    const sessionGate = new Promise<void>(
      (resolve) => (releaseSession = resolve),
    );
    await page.route("**/api/v1/sessions", async (route) => {
      if (route.request().method() === "POST") {
        await sessionGate;
      }
      await route.continue();
    });
    await card.getByRole("button", { name: "快速连接" }).click();
    await expect(progress.getByRole("heading")).toHaveText("正在连接远程电脑");
    await progress
      .getByRole("button", { name: "返回管理", exact: true })
      .click();
    await expect(page.locator(".relay-remote")).not.toBeVisible();
    const late = page.waitForResponse(
      (r) =>
        r.request().method() === "POST" &&
        new URL(r.url()).pathname === base + "/sessions",
    );
    releaseSession();
    await late;
    await expect
      .poll(async () => {
        const history = await (
          await page.request.get(base + "/sessions")
        ).json();
        return history.items.filter(
          (s: any) =>
            s.connectionId === c.id &&
            ["CREATED", "CONNECTING", "ACTIVE"].includes(s.state),
        ).length;
      })
      .toBe(0);
    await expect(page.locator(".relay-remote")).not.toBeVisible();
  } finally {
    releaseCertificate();
    releaseSession();
    await page.request.post(control + "/reset");
    await page.request.delete(base + "/connections/" + c.id, { headers });
  }
});

test("VNC skips certificate checks and displays the actual API failure with retry", async ({
  page,
}) => {
  await page.goto("#/connections");
  await page
    .getByLabel("开发访问令牌")
    .fill(fs.readFileSync(tokenPath!, "utf8"));
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(page.getByPlaceholder("搜索名称、地址或标签")).toBeVisible();
  const base = "/app/remote-desktop/api/v1",
    origin = "http://127.0.0.1:18080";
  const me = await (
    await page.request.get(base + "/me", {
      headers: { "X-Remote-Desktop-Origin": origin },
    })
  ).json();
  const headers = {
    Origin: origin,
    "X-Remote-Desktop-Origin": origin,
    "X-CSRF-Token": me.csrfToken,
  };
  const c = await (
    await page.request.post(base + "/connections", {
      headers,
      data: {
        name: "VNC进度失败验收",
        protocol: "vnc",
        targetOs: "macos",
        host: "127.0.0.1",
        port: 5900,
        credential: {
          password: "synthetic-password",
          username: "",
          domain: "",
        },
      },
    })
  ).json();
  let attempts = 0,
    certificates = 0;
  page.on("request", (r) => {
    if (r.url().includes("/" + c.id + "/certificate")) certificates++;
  });
  await page.route("**/api/v1/sessions", async (route) => {
    if (route.request().method() !== "POST") {
      await route.continue();
      return;
    }
    attempts++;
    await route.fulfill({
      status: 503,
      contentType: "application/json",
      body: JSON.stringify({
        error: {
          code: "QA_REJECTION",
          message: attempts === 1 ? "模拟后端拒绝建立连接" : "模拟重试仍被拒绝",
        },
      }),
    });
  });
  try {
    await page.reload();
    const card = page
      .locator(".connection-card")
      .filter({ hasText: "VNC进度失败验收" });
    await card.getByRole("button", { name: "快速连接" }).click();
    const progress = page.getByRole("region", { name: "连接进度" });
    await expect(progress.getByRole("heading")).toHaveText("远程连接未完成");
    await expect(progress).toContainText("VNC 跳过此步");
    await expect(progress.getByRole("alert")).toHaveText(
      "模拟后端拒绝建立连接",
    );
    await expect(progress.locator('[data-step="2"]')).toHaveAttribute(
      "aria-current",
      "step",
    );
    expect(certificates).toBe(0);
    await progress.getByRole("button", { name: "重试连接" }).click();
    await expect(progress.getByRole("alert")).toHaveText("模拟重试仍被拒绝");
    expect(attempts).toBe(2);
    await settleVisuals(page);
    await page.screenshot({
      path: new URL("../../build/relay-qa/progress-failed.png", import.meta.url)
        .pathname,
    });
    await progress
      .getByRole("button", { name: "返回管理", exact: true })
      .click();
    await expect(page.locator(".relay-remote")).not.toBeVisible();
  } finally {
    await page.request.delete(base + "/connections/" + c.id, { headers });
  }
});

test("VNC uses an automatic target without OS fields or guessed Mac identity, and supports Ctrl/Command paste", async ({
  page,
}) => {
  await page.addInitScript(() =>
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: {
        readText: async () => {
          throw new Error("QA clipboard denied");
        },
      },
    }),
  );
  await page.goto("#/connections");
  await page
    .getByLabel("开发访问令牌")
    .fill(fs.readFileSync(tokenPath!, "utf8"));
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(page.getByPlaceholder("搜索名称、地址或标签")).toBeVisible();
  await page
    .getByRole("button", { name: "新建连接", exact: true })
    .first()
    .click();
  const dialog = page.getByRole("dialog", { name: "新建连接" });
  await expect(dialog.getByText("操作系统", { exact: true })).toHaveCount(0);
  await expect(dialog.getByText("系统版本", { exact: true })).toHaveCount(0);
  await dialog.getByRole("button", { name: "VNC 跨平台桌面" }).click();
  await expect(dialog.getByLabel("端口", { exact: true })).toHaveValue("5900");
  await dialog.getByLabel("连接名称", { exact: true }).fill("VNC自动模式验收");
  await dialog.getByLabel("主机地址", { exact: true }).fill("127.0.0.1");
  await dialog
    .getByRole("switch", { name: "启用文本剪贴板" })
    .locator("..")
    .click();
  await dialog.getByRole("button", { name: "创建连接", exact: true }).click();
  await expect(dialog).not.toBeVisible();
  const card = page
    .locator(".connection-card")
    .filter({ hasText: "VNC自动模式验收" });
  await expect(card).toContainText("系统未识别");
  await expect(card).not.toContainText("macOS");
  const base = "/app/remote-desktop/api/v1",
    origin = "http://127.0.0.1:18080";
  const me = await (
    await page.request.get(base + "/me", {
      headers: { "X-Remote-Desktop-Origin": origin },
    })
  ).json();
  const headers = {
    Origin: origin,
    "X-Remote-Desktop-Origin": origin,
    "X-CSRF-Token": me.csrfToken,
  };
  const all = await (await page.request.get(base + "/connections")).json();
  const c = all.items.find((c: any) => c.name === "VNC自动模式验收");
  expect(c.targetOs).toBe("auto");
  try {
    await card.getByRole("button", { name: "快速连接" }).click();
    const login = page.getByRole("dialog", { name: "远程登录" });
    await expect(login.getByLabel("远程用户名")).toHaveCount(0);
    await login.getByLabel("远程密码").fill("synthetic-vnc-password");
    await login.getByRole("button", { name: "连接", exact: true }).click();
    await expect(page.locator(".desktop-tools .status")).toHaveText("已连接");
    const canvas = page.getByLabel("远程桌面画布");
    await canvas.click();
    await canvas.press("Control+v");
    await expect
      .poll(
        () =>
          JSON.parse(
            fs.readFileSync(
              new URL("../../build/qa-mock-stats.json", import.meta.url),
              "utf8",
            ),
          ).pasteModifier,
      )
      .toBe("ctrl");
    await page
      .getByRole("button", { name: "显示桌面工具栏", exact: true })
      .click();
    await page.getByRole("button", { name: "会话设置", exact: true }).click();
    await page.getByRole("combobox", { name: "远端粘贴快捷键" }).click();
    await page
      .getByRole("option", { name: "Command+V · macOS", exact: true })
      .click();
    await canvas.click();
    await canvas.press("Control+v");
    await expect
      .poll(
        () =>
          JSON.parse(
            fs.readFileSync(
              new URL("../../build/qa-mock-stats.json", import.meta.url),
              "utf8",
            ),
          ).pasteModifier,
      )
      .toBe("command");
    expect(
      await page.evaluate(
        (id) => localStorage.getItem("remote-desktop.vnc-paste." + id),
        c.id,
      ),
    ).toBe("command");
    await page.getByRole("button", { name: "关闭远程工作区" }).click();
    await page.getByRole("button", { name: "断开并关闭" }).click();
    await page.reload();
    await expect(card).toContainText("系统未识别");
    await settleVisuals(page);
    await page.screenshot({
      path: new URL("../../build/relay-qa/vnc-auto.png", import.meta.url)
        .pathname,
    });
  } finally {
    await page.request.delete(base + "/connections/" + c.id, { headers });
  }
});

test("home opens connections, old overview preferences migrate, and navigation has no overview", async ({
  page,
}) => {
  await page.goto("");
  await page
    .getByLabel("开发访问令牌")
    .fill(fs.readFileSync(tokenPath!, "utf8"));
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(page).toHaveURL(/#\/connections$/);
  await expect(page.getByPlaceholder("搜索名称、地址或标签")).toBeVisible();
  await expect(
    page.getByRole("link", { name: "概览", exact: true }),
  ).toHaveCount(0);
  const me = await (
    await page.request.get("/app/remote-desktop/api/v1/me", {
      headers: { "X-Remote-Desktop-Origin": "http://127.0.0.1:18080" },
    })
  ).json();
  const key = "remote-desktop.relay.preferences." + me.uid;
  await page.evaluate((key) => {
    const saved = JSON.parse(localStorage.getItem(key) || "{}");
    saved.settings = { ...saved.settings, startPage: "overview" };
    localStorage.setItem(key, JSON.stringify(saved));
  }, key);
  await page.goto("http://127.0.0.1:18080/app/remote-desktop/");
  await expect(page).toHaveURL(/#\/connections$/);
  await expect
    .poll(() =>
      page.evaluate(
        (key) =>
          JSON.parse(localStorage.getItem(key) || "{}").settings?.startPage,
        key,
      ),
    )
    .toBe("connections");
  await page.keyboard.press("Control+k");
  const palette = page.getByRole("dialog", { name: "命令面板" });
  await expect(palette).toBeVisible();
  await expect(palette.getByText("概览", { exact: true })).toHaveCount(0);
  await page.keyboard.press("Escape");
  await page.goto("#/settings");
  await page.locator(".el-select").filter({has:page.getByRole("combobox",{name:"启动页面"})}).click();
  await expect(
    page.getByRole("option", { name: "概览", exact: true }),
  ).toHaveCount(0);
  await page.keyboard.press("Escape");
  await page.evaluate((key) => {
    const saved = JSON.parse(localStorage.getItem(key) || "{}");
    saved.settings.startPage = "sessions";
    localStorage.setItem(key, JSON.stringify(saved));
  }, key);
  await page.goto("http://127.0.0.1:18080/app/remote-desktop/");
  await expect(page).toHaveURL(/#\/sessions$/);
  await page.goto("#/credentials");
  await page.reload();
  await expect(page).toHaveURL(/#\/credentials$/);
  await page.goto("#/overview");
  await expect(page).toHaveURL(/#\/connections$/);
  await page.setViewportSize({ width: 390, height: 750 });
  await expect(page.getByPlaceholder("搜索名称、地址或标签")).toBeVisible();
  await page.getByRole("button", { name: "打开导航" }).click();
  await expect(
    page.getByRole("link", { name: "概览", exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByRole("link", { name: "我的连接", exact: false }).first(),
  ).toBeVisible();
  await page.setViewportSize({ width: 1280, height: 800 });
  await settleVisuals(page);
  await page.screenshot({
    path: new URL("../../build/relay-qa/connections-home.png", import.meta.url)
      .pathname,
  });
});

for (const protocol of ["vnc", "rdp"] as const) {
  test(`${protocol} remote login remembers credentials only when selected and reuses them after reload`, async ({ page }) => {
    await page.goto("#/connections");
    await page.getByLabel("开发访问令牌").fill(fs.readFileSync(tokenPath!, "utf8"));
    await page.getByRole("button", { name: "登录", exact: true }).click();
    await expect(page.getByPlaceholder("搜索名称、地址或标签")).toBeVisible();
    const base = "/app/remote-desktop/api/v1", origin = "http://127.0.0.1:18080";
    const me = await (await page.request.get(base + "/me", { headers: { "X-Remote-Desktop-Origin": origin } })).json();
    const headers = { Origin: origin, "X-CSRF-Token": me.csrfToken };
    const c = await (await page.request.post(base + "/connections", { headers, data: {
      name: `${protocol}记住凭据验收`, protocol, host: "127.0.0.1", port: protocol === "vnc" ? 5900 : 3390,
    } })).json();
    const credentials = async () => (await (await page.request.get(base + "/credentials")).json()).items;
    const originalCount = (await credentials()).length;
    const card = page.locator(".connection-card").filter({ hasText: c.name });
    const loginDialog = page.getByRole("dialog", { name: "远程登录", exact: true });
    const remember = loginDialog.getByRole("checkbox", { name: "记住凭据，下次自动使用" });
    let savedId = "";
    async function openLogin() {
      await card.getByRole("button", { name: "快速连接", exact: true }).click();
      if (protocol === "rdp") {
        const trust = page.getByRole("button", { name: "信任此电脑并连接", exact: true });
        // First RDP connection requires explicit trust of the synthetic cert.
        await Promise.race([trust.waitFor(), loginDialog.waitFor()]);
        if (await trust.isVisible()) await trust.click();
      }
      await expect(loginDialog).toBeVisible();
      await expect(remember).not.toBeChecked();
      if (protocol === "rdp") await loginDialog.getByLabel("远程用户名", { exact: true }).fill("synthetic-remember-user");
      await loginDialog.getByLabel("远程密码", { exact: true }).fill("synthetic-remember-password");
    }
    async function closeDesktop() {
      await page.getByRole("button", { name: "关闭远程工作区", exact: true }).click();
      await page.getByRole("button", { name: "断开并关闭", exact: true }).click();
      await expect(page.locator(".relay-remote")).not.toBeVisible();
    }
    try {
      await page.reload();
      await openLogin();
      await loginDialog.getByRole("button", { name: "连接", exact: true }).click();
      await expect(page.locator(".desktop-tools .status")).toHaveText("已连接");
      expect((await (await page.request.get(base + "/connections/" + c.id)).json()).credentialId).toBeFalsy();
      expect((await credentials()).length).toBe(originalCount);
      await closeDesktop();
      await page.reload();
      await openLogin();
      await loginDialog.getByText("记住凭据，下次自动使用", { exact: true }).click();
      await expect(remember).toBeChecked();
      await expect(loginDialog).toContainText("凭据将加密保存在 NAS，并关联当前连接。");
      if (protocol === "vnc") {
        // A failed atomic save must retain the input and never create a credential.
        await page.route("**/api/v1/connections/" + c.id, async route => {
          if (route.request().method() !== "PATCH") return route.continue();
          await route.fulfill({ status: 409, contentType: "application/json", body: JSON.stringify({ error: { code: "SAVE_FAILED", message: "连接及凭据未能保存，请刷新后重试" } }) });
        });
      await loginDialog.getByRole("button", { name: "连接", exact: true }).click();
        await expect(loginDialog.getByRole("alert")).toContainText("连接及凭据未能保存");
        await expect(remember).toBeChecked();
        expect((await credentials()).length).toBe(originalCount);
        await page.unroute("**/api/v1/connections/" + c.id);
        await page.setViewportSize({ width: 390, height: 760 });
        await settleVisuals(page);
        await expect(loginDialog.getByText("记住凭据，下次自动使用", { exact: true })).toBeInViewport();
        await page.screenshot({ path: new URL("../../build/relay-qa/remember-vnc-narrow.png", import.meta.url).pathname });
        await page.setViewportSize({ width: 1280, height: 800 });
      }
      await loginDialog.getByRole("button", { name: "连接", exact: true }).click();
      await expect(page.locator(".desktop-tools .status")).toHaveText("已连接");
      await expect(loginDialog).not.toBeVisible();
      const saved = await (await page.request.get(base + "/connections/" + c.id)).json();
      savedId = saved.credentialId;
      expect(savedId).toBeTruthy();
      const savedCredentials = await credentials();
      expect(savedCredentials.length).toBe(originalCount + 1);
      expect(JSON.stringify(savedCredentials)).not.toContain("synthetic-remember-password");
      expect(await page.evaluate(() => JSON.stringify(localStorage))).not.toContain("synthetic-remember-password");
      await closeDesktop();
      await page.reload();
      await card.getByRole("button", { name: "快速连接", exact: true }).click();
      await expect(page.locator(".desktop-tools .status")).toHaveText("已连接");
      await expect(loginDialog).not.toBeVisible();
      expect((await credentials()).length).toBe(originalCount + 1);
      await closeDesktop();
    } finally {
      await page.request.delete(base + "/connections/" + c.id, { headers });
      if (savedId) await page.request.delete(base + "/credentials/" + savedId, { headers });
    }
  });
}

test("general settings retain basic preferences without the removed launch switch", async ({ page }) => {
  const launchRequests: string[] = [];
  page.on("request", request => {
    if (request.url().includes("/launch-settings")) launchRequests.push(request.url());
  });
  await page.goto("#/settings");
  await page.getByLabel("开发访问令牌").fill(fs.readFileSync(tokenPath!, "utf8"));
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(page.getByRole("heading", { name: "基础偏好", exact: true })).toBeVisible();
  await expect(page.getByRole("combobox", { name: "应用打开方式" })).toHaveCount(0);
  await expect(page.getByRole("combobox", { name: "启动页面" })).toBeAttached();
  await expect(page.getByRole("switch", { name: "断开前确认" })).toBeAttached();
  await page.reload();
  await expect(page.getByRole("heading", { name: "基础偏好", exact: true })).toBeVisible();
  expect(launchRequests).toHaveLength(0);
  await page.setViewportSize({ width: 390, height: 760 });
  await settleVisuals(page);
  await page.screenshot({ path: new URL("../../build/relay-qa/settings-without-launch-switch.png", import.meta.url).pathname });
});
