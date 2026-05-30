<!--
  Источник дизайна: singbox-router/project/screens/EmptyState.jsx (EmptyStateScreen)
  Главная композиция: Hero + Quick Start (3 шага) + Recipes.
-->

<script lang="ts">
  import { onMount } from 'svelte';
  import { Plus, Power, Globe } from 'lucide-svelte';
  import { singboxRouter as singboxRouterStore } from '$lib/stores/singboxRouter';
  import { notifications } from '$lib/stores/notifications';
  import { Button } from '$lib/components/ui';
  import { api } from '$lib/api/client';
  import type { SingboxRouterWANInterface } from '$lib/types';
  import EmptyHero from './EmptyHero.svelte';
  import StepCard from './StepCard.svelte';
  import type { StepState } from './StepCard.svelte';
  import RecipeCard from './RecipeCard.svelte';
  import { RECIPES } from './recipes';
  import {
    applyRecipe, createDefaultPolicy, setAutoDetectWan, setManualWan, enableEngine,
  } from './emptyStateActions';
  import { openAddWizard } from './addWizardStore';

  const status = singboxRouterStore.status;
  const settings = singboxRouterStore.settings;

  let busyStep = $state<'policy' | 'wan' | 'enable' | null>(null);
  let wanInterfaces = $state<SingboxRouterWANInterface[]>([]);
  let wanLoadFailed = $state(false);

  onMount(async () => {
    void singboxRouterStore.loadAll();
    try {
      wanInterfaces = await api.singboxRouterListWANInterfaces();
    } catch {
      wanLoadFailed = true;
    }
  });

  const step1Done = $derived(($status?.policyName ?? '') !== '');
  const step2Done = $derived(
    ($settings?.wanAutoDetect ?? false) || ($settings?.wanInterface ?? '') !== '',
  );
  const step3Done = $derived($status?.enabled ?? false);

  function stateFor(n: 1 | 2 | 3): StepState {
    if (n === 1) return step1Done ? 'done' : 'active';
    if (n === 2) {
      if (step2Done) return 'done';
      return step1Done ? 'active' : 'upcoming';
    }
    if (step3Done) return 'done';
    return step1Done && step2Done ? 'active' : 'upcoming';
  }

  async function handlePolicy() {
    busyStep = 'policy';
    try {
      await createDefaultPolicy();
      notifications.success('Policy «awgm-router» создана');
    } catch (e) {
      notifications.error(`Ошибка: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      busyStep = null;
    }
  }

  async function handleAutoWan() {
    busyStep = 'wan';
    try {
      await setAutoDetectWan();
      notifications.success('WAN: авто-определение');
    } catch (e) {
      notifications.error(`Ошибка: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      busyStep = null;
    }
  }

  async function handleManualWan(e: Event) {
    const iface = (e.currentTarget as HTMLSelectElement).value;
    if (!iface) return;
    busyStep = 'wan';
    try {
      await setManualWan(iface);
      notifications.success(`WAN: ${iface}`);
    } catch (err) {
      notifications.error(`Ошибка: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      busyStep = null;
    }
  }

  async function handleEnable() {
    busyStep = 'enable';
    try {
      await enableEngine();
      notifications.success('Движок включён');
    } catch (e) {
      notifications.error(`Ошибка: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      busyStep = null;
    }
  }

  async function handleRecipe(id: string) {
    try {
      await applyRecipe(id);
    } catch (e) {
      notifications.error(`Recipe не доступен: ${e instanceof Error ? e.message : String(e)}`);
    }
  }
</script>

{#snippet iconPlus()}<Plus size={14} />{/snippet}
{#snippet iconPower()}<Power size={14} />{/snippet}
{#snippet iconGlobe()}<Globe size={14} />{/snippet}

<div class="wrap">
  <EmptyHero />

  <section class="section">
    <header class="sec-head">
      <h3 class="sec-title">Запустить за 3 шага</h3>
      <p class="sec-sub">После запуска вы сможете вернуться и настроить правила</p>
    </header>

    <div class="grid-3">
      <StepCard
        n={1}
        title="Выбрать устройства"
        body="Какие устройства роутера должны идти через sing-box. Создайте policy «awgm-router», затем привяжите устройства в LAN-настройках."
        state={stateFor(1)}
      >
        {#snippet cta()}
          {#if step1Done}
            <span class="done-label">Готово · {$status?.policyName ?? ''}</span>
          {:else}
            <Button
              variant="primary"
              size="md"
              fullWidth
              disabled={busyStep !== null}
              onclick={handlePolicy}
              iconBefore={iconPlus}
            >
              Создать policy «awgm-router»
            </Button>
          {/if}
        {/snippet}
      </StepCard>

      <StepCard
        n={2}
        title="Указать WAN"
        body="Через какой внешний интерфейс sing-box будет отправлять трафик в обычный интернет."
        state={stateFor(2)}
        disabled={!step1Done}
      >
        {#snippet cta()}
          {#if step2Done}
            <span class="done-label">
              Готово · {$settings?.wanAutoDetect ? 'auto' : ($settings?.wanInterface ?? '')}
            </span>
          {:else}
            <Button
              variant="primary"
              size="md"
              fullWidth
              disabled={!step1Done || busyStep !== null}
              onclick={handleAutoWan}
              iconBefore={iconGlobe}
            >
              Авто-определение
            </Button>
          {/if}
        {/snippet}
        {#snippet extra()}
          {#if !step2Done && step1Done && wanInterfaces.length > 0}
            <label class="manual-wan">
              <span class="manual-label">или вручную:</span>
              <select
                class="wan-select"
                disabled={busyStep !== null}
                onchange={handleManualWan}
              >
                <option value="">— выберите интерфейс —</option>
                {#each wanInterfaces as iface (iface.name)}
                  <option value={iface.name}>
                    {iface.name}{iface.label ? ` — ${iface.label}` : ''}
                  </option>
                {/each}
              </select>
            </label>
          {:else if wanLoadFailed}
            <span class="hint-mini">Не удалось загрузить список интерфейсов</span>
          {/if}
        {/snippet}
      </StepCard>

      <StepCard
        n={3}
        title="Включить движок"
        body="После включения трафик пойдёт по правилам. Без правил → весь трафик идёт напрямую."
        state={stateFor(3)}
        disabled={!step1Done || !step2Done}
      >
        {#snippet cta()}
          {#if step3Done}
            <span class="done-label">Запущен</span>
          {:else}
            <Button
              variant="primary"
              size="md"
              fullWidth
              disabled={!step1Done || !step2Done || busyStep !== null}
              onclick={handleEnable}
              iconBefore={iconPower}
            >
              Включить
            </Button>
          {/if}
        {/snippet}
      </StepCard>
    </div>

    <div class="recipes">
      <header class="rec-head">
        <h3 class="rec-title">Готовые сценарии</h3>
        <p class="rec-sub">Можно начать с шаблона — потом настроить под себя</p>
      </header>
      <div class="grid-3">
        {#each RECIPES as r (r.id)}
          <RecipeCard
            tag={r.tag}
            title={r.title}
            desc={r.desc}
            count={r.count}
            tint={r.tint}
            onclick={() => handleRecipe(r.id)}
          />
        {/each}
      </div>
    </div>

    {#if step3Done}
      <div class="post-enable">
        <p class="post-text">Движок запущен. Создайте первое правило, чтобы трафик пошёл через туннель.</p>
        <Button variant="primary" size="md" onclick={() => openAddWizard()} iconBefore={iconPlus}>
          Создать первое правило
        </Button>
      </div>
    {/if}
  </section>
</div>

<style>
  .wrap {
    max-width: 880px;
    margin: 0 auto;
    padding: var(--sp-4);
  }
  .section {
    margin-top: 8px;
  }
  .sec-head {
    margin-bottom: 12px;
  }
  .sec-title {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
  }
  .sec-sub {
    margin: 4px 0 0;
    font-size: 12.5px;
    color: var(--text-muted);
  }
  .grid-3 {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
  }
  @media (max-width: 768px) {
    .grid-3 {
      grid-template-columns: 1fr;
    }
  }
  .done-label {
    flex: 1;
    text-align: center;
    padding: 8px 12px;
    border-radius: var(--radius-sm);
    background: rgba(115, 156, 122, 0.14);
    color: var(--color-success, #22c55e);
    font-size: 12px;
    font-weight: 500;
  }
  .manual-wan {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .manual-label {
    font-size: 11px;
    color: var(--text-muted);
  }
  .wan-select {
    padding: 4px 8px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border);
    background: var(--bg-primary);
    color: var(--text-primary);
    font-family: inherit;
    font-size: 12px;
  }
  .hint-mini {
    font-size: 11px;
    color: var(--text-muted);
    font-style: italic;
  }
  .recipes {
    margin-top: 24px;
  }
  .rec-head {
    margin-bottom: 10px;
  }
  .rec-title {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
  }
  .rec-sub {
    margin: 4px 0 0;
    font-size: 12px;
    color: var(--text-muted);
  }
  .post-enable {
    margin-top: 24px;
    padding: 16px;
    background: var(--bg-secondary);
    border: 1px dashed var(--accent-line);
    border-radius: var(--radius);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .post-text {
    margin: 0;
    font-size: 13px;
    color: var(--text-secondary);
  }
</style>
