# 远程桌面 · Relay 前端

基于 `/Users/chenping/Project/deepseek/remote-desktop` 的 Vue 3、TypeScript、Tailwind CSS、Element Plus 界面，已接入本项目 Go 后端。

```bash
npm ci
npm run build
```

生产基础路径为 `/app/remote-desktop/`，使用 Hash 路由。构建结果为 `dist/`，由 Go 的静态服务托管。项目根目录 `scripts/dev.sh` 会构建前端并启动后端；开发模式下使用随机 `RD_DEV_TOKEN` 登录。

本地热更新：先启动后端，然后在此目录运行 `npm run dev`。Vite 将 `/app/remote-desktop/api`（含 WebSocket）代理到 `127.0.0.1:18080`。后端开发来源需与浏览器访问的 Vite 来源一致，例如 `scripts/dev.sh --origin http://127.0.0.1:4173`，访问 `http://127.0.0.1:4173/app/remote-desktop/`。

连接、分组、凭据、会话、证书信任及 WOL 都读取真实接口。浏览器仅保存按 NAS 用户隔离的外观偏好、收藏、标签及部分展示备注；不保存业务对象副本或密码。密码经同源接口交给后端加密保存，编辑时不回显账号、域和密码。

后端支持 RDP / Windows、跨平台 VNC（密码模式）、文本剪贴板与按连接配置的 UDP 9 网络唤醒。远程关机、凭据明文查看、浏览器 JSON 工作区恢复与手动删除历史记录未开放。页面不预置演示设备、模拟桌面或模拟开机结果。

远程桌面继续使用现有 `Desktop.vue` 及 Apache Guacamole 1.6.0 客户端。完整迁移说明见 `../docs/FRONTEND_MIGRATION.md`。

## 回归验收

在项目根目录执行 `python3 scripts/test-relay-ui.py`，脚本会构建并启动隔离的本地测试服务，运行 7 项浏览器集成用例，退出时清理服务与临时数据。测试需要 Playwright Chromium；不会连接真实 NAS 或远程电脑。

手动 TCP 检测会更新卡片/列表的可达状态和毫秒延迟，本次页面内保留结果；完整刷新后重新检测，目标变化或用户切换后清理。检测耗时包含 DNS 和 TCP 建连，不代表桌面操作延迟。

远程连接保留居中进度页，按证书、登录信息、通道和首帧的真实事件更新，不使用模拟进度计时器。关闭证书/登录弹窗后可继续，取消旧请求不会留下迟到会话。

VNC 连接使用自动目标模式，系统未获得可靠识别信息时显示“系统未识别”。表单不再填写系统/版本；Ctrl+V 与 Command+V 可在会话设置中按连接选择。

首页默认进入“我的连接”，不再提供概览。旧的 overview 启动偏好自动迁移为 connections，已有会话记录启动偏好仍保留。连接页按最近使用排序，并提供手动刷新入口。
