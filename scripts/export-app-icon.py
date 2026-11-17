#!/usr/bin/env python3
"""Export the user-selected raster master without altering its artwork."""
from pathlib import Path
from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
SELECTED = ROOT / "web/design/app-icons/selected/dual-screen-lime"
PACK = ROOT / "packaging/remote-desktop"


def main():
    master = Image.open(SELECTED / "master.png").convert("RGBA")
    if master.width != master.height:
        raise ValueError("The selected icon master must be square")
    exports = SELECTED / "exports"
    exports.mkdir(parents=True, exist_ok=True)
    for size in (16, 32, 64, 128, 256, 512, 1024):
        master.resize((size, size), Image.Resampling.LANCZOS).save(
            exports / f"icon-{size}.png"
        )
    targets = {
        PACK / "ICON.PNG": 64,
        PACK / "ICON_256.PNG": 256,
        PACK / "app/ui/images/icon_64.png": 64,
        PACK / "app/ui/images/icon_256.png": 256,
        ROOT / "web/public/favicon.png": 32,
        ROOT / "web/public/apple-touch-icon.png": 180,
    }
    for path, size in targets.items():
        path.parent.mkdir(parents=True, exist_ok=True)
        master.resize((size, size), Image.Resampling.LANCZOS).save(path)
        print(f"{path.relative_to(ROOT)} ({size}×{size})")


if __name__ == "__main__":
    main()
