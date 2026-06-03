/**
 * Единый WS-поток Clash connections для шапки и FlowGraph.
 */
import { derived, writable } from 'svelte/store';
import { singboxRouter } from '$lib/stores/singboxRouter';
import type { ClashConnectionsRaw, ConnectionsSnapshot } from '$lib/types/singboxConnections';
import { parseSnapshot } from '$lib/utils/singboxConnections';
import { createClashWS, type WSStatus } from '$lib/utils/clashWebSocket';
import { formatBytes } from '$lib/utils/format';

const EMPTY: ConnectionsSnapshot = {
	connections: [],
	downloadTotal: 0,
	uploadTotal: 0,
	connectionsTotal: 0,
};
const EMPTY_CLIENTS = new Map<string, string>();

const snapshot = writable<ConnectionsSnapshot>(EMPTY);
const wsStatus = writable<WSStatus>('connecting');

let wsClose: (() => void) | null = null;
let bound = false;

function connect(): void {
	if (wsClose) return;
	wsStatus.set('connecting');
	wsClose = createClashWS<ClashConnectionsRaw>(
		'/api/singbox/clash/connections',
		(raw) => snapshot.set(parseSnapshot(raw, EMPTY_CLIENTS)),
		(s) => wsStatus.set(s),
	);
}

function disconnect(): void {
	wsClose?.();
	wsClose = null;
	snapshot.set(EMPTY);
	wsStatus.set('connecting');
}

/** Подписывает store на enabled-состояние движка (идемпотентно). */
export function bindLiveConnectionsStore(): void {
	if (bound) return;
	bound = true;
	singboxRouter.status.subscribe((s) => {
		if (s?.enabled) connect();
		else disconnect();
	});
}

export const liveConnectionsSnapshot = { subscribe: snapshot.subscribe };
export const liveConnectionsWsStatus = { subscribe: wsStatus.subscribe };

export const liveConnectionsTraffic = derived(
	[snapshot, wsStatus],
	([snap, status]) => {
		if (status !== 'open') return null;
		if (snap.connectionsTotal === 0) return null;
		const up = snap.connections.reduce((n, c) => n + c.upload, 0);
		const down = snap.connections.reduce((n, c) => n + c.download, 0);
		return `↑ ${formatBytes(up)} ↓ ${formatBytes(down)}`;
	},
);
