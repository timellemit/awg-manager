<script lang="ts">
    import { onMount } from 'svelte';
    import { api } from '$lib/api/client';
    import { singboxWizard } from '$lib/stores/singboxWizard';
    import type { RouterPolicy } from '$lib/types';

    const wizardState = singboxWizard.state;

    let policies = $state<RouterPolicy[]>([]);
    let loading = $state(true);

    onMount(async () => {
        try {
            policies = await api.singboxRouterListPolicies();
        } catch {
            policies = [];
        }
        loading = false;
    });

    const mode = $derived($wizardState.policyMode);
    const newName = $derived($wizardState.policyName);
    const existing = $derived($wizardState.existingPolicyName);

    const selectedPolicy = $derived(
        existing ? policies.find((p) => p.name === existing) : undefined,
    );

    function setMode(m: 'create' | 'existing'): void {
        singboxWizard.setPolicyMode(m);
    }
    function onNameInput(e: Event): void {
        singboxWizard.setPolicyName((e.target as HTMLInputElement).value);
    }
    function onExistingChange(e: Event): void {
        const v = (e.target as HTMLSelectElement).value;
        singboxWizard.setExistingPolicyName(v || null);
    }
</script>

<div class="title">Через какую policy будем привязывать устройства?</div>
<div class="hint">Policy управляет какие устройства идут через VPN.</div>

<label class="option" class:checked={mode === 'create'}>
    <input
        type="radio"
        name="policy-mode"
        value="create"
        checked={mode === 'create'}
        onchange={() => setMode('create')}
    />
    <span class="option-name">Создать новую</span>
</label>
{#if mode === 'create'}
    <div class="sub-block">
        <label class="lbl" for="policy-new-name">Имя policy</label>
        <input
            id="policy-new-name"
            class="input"
            value={newName}
            oninput={onNameInput}
            placeholder="SBRouter"
        />
    </div>
{/if}

<label class="option" class:checked={mode === 'existing'}>
    <input
        type="radio"
        name="policy-mode"
        value="existing"
        checked={mode === 'existing'}
        onchange={() => setMode('existing')}
    />
    <span class="option-name">Использовать существующую</span>
</label>
{#if mode === 'existing'}
    <div class="sub-block">
        {#if loading}
            <div class="muted">Загрузка списка...</div>
        {:else if policies.length === 0}
            <div class="muted">Существующих policies нет — выберите "Создать новую".</div>
        {:else}
            <select class="input" onchange={onExistingChange}>
                <option value="">— выбрать —</option>
                {#each policies as p (p.name)}
                    <option value={p.name} selected={existing === p.name}>
                        {p.description} ({p.deviceCount} устройств)
                    </option>
                {/each}
            </select>
            {#if selectedPolicy && selectedPolicy.deviceCount > 0}
                <div class="warning">
                    В этой policy уже {selectedPolicy.deviceCount} устройств.
                    На следующем шаге сможете добавить новые или убрать существующие.
                </div>
            {/if}
        {/if}
    </div>
{/if}

<style>
    .title {
        font-size: 1.05rem;
        color: var(--color-text-primary);
        font-weight: 600;
        margin-bottom: 0.6rem;
    }
    .hint {
        color: var(--color-text-muted);
        font-size: 0.85rem;
        margin-bottom: 1rem;
    }
    .option {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        padding: 0.625rem 0.875rem;
        background: var(--color-bg-secondary);
        border: 1px solid var(--color-border);
        border-radius: 6px;
        cursor: pointer;
        margin-bottom: 0.5rem;
    }
    .option.checked {
        border-color: var(--color-accent);
        background: rgba(122, 162, 247, 0.08);
    }
    .option-name {
        font-size: 0.875rem;
        color: var(--color-text-primary);
        font-weight: 500;
    }
    .sub-block {
        margin: 0.5rem 0 1rem 1.5rem;
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }
    .lbl {
        font-size: 0.75rem;
        color: var(--color-text-muted);
    }
    .input {
        background: var(--color-bg-primary);
        border: 1px solid var(--color-border);
        padding: 0.5rem 0.7rem;
        border-radius: 4px;
        color: var(--color-text-primary);
        font-family: inherit;
        font-size: 0.875rem;
    }
    .muted {
        color: var(--color-text-muted);
        font-size: 0.85rem;
    }
    .warning {
        background: rgba(224, 175, 104, 0.12);
        border-left: 3px solid var(--warning, #e0af68);
        padding: 0.5rem 0.75rem;
        font-size: 0.82rem;
        color: var(--color-text-primary);
        border-radius: 0 4px 4px 0;
    }
</style>
