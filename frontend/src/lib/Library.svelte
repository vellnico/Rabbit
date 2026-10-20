<script>
  // Modrinth browser. With `instance` it installs into that instance;
  // in modpack mode (`oninstall` given) it hands the picked project back.
  import { untrack } from 'svelte'
  import { fade, fly } from 'svelte/transition'
  import { t, tErr, formatCount } from './i18n.svelte.js'
  import { SearchContent, InstallContent, InstalledProjects } from '../../wailsjs/go/main/App.js'

  let { instance = null, types, oninstall = null, onchanged = () => {} } = $props()

  let type = $state(untrack(() => types[0]))
  let query = $state('')
  let sort = $state('relevance')
  let hits = $state([])
  let total = $state(0)
  let loading = $state(false)
  let error = $state('')
  let installed = $state(new Set())
  let busy = $state({}) // project id -> true while installing
  let toast = $state('')
  let toastTimer
  let seq = 0

  const sorts = ['relevance', 'downloads', 'follows', 'newest', 'updated']
  let modsBlocked = $derived(type === 'mod' && instance?.loader === 'vanilla')

  async function loadInstalled() {
    if (instance) installed = new Set((await InstalledProjects(instance.id)) ?? [])
  }

  async function search(append = false) {
    if (modsBlocked) {
      hits = []
      total = 0
      return
    }
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
    const id = setTimeout(() => untrack(() => search()), 300)
    return () => clearTimeout(id)
  })
  $effect(() => {
    instance?.id
    untrack(loadInstalled)
  })

  function showToast(text) {
    toast = text
    clearTimeout(toastTimer)
    toastTimer = setTimeout(() => (toast = ''), 3500)
  }

  async function install(hit) {
    busy[hit.project_id] = true
    error = ''
    try {
      if (oninstall) {
        await oninstall(hit)
      } else {
        const titles = await InstallContent(instance.id, hit.project_id, hit.project_type)
        await loadInstalled()
        showToast(t('installedList', { list: (titles ?? []).join(', ') }))
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
      <div class="seg">
        {#each types as ty (ty)}
          <button class:on={type === ty} onclick={() => (type = ty)}>{t('type.' + ty)}</button>
        {/each}
      </div>
    {/if}
    <div class="search">
      <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true"><path d="M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14Zm9 16-4-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" /></svg>
      <input bind:value={query} placeholder={t('searchPlaceholder')} spellcheck="false" />
    </div>
    <select bind:value={sort} aria-label="sort">
      {#each sorts as s (s)}<option value={s}>{t('sort.' + s)}</option>{/each}
    </select>
  </div>

  {#if modsBlocked}
    <p class="hint">{t('modsNeedLoader')}</p>
  {:else if type === 'shader' && instance}
    <p class="hint">{t('shadersHint')}</p>
  {/if}
  {#if error}<p class="error-text">{error}</p>{/if}

  <div class="results">
    {#each hits as hit, i (hit.project_id)}
      <article class="card" in:fly={{ y: 10, duration: 240, delay: Math.min(i % 20, 10) * 25 }}>
        {#if hit.icon_url}
          <img class="icon" src={hit.icon_url} alt="" loading="lazy" />
        {:else}
          <div class="icon letter">{hit.title[0]}</div>
        {/if}
        <div class="body">
          <div class="title"><b>{hit.title}</b><span class="author">{hit.author}</span></div>
          <p class="desc">{hit.description}</p>
          <div class="meta">
            <span title="downloads">
              <svg viewBox="0 0 24 24" width="13" height="13" aria-hidden="true"><path d="M12 4v11m0 0-4-4m4 4 4-4M5 20h14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" /></svg>
              {formatCount(hit.downloads)}
            </span>
            <span title="followers">
              <svg viewBox="0 0 24 24" width="13" height="13" aria-hidden="true"><path d="M12 20s-7-4.4-7-10a4 4 0 0 1 7-2.6A4 4 0 0 1 19 10c0 5.6-7 10-7 10Z" fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round" /></svg>
              {formatCount(hit.follows)}
            </span>
            {#each (hit.display_categories ?? []).slice(0, 3) as c (c)}<span class="cat">{c}</span>{/each}
          </div>
        </div>
        {#if installed.has(hit.project_id)}
          <span class="done" in:fade><svg viewBox="0 0 24 24" width="15" height="15" aria-hidden="true"><path d="M5 12.5l4.5 4.5L19 7.5" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round" /></svg>{t('installed')}</span>
        {:else}
          <button class="primary inst" onclick={() => install(hit)} disabled={busy[hit.project_id]}>
            {#if busy[hit.project_id]}<i class="spin"></i>{t('installing')}{:else}{t('install')}{/if}
          </button>
        {/if}
      </article>
    {:else}
      {#if !loading && !modsBlocked && !error}<p class="empty">{t('nothingFound')}</p>{/if}
    {/each}

    {#if loading}
      {#each Array(hits.length ? 1 : 4) as _, i (i)}<div class="card skeleton"></div>{/each}
    {:else if hits.length < total}
      <button class="more" onclick={() => search(true)}>{t('loadMore')}</button>
    {/if}
  </div>

  {#if toast}<div class="toast" transition:fly={{ y: 16, duration: 220 }}>{toast}</div>{/if}
</div>

<style>
  .lib { position: relative; display: flex; flex-direction: column; min-height: 0; flex: 1; }
  .toolbar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
  .seg { display: flex; background: var(--bg); border: 1px solid var(--line); border-radius: 10px; padding: 3px; gap: 2px; }
  .seg button { border: none; background: transparent; color: var(--muted); padding: 6px 12px; border-radius: 7px; }
  .seg button:hover:not(:disabled) { background: var(--raised); color: var(--text); }
  .seg button.on { background: var(--raised-2); color: var(--text); box-shadow: inset 0 -2px 0 var(--accent); }
  .search { flex: 1; min-width: 180px; position: relative; display: flex; align-items: center; }
  .search svg { position: absolute; left: 11px; color: var(--muted); pointer-events: none; }
  .search input { padding-left: 34px; }
  select { width: auto; }
  .hint { margin: 10px 0 0; color: var(--muted); font-size: 12.5px; }
  .results { margin-top: 12px; overflow: auto; min-height: 0; flex: 1; display: grid; align-content: start; gap: 8px; padding-right: 4px; }
  .card {
    display: flex; gap: 14px; align-items: center; padding: 12px; border-radius: 12px;
    background: rgb(30 30 34 / .65); border: 1px solid var(--line);
    transition: border-color .2s, transform .2s var(--ease), background .2s;
  }
  .card:hover { border-color: var(--line-2); background: rgb(34 34 39 / .8); }
  .icon { width: 52px; height: 52px; border-radius: 12px; object-fit: cover; flex: none; background: var(--raised); }
  .letter { display: grid; place-items: center; font-weight: 700; font-size: 20px; color: var(--accent); background: rgb(var(--accent-rgb) / .12); }
  .body { flex: 1; min-width: 0; }
  .title { display: flex; align-items: baseline; gap: 8px; min-width: 0; }
  .title b { font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .author { color: var(--muted); font-size: 12px; white-space: nowrap; }
  .desc { margin: 3px 0 6px; color: var(--muted); font-size: 12.5px; display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
  .meta { display: flex; gap: 10px; align-items: center; color: var(--muted); font-size: 11.5px; flex-wrap: wrap; }
  .meta span { display: inline-flex; align-items: center; gap: 4px; }
  .cat { padding: 1px 7px; border-radius: 999px; background: rgb(255 255 255 / .05); border: 1px solid var(--line); text-transform: capitalize; }
  .inst { flex: none; min-width: 116px; display: flex; align-items: center; justify-content: center; gap: 7px; }
  .done { flex: none; min-width: 116px; display: inline-flex; align-items: center; justify-content: center; gap: 6px; color: var(--accent); font-size: 13px; font-weight: 600; }
  .spin { width: 13px; height: 13px; border-radius: 50%; border: 2px solid currentColor; border-right-color: transparent; animation: spin .7s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .skeleton { height: 78px; background: linear-gradient(90deg, var(--raised) 0%, var(--raised-2) 50%, var(--raised) 100%); background-size: 200% 100%; animation: shimmer 1.2s linear infinite; border-color: transparent; }
  @keyframes shimmer { from { background-position: 200% 0; } to { background-position: -200% 0; } }
  .empty { color: var(--muted); text-align: center; margin: 30px 0; }
  .more { justify-self: center; margin: 6px 0 10px; }
  .toast {
    position: absolute; left: 50%; bottom: 8px; transform: translateX(-50%); max-width: 90%;
    background: var(--raised-2); border: 1px solid rgb(var(--accent-rgb) / .5); color: var(--text);
    padding: 9px 16px; border-radius: 10px; font-size: 13px; box-shadow: 0 12px 30px -10px rgb(0 0 0 / .7);
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
</style>
