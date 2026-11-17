# 远程桌面 · 应用图标方案

## 当前采用：墨绿黑底 · 青柠双屏

2026-11-11 用户选定双屏与上行弧形箭头图案，配色与“连接中”页面一致。
原始透明 PNG 保存在 [`selected/dual-screen-lime/master.png`](selected/dual-screen-lime/master.png)，
各尺寸导出保存在 [`selected/dual-screen-lime/exports/`](selected/dual-screen-lime/exports/)。

执行 `python3 scripts/export-app-icon.py`（仓库根目录）可重新导出并更新飞牛包图标与网页 favicon。
该脚本只缩放母版，不重绘或修改图案。下方六套 SVG 为历史探索方案，不代表当前选择。

为 fnOS 应用「远程桌面 / Relay」设计的 6 套应用图标。沿用产品现有的青柠绿 `#D7FB62` 与墨绿 `#182006`，在同一套 512 网格、圆角和描边规则下给出不同语义方向，可直接挑选一套替换进应用。

- 可交互浏览：用浏览器打开 [`gallery.html`](gallery.html)（右上角可切换深浅色）
- 一张图总览：[`preview/contact-sheet.png`](preview/contact-sheet.png)
- SVG 真源：[`concepts/`](concepts/)

## 六个方向

| # | 名称 | 语义 | 优点 | 风险 | 定位 |
| --- | --- | --- | --- | --- | --- |
| 01 | 指针屏 Pointer | 屏幕 + 光标 | 最简洁、对比最强，32px 仍可读 | 小图可能被读成截图 / 标注工具 | **主推** |
| 02 | 信号屏 Signal | 屏幕 + 信号弧 | 「远程 / 串流」一眼成立，浅色底也稳 | 近似 Wi-Fi / 投屏 | 备选 |
| 03 | 双向屏 Swap | 屏幕 + 双向箭头 | 强调实时交互与会话 | 元素多，16px 会糊 | 场景化 |
| 04 | 双屏 Dual | 双屏叠加 + 光标 | 最接近「控制另一台设备」 | 结构复杂，小尺寸偏密 | 场景化 |
| 05 | R 标 Monogram | Relay 首字母 | 最克制、最像品牌标志 | 无功能语义 | 品牌向 |
| 06 | 外联 Remote | 屏幕 + 向外箭头 | 经典构图，零解释成本 | 有「外链 / 新窗口」歧义，16px 丢细节 | 稳健 |

`concepts/00-current.svg` 是当前线上 favicon（青柠底 + 环形箭头），只作为对照，不参与导出。

## 设计规范

- **画布**：512 × 512，图形距边缘 ≥ 64（安全区），避免系统圆角裁切。
- **圆角**：底板圆角 116（22.7%），接近 iOS / macOS squircle 比例；满版铺底，系统再加遮罩也不露边。
- **描边**：主描边 24–30（画布的 5%–6%），是 16px 下仍可见的下限；不使用 20 以下的细线和文字。
- **色彩**：
  - 深色底板渐变 `#18211A → #090D0A`
  - 青柠绿 `#D7FB62 → #C8EF4C`
  - 墨绿（浅底上的图形）`#182006`
  - 品牌绿（04 之外的备选底板）`#1F6F4A → #145137`
- **图层**：一块底板 + 1–2 个主形状；细节（指针切空、信号弧）都用同一色板，不叠加阴影或多层渐变。
- **光学修正**：屏幕类图形统一上移约 4–8，抵消底座带来的视觉重心下沉；小尺寸优先保留「屏幕 + 一个符号」。

## 推荐

1. **01 指针屏（主推）**：语义、识别度与品牌色最平衡，和产品内的 Lucide 线性图标体系也能呼应。
2. **06 外联（稳健）**：如果希望零解释成本就选经典构图，但建议补一版 16px 简化稿。
3. **05 R 标（品牌）**：适合作为长期品牌标志，或与 01 组成「标志 + 功能图形」两层体系。
4. **02 / 03 / 04**：分别适合强调串流、双向会话、控制另一端的场景，也可用作运营图或启动页图形。

## 接入应用

以主推「指针屏」为例，在仓库根目录执行（**先确认已选中方案再覆盖**）：

```bash
cp web/design/app-icons/exports/01-pointer/fnos/icon_256.png \
   packaging/remote-desktop/app/ui/images/icon_256.png
cp web/design/app-icons/exports/01-pointer/fnos/icon_64.png \
   packaging/remote-desktop/app/ui/images/icon_64.png
cp web/design/app-icons/exports/01-pointer/fnos/ICON.PNG \
   packaging/remote-desktop/ICON.PNG
cp web/design/app-icons/concepts/01-pointer.svg \
   web/public/favicon.svg
```

替换后重新打 FPK，并在 fnOS 应用中心实测图标显示尺寸与圆角裁切。网页 favicon 建议在 16 / 32 的浏览器标签下复查；如果 16px 细节丢失，可以只保留「屏幕 + 光标」两个形状。

## 重新生成

```bash
cd web
node design/app-icons/render.mjs         # SVG → build/master/*.png（1024，透明底）
python3 design/app-icons/build_assets.py # 导出 7 档 PNG、fnOS 图标、接触表
node design/app-icons/screenshot.mjs     # gallery.html → preview/gallery.png
```

依赖 Node + Playwright Chromium（渲染 SVG）、Python 3 + Pillow（缩放与拼版）。

## 文件结构

```
design/app-icons/
├── gallery.html            # 可交互浏览页（深浅色切换、各尺寸预览）
├── README.md
├── concepts/               # SVG 真源：00 对照 + 01–06 六套方案
├── exports/<方案>/
│   ├── icon-16.png … icon-1024.png
│   └── fnos/               # fnOS 包内图标：ICON.PNG、icon_256.png、icon_64.png
├── preview/                # contact-sheet.png、gallery.png
├── build/master/           # 1024 透明主稿（中间产物，可重建）
├── render.mjs              # SVG → 主稿
├── build_assets.py         # 缩放、fnOS 图标、接触表
└── screenshot.mjs          # 浏览页截图
```

## 边界

- 未在真实 fnOS 应用中心验证最终显示尺寸、圆角遮罩与缓存刷新；接入后需要重装或升级实测。
- 16px 简化稿没有单独出图，06 在 16px 的箭头头部会丢失细节，选中后建议补一版。
- 图标为原创矢量；`00-current.svg` 仅用于对照，未删除或覆盖现有线上资源。
