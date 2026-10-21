<script>
  import { untrack } from 'svelte'
  import { t, tErr } from './i18n.svelte.js'
  import Icon from './Icon.svelte'
  import { ListContent, RemoveContent, SetContentEnabled } from '../../wailsjs/go/main/App.js'

  let { instance, onlibrary, version = 0 } = $props()

  let items = $state([])
  let loading = $state(true)
  let error = $state('')

  async function load() {
    loading = true
    error = ''
    try {
      items = (await ListContent(instance.id)) ?? []
    } catch (e) {
      error = tErr(e)
    } finally {
      loading = false
    }
  }

  $effect(() => {
    instance.id, version
    untrack(load)
  })

  async function toggle(it) {
    try {
      it.path = await SetContentEnabled(instance.id, it.path, !it.enabled)
      it.enabled = !it.enabled
    } catch (e) {
      error = tErr(e)
    }
  }

  async function remove(it) {
    try {
      await RemoveContent(instance.id, it.path)
      items = items.filter((x) => x !== it)
    } catch (e) {
      error = tErr(e)
    }
  }

  let groups = $derived(
    ['mod', 'resourcepack', 'shader']
      .map((type) => [type, items.filter((i) => i.type === type)])
      .filter(([, list]) => list.length),
  )
</script>

{#if error}<p class="error-text">{error}</p>{/if}
{#if loading && !items.length}
  <p class="note">{t('loading')}</p>
{:else if !items.length}
  <div class="empty-state">
    <div class="glyph"><Icon name="blocks" size={32} /></div>
    <h3>{t('noContent')}</h3>
    <p>{t('noContentText')}</p>
    <button class="primary" onclick={onlibrary}><Icon name="compass" />{t('openLibrary')}</button>
  </div>
{:else}
  <div class="list">
    {#each groups as [type, list] (type)}
      <h3 class="overline">{t('type.' + type)} <span>{list.length}</span></h3>
      {#each list as it (it.type + '/' + it.file)}
        <div class="item" class:off={!it.enabled}>
          <input type="checkbox" class="toggle" checked={it.enabled} onchange={() => toggle(it)} title={it.enabled ? t('disable') : t('enable')} aria-label={it.title || it.file} />
          {#if it.iconUrl}<img class="icon" src={it.iconUrl} alt="" loading="lazy" />{:else}<span class="icon"></span>{/if}
          <div class="text">
            <span class="name">{it.title || it.file}</span>
            <small class="mono">{it.version || it.file}</small>
          </div>
          <button class="subtle icon danger" onclick={() => remove(it)} title={t('remove')} aria-label={t('remove')}><Icon name="trash-2" /></button>
        </div>
      {/each}
    {/each}
  </div>
{/if}

<style>
  .note { margin-top: var(--sp-4); color: var(--text-secondary); }
  .empty-state { margin-top: var(--sp-10); }
  .list { margin-top: var(--sp-3); }
  h3 { margin: var(--sp-5) 0 var(--sp-1); }
  h3:first-child { margin-top: var(--sp-2); }
  h3 span { margin-left: var(--sp-1); }
  .item { display: flex; align-items: center; gap: var(--sp-3); height: 44px; padding: 0 var(--sp-1) 0 var(--sp-2); border-bottom: 1px solid var(--border-subtle); transition: background-color var(--dur-fast); }
  .item:hover { background: var(--bg-surface-2); }
  .item.off .icon, .item.off .text { opacity: 0.45; }
  .icon { width: 28px; height: 28px; flex: none; border-radius: var(--r-xs); object-fit: cover; background: var(--bg-surface-3); }
  .text { flex: 1; min-width: 0; display: flex; gap: var(--sp-3); align-items: baseline; }
  .name { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .text small { font-size: var(--fs-12); color: var(--text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
</style>
