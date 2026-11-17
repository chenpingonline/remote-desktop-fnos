import test from 'node:test';
import assert from 'node:assert/strict';
import { connectionLaunchLabel, localSessionFor, sessionNeedsReconnect } from '../src/local-sessions.ts';

test('returns only a session owned by this page for the same connection', () => {
  const sessions = [{ id: 'live', connectionId: 'pc', status: '已连接' }];
  assert.equal(localSessionFor(sessions, 'pc')?.id, 'live');
  assert.equal(connectionLaunchLabel(sessions, 'pc', 'offline'), '返回会话');
  assert.equal(localSessionFor(sessions, 'other'), undefined);
  // A backend busy flag may describe another browser: never claim it is reusable.
  assert.equal(connectionLaunchLabel([], 'pc', 'busy'), '连接');
  assert.equal(connectionLaunchLabel(sessions, 'other', 'offline'), '连接');
});

test('connecting and unstable sessions are returned to without starting duplicates', () => {
  for (const status of ['等待连接', '连接中', '已连接', '连接不稳定']) {
    assert.equal(connectionLaunchLabel([{ id: 'pending', connectionId: 'pc', status }], 'pc', 'online'), '返回会话');
    assert.equal(sessionNeedsReconnect(status), false);
  }
});

test('failed sessions reconnect and an existing usable duplicate wins', () => {
  for (const status of ['已断开', '连接失败', '已中断', '正在断开']) {
    const sessions = [{ id: 'failed', connectionId: 'pc', status }];
    assert.equal(connectionLaunchLabel(sessions, 'pc', 'busy'), '重新连接');
    sessions.push({ id: 'live', connectionId: 'pc', status: '已连接' });
    assert.equal(localSessionFor(sessions, 'pc')?.id, 'live');
    assert.equal(connectionLaunchLabel(sessions, 'pc', 'busy'), '返回会话');
  }
});
