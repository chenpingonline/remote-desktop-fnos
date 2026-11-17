# 高分屏与分辨率调整

- 自动分辨率以窗口 CSS 尺寸、设备像素比、内容大小计算画面像素和 DPI，在 4096 × 2160、96–192 DPI 的既有范围内限制。2 倍屏上的标准内容优先使用 2 倍像素与 192 DPI。
- 缩放由独立裁切层提供实际布局边界，避免 Guacamole 原始尺寸的后代撑出滚动范围。六种固定分辨率均验收窗口内无额外滚动条。“铺满窗口（拉伸）”支持横纵不同缩放，鼠标坐标分别还原；全屏变化后重新测量画布。
- “像素原始尺寸”将一个远程像素映射到一个本机物理像素；“适应窗口”保持比例，因此固定分辨率与窗口比例不一致时会留边。
- 内容大小保存后通过显式重连生效，避免用有损前端放大替代 Windows UI 缩放。DPI 到达上限后，会降低画面尺寸以维持所选内容大小。固定分辨率也会协商 DPI。
- `patch-display-scaling.py` 对固定的 Guacamole 1.6.0 源码补充 FreeRDP 桌面缩放与物理尺寸，同时让 display-update 保留该缩放。补丁锚点变化会使构建失败，不能静默漏掉。
- 画质模式默认“均衡”：RDP 开启 GFX，并允许 Guacamole 自动选择有损图像更新；保留字体平滑。“清晰优先”恢复 `force-lossless=true` 与 `disable-gfx=true`，适合文字和代码。“流畅优先”同样开启 GFX 和自适应编码，并在自动分辨率下限制为最高 1920 × 1080（保持比例）；用户选择的固定分辨率优先。VNC 均衡使用 quality-level=6 / compress-level=2，流畅使用 4 / 1；实际编码能力取决于 VNC 服务端。模式按浏览器和协议保存，显式重连后生效，当前已连接会话不会被静默重配。旧客户端未传模式时后端默认均衡，非法模式返回 400。
  - 这不是固定 FPS 承诺，也没有新增 H.264 视频直传。真实播放性能仍需在目标 Windows / NAS / 网络下测量。
  - 依据：[Guacamole 配置](https://guacamole.apache.org/doc/gug/configuring-guacamole.html)。
- 分辨率选择先合并 600 ms；一次只等待一个远端尺寸确认，相邻发送至少间隔 1 秒。等待期间保留最新目标。8 秒未确认就停止发送，允许显式重连；迟到的正确尺寸仍可解除等待。实际分辨率读数来自远端，而不是所选目标。

依据：[Guacamole 显示设置](https://guacamole.apache.org/doc/gug/configuring-guacamole.html#display-settings)。Guacamole 原版 DPI 主要参与分辨率换算，不能把仅设置 DPI 当作 Windows 已应用 UI 缩放的证据。

验证：`node --experimental-strip-types --test web/unit/*.test.ts`、`go test ./internal/app`、`npm --prefix web run build`、原生 Docker 构建。浏览器模拟验收启动 `npm --prefix web run dev -- --port 18081`，在 `web` 下执行 `npx playwright test tests/display.spec.ts`。测试覆盖 2 倍像素、像素原始尺寸、内容大小待重连、慢响应时合并选择、无响应时停止请求。新增模拟 PNG 文字与单像素细线，实际 2 倍屏截图像素比对确认黑白细线未被插值抹灰；六种固定尺寸、铺满模式鼠标坐标、原始尺寸滚动边界、全屏切换均有回归测试。模拟界面不证明真实 Windows 接受了缩放，也不能确定此前断线原因；真实 RDP 清晰度和连续切换仍需安装后验证。

## 0.1.40 性能验证

- 图像指令中的 ASCII / Base64 改为分块解析；Unicode 字符计数、无效 UTF-8 和 4 MiB 上限校验保留。
- Apple M2 Pro、本机 64 KiB 图像 blob 微基准（两次）：约 452 µs/op → 36.4 µs/op；约 290 KB/op → 69.8 KB/op；29 → 8 次分配。该结果只代表解析器，不代表端到端帧率。
- 前端构建、13 项前端单测通过；后端 app / guac 测试除两项必须监听本地端口的既有测试外通过。两项测试因环境禁止绑定端口无法执行，非功能断言失败。真实远程视频播放帧率、网络带宽与 NAS CPU 尚未实测。
