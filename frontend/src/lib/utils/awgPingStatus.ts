import type { TunnelListItem } from '$lib/types';

export type AwgPingStatusNote = { text: string; tone: 'recovering' | 'transitional' };
export type AwgPingLabelVariant = 'short' | 'full';

/** Label for the ping row during start / stop / ping-check recovery. */
export function awgPingStatusNote(
	tunnel: Pick<TunnelListItem, 'status' | 'pingCheck'>,
	variant: AwgPingLabelVariant = 'short',
): AwgPingStatusNote | null {
	switch (tunnel.status) {
		case 'starting':
			return { text: 'Запускается', tone: 'transitional' };
		case 'needs_stop':
			return { text: 'Остановка...', tone: 'transitional' };
	}

	if (tunnel.status === 'running' && tunnel.pingCheck.status === 'recovering') {
		const n = tunnel.pingCheck.restartCount;
		if (variant === 'full') {
			return {
				text: n > 0 ? `Восстановление (${n})` : 'Проверка связи...',
				tone: 'recovering',
			};
		}
		return {
			text: n > 0 ? `Восст. (${n})` : 'Восстановление...',
			tone: 'recovering',
		};
	}

	return null;
}

export function awgConnectivityCheckEnabled(
	tunnel: Pick<TunnelListItem, 'connectivityCheck'>,
): boolean {
	return (tunnel.connectivityCheck?.method ?? 'http') !== 'disabled';
}

/** Running tunnel with an enabled check that reported no path to the internet. */
export function awgConnectivityDown(
	tunnel: Pick<TunnelListItem, 'status' | 'connectivityCheck'>,
	conn: { connected: boolean; latency?: number | null } | undefined,
): boolean {
	if (tunnel.status !== 'running') return false;
	if (!awgConnectivityCheckEnabled(tunnel)) return false;
	return conn !== undefined && !conn.connected;
}

export function awgShowConnectivityRow(status: string): boolean {
	return (
		status === 'running' ||
		status === 'broken' ||
		status === 'starting' ||
		status === 'needs_stop'
	);
}

/** List layout: ping chip only when there is a numeric latency to show. */
export function awgListShowsPingButton(
	tunnel: Pick<TunnelListItem, 'status' | 'connectivityCheck' | 'pingCheck'>,
	connectivity: { connected: boolean; latency: number | null } | undefined,
): boolean {
	if (tunnel.status !== 'running') return false;
	if (!awgConnectivityCheckEnabled(tunnel)) return false;
	if (awgPingStatusNote(tunnel)) return false;
	return connectivity?.connected === true && connectivity.latency !== null;
}
