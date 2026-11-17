#!/usr/bin/env python3
"""生成图标导出尺寸、小尺寸预览与总览接触表。

依赖 design/app-icons/build/master/*.png（由 render.mjs 以 1024×1024 透明底渲染）。
用法：python3 design/app-icons/build_assets.py
"""
from __future__ import annotations

from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parent
MASTER_DIR = ROOT / "build" / "master"
EXPORT_DIR = ROOT / "exports"
PREVIEW_DIR = ROOT / "preview"

BG = (10, 14, 11)
SURFACE = (18, 23, 19)
INK = (238, 241, 233)
MUTED = (162, 171, 160)
FAINT = (122, 132, 121)
LINE = (38, 48, 40)
ACCENT = (215, 251, 98)
LIGHT_BG = (244, 245, 242)
LIGHT_INK = (23, 25, 28)
LIGHT_MUTED = (95, 103, 110)

SIZES = [1024, 512, 256, 128, 64, 32, 16]

# id -> (中文名, 英文名, 一句话定位, 推荐标记)
META = {
    "01-pointer": ("指针屏", "Pointer", "屏幕 + 光标，接管远端", "主推"),
    "02-signal": ("信号屏", "Signal", "屏幕 + 信号弧，远程串流", "备选"),
    "03-swap": ("双向屏", "Swap", "屏幕 + 双向箭头，双向会话", ""),
    "04-dual-screen": ("双屏", "Dual", "双屏 + 光标，控制另一端", ""),
    "05-monogram-r": ("R 标", "Monogram", "Relay 首字母，品牌向", "品牌向"),
    "06-remote-arrow": ("外联", "Remote", "屏幕 + 外箭头，经典语义", "稳健"),
}
ORDER = list(META.keys())


def font(size: int, weight: str = "W3") -> ImageFont.FreeTypeFont:
    path = "/System/Library/Fonts/Hiragino Sans GB.ttc"
    index = 0 if weight == "W3" else 1
    return ImageFont.truetype(path, size, index=index)


def rounded_tile(im: Image.Image, mask_size: int, radius: int) -> Image.Image:
    """把方形 RGBA 图标按圆角裁切，用于展示在预览底板上。"""
    mask = Image.new("L", (mask_size, mask_size), 0)
    d = ImageDraw.Draw(mask)
    d.rounded_rectangle((0, 0, mask_size - 1, mask_size - 1), radius=radius, fill=255)
    out = im.copy()
    out.putalpha(mask)
    return out


def load(master: Path, size: int) -> Image.Image:
    im = Image.open(master).convert("RGBA")
    return im.resize((size, size), Image.LANCZOS)


def build_exports() -> None:
    for cid in ORDER:
        master = MASTER_DIR / f"{cid}.png"
        out_dir = EXPORT_DIR / cid
        fnos_dir = out_dir / "fnos"
        out_dir.mkdir(parents=True, exist_ok=True)
        fnos_dir.mkdir(parents=True, exist_ok=True)
        for size in SIZES:
            load(master, size).save(out_dir / f"icon-{size}.png")
        for size, name in ((256, "icon_256.png"), (64, "icon_64.png")):
            load(master, size).save(fnos_dir / name)
        load(master, 64).save(fnos_dir / "ICON.PNG")
        print("exported", cid)


def draw_centered(draw: ImageDraw.ImageDraw, cx: int, y: int, text: str, f, fill) -> None:
    w = draw.textlength(text, font=f)
    draw.text((cx - w / 2, y), text, font=f, fill=fill)


def contact_sheet() -> None:
    S = 2  # 2x 超采样
    W, H = 1960, 1060
    canvas = Image.new("RGB", (W * S, H * S), BG)
    d = ImageDraw.Draw(canvas)
    title = font(38 * S, "W6")
    sub = font(17 * S, "W3")
    section = font(15 * S, "W6")
    name_f = font(23 * S, "W6")
    tag_f = font(13 * S, "W3")
    tiny = font(10 * S, "W3")
    badge_f = font(11 * S, "W6")

    def rr(box, radius, fill, outline=None, width=0):
        d.rounded_rectangle(tuple(v * S for v in box), radius=radius * S, fill=fill,
                            outline=outline, width=width * S)

    def text(x, y, s, f, fill, anchor=None):
        d.text((x * S, y * S), s, font=f, fill=fill, anchor=anchor)

    # 页头
    text(80, 52, "Relay 远程桌面 · 应用图标方案", title, INK)
    text(80, 108, "6 个方向 · 512 网格 · 圆角 116 · 品牌青柠绿 #D7FB62 / 墨绿 #182006",
         sub, MUTED)
    d.line((80 * S, 154 * S, 1880 * S, 154 * S), fill=LINE, width=S)

    cols = 6
    col_w = (1800 - 5 * 24) / cols
    centers = [80 + col_w * i + col_w / 2 for i in range(cols)]

    # 大图
    text(80, 176, "方案总览", section, ACCENT)
    icon_size = 232
    icon_y = 206
    for i, cid in enumerate(ORDER):
        cx = centers[i]
        name, en, tag, rec = META[cid]
        icon = load(MASTER_DIR / f"{cid}.png", icon_size * S)
        canvas.paste(icon, (int((cx - icon_size / 2) * S), icon_y * S), icon)
        if rec:
            tw = d.textlength(rec, font=badge_f)
            bx = cx - (tw + 20) / 2
            by = icon_y - 27
            rr((bx, by, bx + tw + 20, by + 21), 10.5, ACCENT)
            text(bx + 10, by + 3.5, rec, badge_f, (24, 32, 6))
        text(cx, icon_y + icon_size + 16, name, name_f, INK, anchor="ma")
        text(cx, icon_y + icon_size + 48, en.upper(), tiny, FAINT, anchor="ma")
        text(cx, icon_y + icon_size + 70, tag, tag_f, MUTED, anchor="ma")

    # 小尺寸
    small_label_y = 592
    text(80, small_label_y, "小尺寸可读性（按实际像素预览：64 / 32 / 16）", section, ACCENT)
    small_y = 626
    for i, cid in enumerate(ORDER):
        cx = centers[i]
        total = 64 + 20 + 32 + 20 + 16
        x = cx - total / 2
        pairs = ((64, x), (32, x + 84), (16, x + 136))
        for size, px in pairs:
            icon = load(MASTER_DIR / f"{cid}.png", size * S)
            canvas.paste(icon, (int(px * S), small_y * S), icon)
        for label, px in (("64", x), ("32", x + 84), ("16", x + 136)):
            text(px, small_y + 72, label, tiny, FAINT)

    # 浅色界面
    band_y = 742
    rr((80, band_y, 1880, band_y + 220), 24, LIGHT_BG)
    text(116, band_y + 24, "浅色界面（fnOS 应用中心 / 浏览器标签页）", section, LIGHT_INK)
    text(116, band_y + 50, "深色底板在浅色系统里仍有清晰边界，不会和白色背景粘在一起。", tiny, LIGHT_MUTED)
    for i, cid in enumerate(ORDER):
        cx = centers[i]
        icon = load(MASTER_DIR / f"{cid}.png", 96 * S)
        canvas.paste(icon, (int((cx - 48) * S), (band_y + 70) * S), icon)
        text(cx, band_y + 70 + 104, META[cid][0], tiny, LIGHT_MUTED, anchor="ma")

    text(80, 1000, "1024 px 主稿按 Lanczos 下采样；SVG 源文件在 concepts/，可直接改色或改尺寸。",
         tiny, FAINT)
    text(1880, 1000, "主推 01 指针屏 · 稳健 06 外联 · 品牌向 05 R 标", tiny, FAINT, anchor="ra")

    canvas = canvas.resize((W, H), Image.LANCZOS)
    canvas.save(PREVIEW_DIR / "contact-sheet.png")
    print("wrote contact sheet")


if __name__ == "__main__":
    build_exports()
    contact_sheet()
