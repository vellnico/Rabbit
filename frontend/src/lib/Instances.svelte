<script>
  import Icon from './Icon.svelte'
  import { t } from './i18n.svelte.js'
  import { loaders } from './theme.js'

  let { app, selected, status, onpick, oncreate } = $props()

  let query = $state('')
  let list = $derived((app.instances ?? []).filter((i) => i.name.toLowerCase().includes(query.trim().toLowerCase())))
  const loaderOf = (inst) => loaders[inst.loader] ?? loaders.vanilla
</script>

<header class="head">
  <h1>{t('instances')} <span class="count">{app.instances?.length ?? 0}</span></h1>
  <div class="tools">
    {#if app.instances?.length}
      <div class="q">
        <Icon name="search" size={14} />
        <input type="search" bind:value={query} placeholder={t('searchPlaceholderShort')} spellcheck="false" />
      </div>
    {/if}
    <button class="primary" onclick={oncreate}><Icon name="plus" />{t('newInstance')}</button>
  </div>
</header>

{#if !app.instances?.length}
  <div class="empty-state">
    <div class="glyph"><Icon name="layout-grid" size={32} /></div>
    <h3>{t('welcome')}</h3>
    <p>{t('welcomeSub')}</p>
  </div>
{:else if !list.length}
  <p class="nothing">{t('nothingFound')}</p>
{:else}
  <div class="grid">
    {#each list as inst (inst.id)}
      <button class="card" class:sel={selected?.id === inst.id} onclick={() => onpick(inst.id)}>
        <span class="glyph"><Icon name={inst.loader === 'vanilla' ? 'package' : 'blocks'} size={20} /></span>
        <span class="text">
          <b>{inst.name}</b>
          <span class="meta">
            <span class="mono">{inst.gameVersion}</span>
            <span class="badge" style="--dot:{loaderOf(inst).color}"><span class="dot"></span>{loaderOf(inst).name}</span>
          </span>
        </span>
        {#if status[inst.id]?.state === 'running'}
          <span class="badge success">{t('runningShort')}</span>
        {:else if selected?.id === inst.id}
          <span class="badge accent">{t('active')}</span>
        {/if}
      </button>
    {/each}
  </div>
{/if}

<style>
  .head { display: flex; align-items: center; justify-content: space-between; gap: var(--sp-4); min-height: var(--control-h-md); margin-bottom: var(--sp-6); }
  h1 { font-size: var(--fs-24); font-weight: var(--fw-semibold); letter-spacing: var(--tracking-tight); line-height: var(--lh-tight); }
  .count { margin-left: var(--sp-1); font-size: var(--fs-12); font-weight: var(--fw-regular); letter-spacing: 0; color: var(--text-tertiary); }
  .tools { display: flex; gap: var(--sp-2); }
  .q { position: relative; width: 220px; display: flex; align-items: center; }
  .q :global(svg) { position: absolute; left: 10px; color: var(--text-tertiary); pointer-events: none; }
  .q input { padding-left: var(--sp-8); }
  .nothing { color: var(--text-secondary); }

  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(var(--card-min), 1fr)); gap: var(--sp-3); }
  .card {
    height: auto; display: flex; align-items: center; justify-content: flex-start; gap: var(--sp-3);
    padding: 14px; text-align: left; font-weight: var(--fw-regular);
    background: var(--bg-surface-2); border: 1px solid var(--border-subtle); border-radius: var(--r-md);
  }
  .card:hover:not(:disabled) { background: var(--bg-surface-3); border-color: var(--border-default); }
  .card.sel { border-color: var(--accent-a24); }
  .card .glyph { flex: none; width: var(--rail-item); height: var(--rail-item); display: grid; place-items: center; border-radius: var(--r-sm); background: var(--bg-surface-4); color: var(--text-secondary); }
  .text { flex: 1; min-width: 0; display: grid; gap: var(--sp-1); }
  .text b { font-weight: var(--fw-semibold); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .meta { display: flex; align-items: center; gap: var(--sp-2); font-size: var(--fs-12); color: var(--text-secondary); }
  .card > .badge { align-self: flex-start; }
</style>
