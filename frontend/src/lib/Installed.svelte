<script>
  import { untrack } from 'svelte'
  import { t, tErr } from './i18n.svelte.js'
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
  <div class="empty">
    <p>{t('noContent')}. {t('noContentText')}</p>
    <button class="primary" onclick={onlibrary}>{t('openLibrary')}</button>
  </div>
{:else}
  <div class="list">
    {#each groups as [type, list] (type)}
      <h3>{t('type.' + type)} <span>{list.length}</span></h3>
      {#each list as it (it.type + '/' + it.file)}
        <div class="item" class:off={!it.enabled}>
          <input type="checkbox" checked={it.enabled} onchange={() => toggle(it)} title={it.enabled ? t('disable') : t('enable')} aria-label={it.title || it.file} />
          {#if it.iconUrl}<img class="icon" src={it.iconUrl} alt="" loading="lazy" />{:else}<span class="icon"></span>{/if}
          <div class="text">
            <span class="name">{it.title || it.file}</span>
            <small>{it.version || it.file}</small>
          </div>
          <button class="plain del" onclick={() => remove(it)} title={t('remove')} aria-label={t('remove')}>
            <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true"><path d="M3 4.5h10M6.5 4.5V3h3v1.5M4.5 4.5l.6 8.5h5.8l.6-8.5" fill="none" stroke="currentColor" stroke-width="1.2" /></svg>
          </button>
        </div>
      {/each}
    {/each}
  </div>
{/if}

<style>
  .note { margin: 0; color: var(--muted); }
  .empty { display: grid; justify-items: start; gap: 10px; }
  .empty p { margin: 0; color: var(--muted); }
  .list { flex: 1; min-height: 0; overflow: auto; }
  h3 { margin: 14px 0 4px; font-size: 13px; font-weight: 600; }
  h3:first-child { margin-top: 0; }
  h3 span { color: var(--muted); font-weight: 400; margin-left: 4px; }
  .item { display: flex; align-items: center; gap: 10px; padding: 6px 2px; border-bottom: 1px solid var(--border); }
  .item.off .icon, .item.off .text { opacity: .45; }
  .icon { width: 28px; height: 28px; flex: none; border-radius: 4px; object-fit: cover; background: var(--selected); }
  .text { flex: 1; min-width: 0; display: flex; gap: 10px; align-items: baseline; }
  .name { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .text small { font-size: 12px; color: var(--muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .del { min-height: 28px; width: 28px; padding: 0; display: grid; place-items: center; color: var(--muted); }
  .del:hover:not(:disabled) { color: var(--danger); }
</style>
