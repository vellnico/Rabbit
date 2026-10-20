<script>
  // Modrinth browser. With `instance` it installs into that instance;
  // in modpack mode (`oninstall` given) it hands the picked project back.
  import { untrack } from 'svelte'
  import { t, tErr, formatCount } from './i18n.svelte.js'
  import { SearchContent, InstallContent, InstalledProjects } from '../../wailsjs/go/main/App.js'

  let { instance = null, types, oninstall = null, onchanged = () => {} } = $props()

  let type = $state(untrack(() => types[0]))
  let query = $state('')
  let sort = $state('relevance')
  let hits = $state([])
  let total = $state(0)
  let loading = $state(true)
  let error = $state('')
  let installed = $state(new Set())
  let busy = $state({}) // project id -> true while installing
  let toast = $state(null) // { title, text }
  let toastTimer
  let seq = 0

  const sorts = ['relevance', 'downloads', 'follows', 'newest', 'updated']

  async function loadInstalled() {
    if (instance) installed = new Set((await InstalledProjects(instance.id)) ?? [])
  }

  async function search(append = false) {
    const my = ++seq
    loading = true
    error = ''
    try {
      const res = await SearchContent({
        query,
        type,
        sort,
        offset: append ? hits.length : 0,
        gameVersion: instance?.gameVersion ?? '',
        loader: instance?.loader ?? '',
      })
      if (my !== seq) return
      hits = append ? [...hits, ...(res.hits ?? [])] : (res.hits ?? [])
      total = res.total_hits
    } catch (e) {
      if (my === seq) error = tErr(e)
    } finally {
      if (my === seq) loading = false
    }
  }

  // Search again (debounced) when the query, type or sort changes.
  $effect(() => {
    query, type, sort
    loading = true
    const id = setTimeout(() => untrack(() => search()), 300)
    return () => clearTimeout(id)
  })
  $effect(() => {
    instance?.id
    untrack(loadInstalled)
  })

  async function install(hit) {
    busy[hit.project_id] = true
    error = ''
    try {
      if (oninstall) {
        await oninstall(hit)
      } else {
        const titles = await InstallContent(instance.id, hit.project_id, hit.project_type)
        await loadInstalled()
        toast = { title: t('installed'), text: (titles ?? []).join(', ') }
        clearTimeout(toastTimer)
        toastTimer = setTimeout(() => (toast = null), 4000)
        onchanged()
      }
    } catch (e) {
      error = tErr(e)
    } finally {
      delete busy[hit.project_id]
    }
  }
</script>

<div class="lib">
  <div class="toolbar">
    {#if types.length > 1}
      <div class="kinds">
        {#each types as ty (ty)}
          <button class:on={type === ty} aria-pressed={type === ty} onclick={() => (type = ty)}>{t('type.' + ty)}</button>
        {/each}
      </div>
    {/if}
    <input class="q" bind:value={query} placeholder={t('searchPlaceholder')} spellcheck="false" />
    <select bind:value={sort} aria-label={t('sort.relevance')}>
      {#each sorts as s (s)}<option value={s}>{t('sort.' + s)}</option>{/each}
    </select>
  </div>

  {#if type === 'shader' && instance}<p class="hint">{t('shadersHint')}</p>{/if}
  {#if error}<p class="error-text">{error}</p>{/if}

  <div class="slot results">
    {#each hits as hit (hit.project_id)}
      <article class="hit">
        <div class="slot icon">
          {#if hit.icon_url}<img src={hit.icon_url} alt="" loading="lazy" />{/if}
        </div>
        <div class="body">
          <b class="title">{hit.title}</b>
          <span class="by">{hit.author}</span>
          <p class="desc">{hit.description}</p>
          <span class="downloads">
            <svg class="px" viewBox="0 0 5 6" width="10" height="12" shape-rendering="crispEdges" aria-hidden="true"><path fill="currentColor" d="M2 0h1v3h1v1H3v1H2V4H1V3h1zM0 3h1v1H0zM4 3h1v1H4zM0 5h5v1H0z" /></svg>
            {formatCount(hit.downloads)}
          </span>
        </div>
        {#if installed.has(hit.project_id)}
          <span class="done">{t('installed')}</span>
        {:else}
          <button class="primary get" onclick={() => install(hit)} disabled={busy[hit.project_id]}>
            {busy[hit.project_id] ? t('installing') : t('install')}
          </button>
        {/if}
      </article>
    {:else}
      {#if !loading && !error}<p class="empty">{t('nothingFound')}</p>{/if}
    {/each}

    {#if loading}
      <p class="empty">{t('loading')}</p>
    {:else if hits.length < total}
      <button class="more" onclick={() => search(true)}>{t('loadMore')}</button>
    {/if}
  </div>

  {#if toast}
    <!-- An advancement-style toast. -->
    <div class="toast" role="status">
      <b>{toast.title}</b>
      <span>{toast.text}</span>
    </div>
  {/if}
</div>

<style>
  .lib { position: relative; display: flex; flex-direction: column; min-height: 0; flex: 1; }
  .toolbar { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; }
  .kinds { display: flex; gap: 4px; }
  .kinds button { font-size: 14px; padding: 7px 10px; }
  /* The chosen kind looks pressed in. */
  .kinds button.on { background: #6f6f6f; box-shadow: inset 2px 2px 0 #4a4a4a, inset -2px -2px 0 #9a9a9a; outline: 2px solid var(--accent); outline-offset: -4px; }
  .q { flex: 1; min-width: 160px; }
  select { width: auto; }
  .hint { margin: 8px 0 0; font-size: 13px; color: #555; }

  .results { margin-top: 8px; padding: 4px; overflow: auto; min-height: 0; flex: 1; display: grid; align-content: start; gap: 2px; }
  /* Rows read like the multiplayer server list. */
  .hit { display: flex; gap: 10px; align-items: center; padding: 6px; border: 2px solid transparent; }
  .hit:hover { background: rgb(0 0 0 / .18); border-color: rgb(255 255 255 / .35); }
  .icon { width: 52px; height: 52px; flex: none; padding: 2px; display: grid; }
  .icon img { width: 100%; height: 100%; object-fit: cover; }
  .body { flex: 1; min-width: 0; display: grid; grid-template-columns: auto 1fr; column-gap: 8px; align-items: baseline; }
  .title { font: 16px var(--pixel); font-weight: 400; color: #fff; text-shadow: 2px 2px 0 #3f3f3f; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .by { font-size: 12px; color: #e2e2e2; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .desc { grid-column: 1 / -1; margin: 2px 0 3px; color: #f2f2f2; font-size: 12.5px; display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
  .downloads { grid-column: 1 / -1; display: inline-flex; align-items: center; gap: 5px; font-size: 12px; color: #e2e2e2; }
  .get { flex: none; min-width: 120px; font-size: 15px; }
  .done { flex: none; min-width: 120px; text-align: center; font: 15px var(--pixel); color: #b6ff7f; text-shadow: 2px 2px 0 #2c4a14; }
  .empty { margin: 24px 0; text-align: center; color: #fff; text-shadow: 1px 1px 0 #3f3f3f; }
  .more { justify-self: center; margin: 8px 0; }

  .toast {
    position: fixed; top: 16px; right: 16px; z-index: 20; width: 300px;
    display: grid; gap: 2px; padding: 10px 12px;
    background: #212121; border: 2px solid #000; box-shadow: inset 2px 2px 0 #555, inset -2px -2px 0 #111;
    animation: drop .9s steps(6) both;
  }
  .toast b { font: 15px var(--pixel); font-weight: 400; color: #ffff55; }
  .toast span { color: #fff; font-size: 13px; }
  @keyframes drop { from { transform: translateY(-120%); } 30% { transform: none; } }
</style>
