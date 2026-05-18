<script lang="ts">
	import type { SingboxDelayState } from '$lib/utils/singboxDelay';

	type ConnState = 'idle' | 'connected' | 'disconnected' | 'checking';

	interface Props {
		/** Singbox delay check: explicit label (skips AWG derivation). */
		label?: string;
		/** Singbox latency tier for text colour. */
		state?: SingboxDelayState;
		/** AWG connectivity check — used when `label` is omitted. */
		connectivity?: ConnState;
		latencyMs?: number | null;
		recovering?: boolean;
		checking?: boolean;
		disabled?: boolean;
		/** `sm` dense AWG · `mid` subscription members · `md` default cards/list */
		size?: 'sm' | 'mid' | 'md';
		/** Colored tier border (subscription member tunnel cards). */
		forceBorder?: boolean;
		title?: string;
		onclick?: (e: MouseEvent) => void;
	}

	let {
		label: labelProp,
		state,
		connectivity,
		latencyMs = null,
		recovering = false,
		checking = false,
		disabled = false,
		size = 'md',
		forceBorder = false,
		title: titleProp,
		onclick,
	}: Props = $props();

	function awgTier(ms: number): string {
		if (ms < 80) return 'good';
		if (ms < 130) return 'warn';
		if (ms < 200) return 'high';
		return 'bad';
	}

	const isSingbox = $derived(labelProp !== undefined);

	let label = $derived.by(() => {
		if (labelProp !== undefined) {
			return checking ? '...' : labelProp;
		}
		if (checking || connectivity === 'checking') return '...';
		if (connectivity === 'connected' && latencyMs !== null) return `${latencyMs}ms`;
		if (connectivity === 'connected') return 'OK';
		if (connectivity === 'disconnected') return '—';
		return '...';
	});

	let tierClass = $derived.by(() => {
		if (isSingbox && state) return `tier-${state}`;
		if (recovering) return '';
		if (checking || connectivity === 'checking') return '';
		if (connectivity === 'disconnected') return 'tier-bad';
		if (connectivity === 'connected' && latencyMs !== null) return `tier-${awgTier(latencyMs)}`;
		return '';
	});

	let title = $derived(
		titleProp ??
			(isSingbox
				? 'Обновить delay'
				: connectivity === 'disconnected'
					? 'Нет связи. Нажать для проверки'
					: 'Проверить связь'),
	);

	let isSpinning = $derived(checking || (!isSingbox && connectivity === 'checking'));
	let isDisabled = $derived(disabled || isSpinning);
	/** Icon only for numeric latency or while an explicit recheck is in flight. */
	let showRefreshIcon = $derived(isSpinning || /^\d+ms$/.test(label));
</script>

<button
	type="button"
	class="ping-btn {size} {tierClass}"
	class:force-border={forceBorder}
	class:spinning={isSpinning}
	{title}
	{onclick}
	disabled={isDisabled}
>
	{label}
	{#if showRefreshIcon}
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
			<path d="M23 4v6h-6M1 20v-6h6" />
			<path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
		</svg>
	{/if}
</button>

<style>
	.ping-btn {
		background: none;
		border: 1px solid transparent;
		color: var(--color-text-muted);
		font-family: var(--font-mono, monospace);
		font-size: 12px;
		font-weight: 500;
		padding: 2px 6px;
		border-radius: var(--radius-sm, 4px);
		cursor: pointer;
		display: inline-flex;
		width: auto;
		max-width: 100%;
		align-items: center;
		gap: 4px;
		flex: 0 0 auto;
		font-variant-numeric: tabular-nums;
		transition: background 0.15s ease, border-color 0.15s ease, color 0.4s ease;
		white-space: nowrap;
	}

	.ping-btn:hover:not(:disabled) {
		background: var(--color-bg-hover, rgba(255, 255, 255, 0.05));
	}

	.ping-btn:not(.force-border):hover:not(:disabled) {
		border-color: var(--color-border);
	}

	.ping-btn:disabled {
		cursor: default;
		opacity: 0.55;
	}

	.ping-btn.mid {
		font-size: 10.5px;
		line-height: 1.25;
		padding: 1px 5px;
		gap: 3px;
		border-radius: 3px;
	}

	.ping-btn.sm {
		font-size: 9px;
		line-height: 1.2;
		padding: 1px 4px;
		gap: 2px;
		border-radius: 3px;
	}

	.ping-btn.force-border {
		border-color: var(--color-muted-border);
	}

	/* AWG connectivity tiers */
	.ping-btn.tier-good {
		color: var(--color-success);
	}
	.ping-btn.force-border.tier-good {
		border-color: var(--color-success-border);
	}
	.ping-btn.tier-warn {
		color: var(--color-warning);
	}
	.ping-btn.force-border.tier-warn {
		border-color: var(--color-warning-border);
	}
	.ping-btn.tier-high {
		color: var(--color-broken);
	}
	.ping-btn.force-border.tier-high {
		border-color: var(--color-broken-border);
	}
	.ping-btn.tier-bad {
		color: var(--color-error);
	}
	.ping-btn.force-border.tier-bad {
		border-color: var(--color-error-border);
	}

	/* Singbox delay tiers */
	.ping-btn.tier-ok {
		color: var(--latency-color-ok);
	}
	.ping-btn.force-border.tier-ok {
		border-color: var(--color-success-border);
	}
	.ping-btn.tier-slow {
		color: var(--latency-color-slow);
	}
	.ping-btn.force-border.tier-slow {
		border-color: var(--color-warning-border);
	}
	.ping-btn.tier-fail {
		color: var(--latency-color-fail);
	}
	.ping-btn.force-border.tier-fail {
		border-color: var(--color-error-border);
	}
	.ping-btn.tier-unknown,
	.ping-btn.tier-stopped {
		color: var(--color-text-muted);
	}
	.ping-btn.force-border.tier-unknown,
	.ping-btn.force-border.tier-stopped {
		border-color: var(--color-muted-border);
	}

	.ping-btn svg {
		flex-shrink: 0;
		width: 11px;
		height: 11px;
		opacity: 0.45;
		transition: opacity 0.15s ease, transform 0.3s;
	}

	.ping-btn.mid svg {
		width: 10px;
		height: 10px;
	}

	.ping-btn.sm svg {
		width: 9px;
		height: 9px;
	}

	.ping-btn:hover:not(:disabled) svg {
		opacity: 1;
	}

	.ping-btn.spinning svg {
		opacity: 0.7;
		animation: ping-spin 0.9s linear infinite;
	}

	@keyframes ping-spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
