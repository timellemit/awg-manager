<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { api } from '$lib/api/client';
	import type { PolicyDevice } from '$lib/types';
	import { singboxWizard } from '$lib/stores/singboxWizard';

	const wizardState = singboxWizard.state;

	let allDevices = $state<PolicyDevice[]>([]);
	let loading = $state(true);
	let snapshotTaken = $state(false);

	// In 'existing' mode, the chosen policy's NDMS internal name is known
	// immediately (user picked it on StepPolicy). In 'create' mode, the policy
	// doesn't exist until Phase 1 of apply — resolvedPolicyName stays null
	// during navigation, which correctly leaves StepDevices showing only
	// unassigned devices (nothing to pre-check yet).
	const policyName = $derived(
		$wizardState.policyMode === 'existing'
			? $wizardState.existingPolicyName
			: $wizardState.resolvedPolicyName,
	);
	const policyMode = $derived($wizardState.policyMode);
	const selectedMacs = $derived($wizardState.deviceMacs);

	// Eligible: unassigned (policy='') OR already in our resolved policy.
	// Devices in OTHER policies are hidden — wizard does not reassign cross-policy in v1.
	const visible = $derived(
		policyName
			? allDevices.filter((d) => d.policy === '' || d.policy === policyName)
			: allDevices.filter((d) => d.policy === ''),
	);

	onMount(async () => {
		try {
			allDevices = await api.listPolicyDevices();
		} catch {
			allDevices = [];
		}
		loading = false;
	});

	// Once data loaded AND resolved policy known: snapshot current membership;
	// pre-check those devices for 'existing' mode, leave empty for 'create' (UX 2).
	// Snapshot taken once via flag — re-runs guarded.
	$effect(() => {
		if (loading) return;
		if (snapshotTaken) return;
		if (policyMode === 'existing' && !policyName) return;
		snapshotTaken = true;
		const initial = policyName
			? allDevices.filter((d) => d.policy === policyName).map((d) => d.mac)
			: [];
		singboxWizard.setInitialDeviceMacs(initial);
		if (policyMode === 'existing') {
			singboxWizard.setDeviceMacs(initial);
		} else {
			singboxWizard.setDeviceMacs([]);
		}
	});

	function toggle(mac: string): void {
		const cur = untrack(() => $wizardState.deviceMacs);
		const next = cur.includes(mac) ? cur.filter((m) => m !== mac) : [...cur, mac];
		singboxWizard.setDeviceMacs(next);
	}
	function selectAll(): void {
		singboxWizard.setDeviceMacs(visible.map((d) => d.mac));
	}
	function selectNone(): void {
		singboxWizard.setDeviceMacs([]);
	}
	function isInPolicy(d: PolicyDevice): boolean {
		return policyName !== null && d.policy === policyName;
	}
</script>

<div class="title">Какие устройства пустить через мастер?</div>
<div class="hint">
	{#if $wizardState.policyMode === 'existing'}
		Используется access policy <b>{policyName ?? '...'}</b>. Показаны только устройства,
		не привязанные к другой policy.
	{:else}
		Создаётся access policy <b>{policyName ?? '...'}</b>. Показаны только устройства,
		не привязанные к другой policy.
	{/if}
</div>

{#if loading}
	<div class="muted">Загрузка устройств...</div>
{:else if visible.length === 0}
	<div class="muted">Нет доступных устройств. Освободите устройства из других policies в /routing → Политики доступа.</div>
{:else}
	<div class="bulk">
		<button type="button" class="link" onclick={selectAll}>Выбрать все</button>
		<button type="button" class="link" onclick={selectNone}>Снять все</button>
		<span class="counter">{selectedMacs.length} / {visible.length}</span>
	</div>
	<div class="device-list">
		{#each visible as d (d.mac)}
			{@const checked = selectedMacs.includes(d.mac)}
			{@const inPolicy = isInPolicy(d)}
			<label class="device" class:checked>
				<input type="checkbox" {checked} onchange={() => toggle(d.mac)} />
				<span class="device-name">{d.name || d.hostname || d.mac}</span>
				{#if inPolicy}
					<span class="badge-managed" title="Уже в этой policy">в policy</span>
				{/if}
				<span class="device-meta">{d.ip}</span>
				<span class="device-meta">{d.mac}</span>
			</label>
		{/each}
	</div>
{/if}

<style>
	.title { font-size: 1.05rem; color: var(--color-text-primary); font-weight: 600; margin-bottom: 0.6rem; }
	.hint { color: var(--color-text-muted); font-size: 0.85rem; margin-bottom: 1rem; }
	.muted { color: var(--color-text-muted); font-size: 0.85rem; }
	.bulk {
		display: flex;
		gap: 0.75rem;
		align-items: center;
		margin-bottom: 0.5rem;
		font-size: 0.82rem;
	}
	.link {
		background: none;
		border: none;
		color: var(--color-accent);
		cursor: pointer;
		padding: 0;
		font: inherit;
	}
	.counter {
		margin-left: auto;
		color: var(--color-text-muted);
	}
	.device-list { display: flex; flex-direction: column; gap: 0.3rem; }
	.device {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.45rem 0.7rem;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: 4px;
		cursor: pointer;
		font-size: 0.85rem;
	}
	.device.checked { border-color: var(--color-accent); }
	.device-name { color: var(--color-text-primary); flex: 0 0 auto; }
	.device-meta {
		color: var(--color-text-muted);
		font-family: var(--font-mono, ui-monospace, monospace);
		font-size: 0.75rem;
	}
	.badge-managed {
		background: rgba(120, 130, 200, 0.18);
		color: var(--color-text-muted);
		font-size: 0.65rem;
		padding: 0.1rem 0.4rem;
		border-radius: 999px;
	}
</style>
