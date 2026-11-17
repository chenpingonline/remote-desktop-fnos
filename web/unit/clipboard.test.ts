import { test } from 'node:test';
import assert from 'node:assert/strict';
import { sendClipboard } from '../src/clipboard.ts';

for (const text of ['', '中文😀\nsecond line', '字😀'.repeat(3000)]) {
  test(`clipboard completes without server ACK (${text.length} characters)`, () => {
    const chunks: Uint8Array[] = [];
    let ended = false, completed = false;
    const stream = {
      onack: undefined,
      sendBlob(blob: string) {
        assert.equal(ended, false);
        const bytes = Uint8Array.from(atob(blob), c => c.charCodeAt(0));
        assert.ok(bytes.length <= 4096);
        chunks.push(bytes);
      },
      sendEnd() { ended = true; },
    };
    sendClipboard(stream, text, message => assert.fail(message), () => {
      assert.equal(ended, true);
      completed = true;
    });
    assert.equal(completed, true);
    assert.equal(Buffer.concat(chunks).toString('utf8'), text);
  });
}
