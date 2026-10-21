<script>
  // Modrinth browser. With `instance` it installs into that instance;
  // in modpack mode (`oninstall` given) it hands the picked project back.
  import { untrack } from 'svelte'
  import { t, tErr, formatCount } from './i18n.svelte.js'
  import Icon from './Icon.svelte'
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
      <div class="seg" role="radiogroup">
        {#each types as ty (ty)}
          <button class="subtle" class:on={type === ty} role="radio" aria-checked={type === ty} onclick={() => (type = ty)}>{t('type.' + ty)}</button>
        {/each}
      </div>
    {/if}
    <div class="q">
      <Icon name="search" />
      <input type="search" bind:value={query} placeholder={t('searchPlaceholder')} spellcheck="false" />
    </div>
    <select bind:value={sort} aria-label={t('sort.relevance')}>
      {#each sorts as s (s)}<option value={s}>{t('sort.' + s)}</option>{/each}
    </select>
  </div>

  {#if type === 'shader' && instance}<p class="hint">{t('shadersHint')}</p>{/if}
  {#if error}<p class="error-text">{error}</p>{/if}

  <div class="results">
    {#each hits as hit (hit.project_id)}
      <article class="hit">
        {#if hit.icon_url}<img class="icon" src={hit.icon_url} alt="" loading="lazy" />{:else}<span class="icon"></span>{/if}
        <div class="body">
          <div class="line"><b>{hit.title}</b><span class="by">{hit.author}</span></div>
          <p class="desc">{hit.description}</p>
          <span class="by stat"><Icon name="download" size={12} />{formatCount(hit.downloads)}</span>
        </div>
        {#if installed.has(hit.project_id)}
          <span class="done"><Icon name="check" />{t('installed')}</span>
        {:else}
          <button class="get" onclick={() => install(hit)} disabled={busy[hit.project_id]}>
            {#if busy[hit.project_id]}<Icon name="loader-circle" spin />{t('installing')}{:else}<Icon name="download" />{t('install')}{/if}
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
    <div class="toast" role="status"><b>{toast.title}:</b> {toast.text}</div>
  {/if}
</div>

<style>
  .lib { position: relative; display: flex; flex-direction: column; min-height: 0; flex: 1; }
  .toolbar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
  .seg { display: flex; gap: 2px; padding: 2px; border-radius: var(--radius); background: var(--bg); }
  .seg button { height: 28px; padding: 0 10px; color: var(--muted); }
  .seg button.on { background: var(--surface); color: var(--text); box-shadow: 0 0 0 1px var(--border); }
  .q { flex: 1; min-width: 160px; position: relative; display: flex; align-items: center; }
  .q :global(.icon) { position: absolute; left: 10px; color: var(--muted); pointer-events: none; }
  .q input { padding-left: 32px; }
  select { width: auto; }
  .hint { margin: 8px 0 0; color: var(--muted); }
  .results { margin-top: 10px; overflow: auto; min-height: 0; flex: 1; }
  .hit { display: flex; gap: 12px; align-items: center; padding: 10px 2px; border-bottom: 1px solid var(--border); }
  .icon { width: 40px; height: 40px; flex: none; border-radius: var(--radius); object-fit: cover; background: var(--selected); }
  .body { flex: 1; min-width: 0; }
  .line { display: flex; gap: 8px; align-items: baseline; min-width: 0; }
  .line b { font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .by { font-size: 12px; color: var(--muted); white-space: nowrap; }
  .desc { margin: 1px 0 2px; color: var(--muted); display: -webkit-box; -webkit-line-clamp: 1; line-clamp: 1; -webkit-box-orient: vertical; overflow: hidden; }
  .get { flex: none; min-width: 118px; }
  .done { flex: none; min-width: 118px; display: inline-flex; align-items: center; justify-content: center; gap: 6px; color: var(--muted); }
  .stat { display: inline-flex; align-items: center; gap: 4px; }
  .empty { margin: 24px 0; text-align: center; color: var(--muted); }
  .more { display: block; margin: 12px auto; }
  .toast {
    position: absolute; right: 0; bottom: 0; max-width: 70%; padding: 8px 12px;
    background: var(--text); color: var(--surface); border-radius: var(--radius);
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
</style>
