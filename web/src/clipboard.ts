export function sendClipboard(
  stream: any,
  text: string,
  onError: (s: string) => void,
  onComplete: () => void = () => {},
) {
  const bytes = new TextEncoder().encode(text);
  // Guacamole 1.6.0 RDP/VNC clipboard handlers do not acknowledge stream
  // creation or blobs. Send the complete bounded stream without waiting for ACKs.
  stream.onack = (status: any) => {
    if (status.isError()) onError("远端拒绝剪贴板文本");
  };
  for (let offset = 0; offset < bytes.length; offset += 4096) {
    const chunk = bytes.subarray(offset, offset + 4096);
    stream.sendBlob(btoa(String.fromCharCode(...chunk)));
  }
  stream.sendEnd();
  onComplete();
}
export function receiveClipboard(stream: any, onText: (s: string) => void) {
  const decoder = new TextDecoder("utf-8", { fatal: true });
  let text = "",
    size = 0,
    failed = false;
  stream.onblob = (blob: string) => {
    if (failed) return;
    try {
      const bytes = Uint8Array.from(atob(blob), (c) => c.charCodeAt(0));
      size += bytes.length;
      if (size > 1 << 20) throw new Error("limit");
      text += decoder.decode(bytes, { stream: true });
      stream.sendAck("OK", 0);
    } catch {
      failed = true;
      stream.sendAck("文本编码不合法或超出限制", 256);
    }
  };
  stream.onend = () => {
    if (!failed) {
      try {
        text += decoder.decode();
        onText(text);
      } catch {}
    }
  };
}
