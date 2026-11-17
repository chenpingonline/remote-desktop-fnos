export const resolutionOptions = [
  { value: "auto", label: "自动跟随窗口" },
  ...[[1280, 720], [1600, 900], [1920, 1080], [1920, 1200], [2560, 1440], [3840, 2160]].map(([width, height]) => ({ value: `${width}x${height}`, label: `${width} × ${height}` })),
];
export function fixedResolution(mode: string, dpi = 96) {
  if (mode === "auto" || !resolutionOptions.some(option => option.value === mode)) return undefined;
  const [width, height] = mode.split("x").map(Number);
  return { width, height, dpi };
}
export function loadResolutionMode(protocol: string) {
  if (protocol !== "rdp") return "auto";
  try {
    const mode = localStorage.getItem(`remote-desktop.resolution.${protocol}`) || "auto";
    return resolutionOptions.some(option => option.value === mode) ? mode : "auto";
  } catch { return "auto"; }
}
export function saveResolutionMode(protocol: string, mode: string) {
  if (!resolutionOptions.some(option => option.value === mode)) return;
  try { localStorage.setItem(`remote-desktop.resolution.${protocol}`, mode); } catch { /* Current session remains usable. */ }
}
// RDP supports desktop scaling from 100% to 200% in this application.
export function pixelRatio() {
  return typeof window === "undefined" ? 1 : Math.max(1, window.devicePixelRatio || 1);
}
export function nativeDisplayScale(zoom: number, ratio = pixelRatio()) {
  return zoom / Math.max(1, ratio);
}
export function displayTransform(width: number, height: number, remoteWidth: number, remoteHeight: number, mode: string, ratio = pixelRatio()) {
  const x = width / remoteWidth, y = height / remoteHeight;
  const scale = mode === "fit" ? Math.min(x, y) : nativeDisplayScale(Number(mode), ratio);
  return mode === "fill" ? { x, y } : { x: scale, y: scale };
}
export const displayScales = [0.5, 0.75, 1, 1.25, 1.5, 1.75, 2];
export function loadDisplayScale(protocol: string) {
  if (protocol !== "rdp") return 1;
  try {
    const value = Number(localStorage.getItem(`remote-desktop.display-scale.${protocol}`));
    return displayScales.includes(value) ? value : 1;
  } catch { return 1; }
}
export function saveDisplayScale(protocol: string, value: number) {
  if (!displayScales.includes(value)) return;
  try { localStorage.setItem(`remote-desktop.display-scale.${protocol}`, String(value)); } catch { /* Session setting still works. */ }
}
export function displaySize(width: number, height: number, scale = 1, ratio = pixelRatio()) {
  const selected = displayScales.includes(scale) ? scale : 1;
  const density = Math.min(2, Math.max(1, Number.isFinite(ratio) ? ratio : 1));
  // Preserve useful content size if the maximum framebuffer or 200% DPI is reached.
  const factor = Math.min(Math.max(density, 1 / selected), 2 / selected, 4096 / Math.max(width, 1), 2160 / Math.max(height, 1));
  return {
    width: Math.max(320, Math.round(width * factor)),
    height: Math.max(200, Math.round(height * factor)),
    dpi: Math.max(96, Math.min(192, Math.round(96 * factor * selected))),
  };
}
export function sessionDisplay(width: number, height: number, scale = 1, mode = "auto", ratio = pixelRatio()) {
  const automatic = displaySize(width, height, scale, ratio);
  return fixedResolution(mode, automatic.dpi) || automatic;
}

// Negotiate against the real workspace from the first connection, including
// the tab row which is created when the first remote session opens.
export function initialSessionDisplay(workspace: HTMLElement | null, scale = 1, mode = "auto") {
  let tabHeight = workspace?.querySelector<HTMLElement>(".tabbar")?.offsetHeight || 0;
  if (workspace && !tabHeight) {
    const measurement = document.createElement("div");
    measurement.className = "tabbar";
    measurement.style.cssText = "position:absolute;visibility:hidden;pointer-events:none";
    const button = document.createElement("button");
    button.textContent = "管理";
    measurement.appendChild(button);
    workspace.appendChild(measurement);
    tabHeight = measurement.offsetHeight;
    measurement.remove();
  }
  return sessionDisplay(workspace?.clientWidth || window.innerWidth,
    Math.max(200, (workspace?.clientHeight || window.innerHeight) - tabHeight), scale, mode);
}

export const renderModeOptions = [
  { value: "balanced", label: "均衡 · 推荐" },
  { value: "smooth", label: "流畅优先" },
  { value: "clear", label: "清晰优先" },
];
export type RenderMode = "balanced" | "smooth" | "clear";
export function loadRenderMode(protocol: string): RenderMode {
  try {
    const value = localStorage.getItem(`remote-desktop.render-mode.${protocol}`);
    return value === "clear" || value === "smooth" ? value : "balanced";
  } catch { return "balanced"; }
}
export function saveRenderMode(protocol: string, mode: string) {
  if (!renderModeOptions.some(option => option.value === mode)) return;
  try { localStorage.setItem(`remote-desktop.render-mode.${protocol}`, mode); } catch { /* Current session remains usable. */ }
}
// Only automatic RDP resolution is capped. Explicit resolutions remain authoritative.
export function performanceDisplay(width: number, height: number, scale = 1, mode = "auto", profile = "balanced", ratio = pixelRatio()) {
  const size = sessionDisplay(width, height, scale, mode, ratio);
  if (profile !== "smooth" || mode !== "auto") return size;
  const factor = Math.min(1, 1920 / size.width, 1080 / size.height);
  return {
    width: Math.max(320, Math.round(size.width * factor)),
    height: Math.max(200, Math.round(size.height * factor)),
    dpi: Math.max(96, Math.round(size.dpi * factor)),
  };
}

// RDP desktop wallpaper is enabled unless explicitly disabled on this browser.
export function loadWallpaper(): boolean {
  try { return localStorage.getItem("remote-desktop.rdp-wallpaper") !== "false"; }
  catch { return true; }
}
export function saveWallpaper(enabled: boolean) {
  try { localStorage.setItem("remote-desktop.rdp-wallpaper", String(enabled)); } catch {}
}
