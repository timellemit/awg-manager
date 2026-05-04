<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import type { PolicyDevice } from '$lib/types';
	import { singboxWizard } from '$lib/stores/singboxWizard';

	const wizardState = singboxWizard.state;

	let allDevices = $state<PolicyDevice[]>([]);
	let loading = $state(true);

	onMount(async () => {
		try {
			allDevices = await api.listPolicyDevices();
		} catch {
			allDevices = [];
		}
		loading = false;
	});

	const policyName = $derived($wizardState.policyName);
	const eligible = $derived(
		allDevices.filter((d) => d.policy === '' || d.policy === policyName),
	);
	const hidden = $derived(allDevices.length - eligible.length);
	const selectedMacs = $derived($wizardState.deviceMacs);
	const allSelected = $derived(
		eligible.length > 0 && eligible.every((d) => selectedMacs.includes(d.mac)),
	);

	function toggleAll(): void {
		if (allSelected) {
			singboxWizard.setDeviceMacs([]);
		} else {
			singboxWizard.setDeviceMacs(eligible.map((d) => d.mac));
		}
	}

	function toggleDevice(mac: string): void {
		const next = selectedMacs.includes(mac)
			? selectedMacs.filter((m) => m !== mac)
			: [...selectedMacs, mac];
		singboxWizard.setDeviceMacs(next);
	}

	$effect(() => {
		if (!loading && eligible.length > 0 && selectedMacs.length === 0) {
			singboxWizard.setDeviceMacs(eligible.map((d) => d.mac));
		}
	});
</script>

<div class="title">Какие устройства пустить через мастер?</div>
<div class="subtitle">
	Создаётся access policy <b>{policyName}</b>. Показаны только устройства, не привязанные к другой policy.
</div>

{#if loading}
	<div class="hint">Загрузка устройств...</div>
{:else if eligible.length === 0}
	<div class="empty">
		Все устройства уже привязаны к другим policy. Откройте управление policy и освободите хотя бы одно устройство, либо назначьте им {policyName} вручную.
		<a class="link" href="/routing?tab=policy">Открыть policy management</a>
	</div>
{:else}
	<div class="list">
		<button type="button" class="row toggle-all" onclick={toggleAll}>
			<span class="cb" class:checked={allSelected}></span>
			<div class="name">Все устройства</div>
			<div class="meta mono">{eligible.length}</div>
		</button>
		{#each eligible as d (d.mac)}
			<button type="button" class="row" onclick={() => toggleDevice(d.mac)}>
				<span class="cb" class:checked={selectedMacs.includes(d.mac)}></span>
				<div class="name">{d.name || d.hostname || d.mac}</div>
				<div class="meta mono">{d.ip} {d.mac}</div>
			</button>
		{/each}
	</div>
	{#if hidden > 0}
		<div class="hint">{hidden} устройств уже привязаны к другим policy и в список не попали.</div>
	{/if}
{/if}

<style>
	.title { font-size: 1.05rem; color: var(--color-text-primary); font-weight: 600; margin-bottom: 0.3rem; }
	.subtitle { color: var(--color-text-muted); font-size: 0.85rem; margin-bottom: 1rem; }
	.hint { color: var(--color-text-muted); font-size: 0.78rem; margin-top: 0.6rem; }
	.empty {
		padding: 1.2rem;
		border: 1px dashed var(--color-border);
		border-radius: 6px;
		color: var(--color-text-muted);
		font-size: 0.85rem;
		text-align: center;
	}
	.link {
		display: inline-block;
		margin-top: 0.6rem;
		color: var(--color-accent);
	}
	.list {
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: 6px;
		overflow: hidden;
	}
	.row {
		display: flex;
		align-items: center;
		gap: 0.7rem;
		padding: 0.55rem 0.75rem;
		border: 0;
		border-bottom: 1px solid var(--color-border);
		background: transparent;
		width: 100%;
		font: inherit;
		text-align: left;
		color: var(--color-text-primary);
		cursor: pointer;
	}
	.row:last-child { border-bottom: 0; }
	.row.toggle-all { background: var(--color-bg-primary); font-weight: 600; }
	.cb {
		width: 14px; height: 14px;
		border: 1px solid var(--color-border);
		border-radius: 3px;
		background: var(--color-bg-primary);
		flex-shrink: 0;
	}
	.cb.checked { background: var(--color-accent); border-color: var(--color-accent); }
	.name { flex: 1; }
	.meta { font-size: 0.75rem; color: var(--color-text-muted); }
	.mono { font-family: var(--font-mono, ui-monospace, monospace); }
</style>
