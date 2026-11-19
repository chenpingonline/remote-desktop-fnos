// Shared by the desktop tabs of one workspace. Only the authenticated owner's
// gateway can resolve a token. The bytes never enter the browser clipboard.
function newOffer(source: string, detecting: boolean) {
  let finish!: (token?: string) => void;
  const ready = new Promise<string | undefined>((resolve) => { finish = resolve; });
  return { source, detecting, ready, finish };
}
export const fileClipboardMime = "application/x-fnos-rdp-files";
let current: ReturnType<typeof newOffer> | undefined;
let timeout: ReturnType<typeof setTimeout> | undefined;

export function clearFileOffer(source?: string) {
  if (source && current?.source !== source) return;
  clearTimeout(timeout);
  current?.finish();
  current = undefined;
}
export function beginFileOffer(source: string, detecting = false) {
  // Windows reports the clipboard format asynchronously after Ctrl+C. Keep
  // an immediately following paste attached to the same copy operation.
  if (!detecting && current?.source === source && current.detecting) {
    current.detecting = false;
  } else {
    clearFileOffer();
    current = newOffer(source, detecting);
  }
  clearTimeout(timeout);
  timeout = setTimeout(clearFileOffer, detecting ? 1500 : 300000);
}
export function finishFileOffer(source: string, token: string) {
  if (current?.source !== source) return;
  clearTimeout(timeout);
  current.finish(token);
}
export function fileOffer() { return current; }
