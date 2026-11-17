import assert from 'node:assert/strict';
import { test } from 'node:test';
import { compareConnections, type ConnectionSort } from '../src/connection-sort.ts';

const entries = [
  { id: 'a', name: '设备 10', host: '192.0.2.10', lastConnected: '2026-11-16T01:00:00Z', status: 'offline', favorite: false },
  { id: 'b', name: '设备 2', host: '192.0.2.2', lastConnected: '2026-11-15T01:00:00Z', status: 'online', favorite: true },
  { id: 'c', name: '设备 1', host: '192.0.2.1', lastConnected: null, status: 'unknown', favorite: false },
];
const order = (sort: ConnectionSort) => [...entries].sort((a, b) => compareConnections(a, b, sort)).map(c => c.id);

test('recent use puts never-connected devices last; names and hosts use natural numeric order', () => {
  assert.deepEqual(order('recent'), ['a', 'b', 'c']);
  assert.deepEqual(order('name'), ['c', 'b', 'a']);
  assert.deepEqual(order('host'), ['c', 'b', 'a']);
});
test('reachable and favorite priorities fall back to recent use', () => {
  assert.deepEqual(order('reachable'), ['b', 'a', 'c']);
  assert.deepEqual(order('favorite'), ['b', 'a', 'c']);
  assert.ok(compareConnections({ ...entries[0], status: 'busy' }, entries[2], 'reachable') < 0);
});
test('equal timestamps and invalid dates have deterministic ordering', () => {
  assert.ok(compareConnections({ ...entries[0], lastConnected: 'invalid' }, entries[2], 'recent') > 0);
  assert.equal(compareConnections(entries[0], entries[0], 'recent'), 0);
});
