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
    <p><b>{t('noContent')}</b> {t('noContentText')}</p>
    <button class="primary" onclick={onlibrary}>{t('openLibrary')}</button>
  </div>
{:else}
  <div class="slot list">
    {#each groups as [type, list] (type)}
      <h3>{t('type.' + type)} <span>{list.length}</span></h3>
      {#each list as it (it.type + '/' + it.file)}
        <div class="item" class:off={!it.enabled}>
          <div class="slot icon">{#if it.iconUrl}<img src={it.iconUrl} alt="" loading="lazy" />{/if}</div>
          <div class="text">
            <b>{it.title || it.file}</b>
            <small>{it.version || it.file}</small>
          </div>
          <label class="check" title={it.enabled ? t('disable') : t('enable')}>
            <input type="checkbox" checked={it.enabled} onchange={() => toggle(it)} aria-label={it.title || it.file} />
          </label>
          <button class="del" onclick={() => remove(it)} title={t('remove')} aria-label={t('remove')}>
            <svg class="px" viewBox="0 0 7 7" width="12" height="12" shape-rendering="crispEdges" aria-hidden="true"><path fill="currentColor" d="M0 0h1v1H0zM1 1h1v1H1zM2 2h1v1H2zM3 3h1v1H3zM4 4h1v1H4zM5 5h1v1H5zM6 6h1v1H6zM6 0h1v1H6zM5 1h1v1H5zM4 2h1v1H4zM2 4h1v1H2zM1 5h1v1H1zM0 6h1v1H0z" /></svg>
          </button>
        </div>
      {/each}
    {/each}
  </div>
{/if}

<style>
  .note { margin: 8px 0; color: #555; }
  .empty { display: grid; justify-items: start; gap: 10px; max-width: 420px; }
  .empty p { margin: 0; }
  .list { flex: 1; min-height: 0; overflow: auto; padding: 4px 6px 6px; display: grid; align-content: start; gap: 2px; }
  h3 { margin: 8px 2px 4px; font: 16px var(--pixel); font-weight: 400; color: #fff; text-shadow: 2px 2px 0 #3f3f3f; }
  h3:first-child { margin-top: 2px; }
  h3 span { color: #ffff55; }
  .item { display: flex; align-items: center; gap: 10px; padding: 4px; border: 2px solid transparent; }
  .item:hover { background: rgb(0 0 0 / .18); border-color: rgb(255 255 255 / .35); }
  .item.off .text, .item.off .icon { opacity: .45; }
  .icon { width: 40px; height: 40px; flex: none; padding: 2px; display: grid; }
  .icon img { width: 100%; height: 100%; object-fit: cover; }
  .text { flex: 1; min-width: 0; display: grid; }
  .text b { font-weight: 600; color: #fff; text-shadow: 1px 1px 0 #3f3f3f; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .text small { font-size: 12px; color: #ececec; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .check { margin: 0; padding: 6px; cursor: pointer; display: grid; }
  .check input { width: 18px; height: 18px; cursor: pointer; }
  .del { padding: 6px; display: grid; }
  .del:hover:not(:disabled) { color: #ff5555; }
</style>
