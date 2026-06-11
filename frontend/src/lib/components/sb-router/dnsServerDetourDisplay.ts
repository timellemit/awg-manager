import type {
	SingboxProxyGroup,
	SingboxRouterDNSServer,
	SingboxRouterOutbound,
	SingboxTunnel,
	Subscription,
} from '$lib/types';
import type { OutboundGroup } from '$lib/components/routing/singboxRouter/outboundOptions';
import { resolveOutboundDisplay } from './adapters';
import type { OutboundDisplay } from './types';

/** Явный direct outbound в detour DNS-сервера. */
export function isDnsServerDirectDetour(detour?: string): boolean {
	return detour?.trim() === 'direct';
}

/** Detour не задан — sing-box маршрутизирует DNS по таблице route (дефолт в UI). */
export function isDnsServerViaRouteDetour(detour?: string): boolean {
	return !detour?.trim();
}

/**
 * DNS server detour chip:
 * - direct → «Напрямую»
 * - пустой detour → «через route»
 * - конкретный outbound (подписка, туннель, composite…) → обычный мелкий бейдж цели
 */
export function dnsServerDetourDisplay(
	server: SingboxRouterDNSServer,
	outbounds: SingboxRouterOutbound[],
	outboundOptions: OutboundGroup[] = [],
	subscriptions: Subscription[] | null = null,
	proxyGroups: SingboxProxyGroup[] = [],
	singboxTunnels: SingboxTunnel[] = [],
): OutboundDisplay {
	const detour = server.detour?.trim() ?? '';

	if (detour === 'direct') {
		return resolveOutboundDisplay(
			'direct',
			'direct',
			outbounds,
			outboundOptions,
			subscriptions,
			proxyGroups,
			singboxTunnels,
		);
	}

	if (!detour) {
		return {
			name: 'via-route',
			label: 'через route',
			kind: 'via-route',
			tone: 'via-route',
		};
	}

	return resolveOutboundDisplay(
		detour,
		'route',
		outbounds,
		outboundOptions,
		subscriptions,
		proxyGroups,
		singboxTunnels,
	);
}
