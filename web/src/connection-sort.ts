import type { Connection } from './types';

export type ConnectionSort = 'recent' | 'name' | 'host' | 'reachable' | 'favorite';
export const connectionSortOptions: { value: ConnectionSort; label: string }[] = [
  { value: 'recent', label: '最近使用' },
  { value: 'name', label: '名称' },
  { value: 'host', label: '主机地址' },
  { value: 'reachable', label: '端口可达优先' },
  { value: 'favorite', label: '收藏优先' },
];
type SortableConnection = Pick<Connection, 'id' | 'name' | 'host' | 'lastConnected' | 'status' | 'favorite'>;
const collator = new Intl.Collator('zh-CN', { numeric: true, sensitivity: 'base' });
const lastUsed = (value: string | null) => value ? Date.parse(value) || 0 : 0;
const reachable = (status: string) => status === 'online' || status === 'busy';

export function compareConnections(a: SortableConnection, b: SortableConnection, sort: ConnectionSort): number {
  const byName = () => collator.compare(a.name, b.name);
  const byHost = () => collator.compare(a.host, b.host);
  const recent = () => lastUsed(b.lastConnected) - lastUsed(a.lastConnected);
  let result = 0;
  if (sort === 'name') result = byName();
  else if (sort === 'host') result = byHost();
  else if (sort === 'reachable') result = Number(reachable(b.status)) - Number(reachable(a.status));
  else if (sort === 'favorite') result = Number(b.favorite) - Number(a.favorite);
  return result || (sort === 'name' || sort === 'host' ? 0 : recent()) || byName() || byHost() || collator.compare(a.id, b.id);
}
