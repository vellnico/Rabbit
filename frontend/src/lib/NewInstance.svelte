<script>
  import { onMount } from 'svelte'
  import Modal from './Modal.svelte'
  import Library from './Library.svelte'
  import { t, tErr } from './i18n.svelte.js'
  import { loaders } from './theme.js'
  import { ListVersions, ListLoaderVersions, CreateInstance, ImportModpack, InstallModpack } from '../../wailsjs/go/main/App.js'
  import { EventsOn } from '../../wailsjs/runtime/runtime.js'

  let { onclose, oncreated } = $props()

  let versions = $state([])
  let loadError = $state('')
  let snapshots = $state(false)
  let name = $state('')
  let version = $state('')
  let loader = $state('vanilla')
  let loaderVersions = $state([])
  let loaderLoading = $state(false)
  let loaderVersion = $state('')
  let busy = $state(false)
  let error = $state('')
  let mode = $state('create') // create | modpacks | import
  let importing = $state(null) // {stage, done, total} while a modpack installs

  const modes = ['create', 'modpacks', 'import']

  onMount(() => EventsOn('import', (e) => (importing = e)))

  // Shared by "modpacks" and "import": runs the backend call with progress.
  async function runImport(fn) {
    importing = { stage: 'read', done: 0, total: 0 }
    error = ''
    try {
      const inst = await fn()
      if (inst?.id) oncreated(inst)
    } catch (e) {
      error = tErr(e)
    } finally {
      importing = null
    }
  }

  let shown = $derived(versions.filter((v) => v.type === 'release' || (snapshots && v.type === 'snapshot')))

  ListVersions()
    .then((list) => {
      versions = list
      version = list.find((v) => v.type === 'release')?.id ?? ''
    })
    .catch((e) => (loadError = tErr(e)))

  $effect(() => {
    if (loader === 'vanilla' || !version) return
    const l = loader
    const v = version
    loaderVersions = []
    loaderVersion = ''
    loaderLoading = true
    ListLoaderVersions(l, v)
      .then((list) => {
        if (l !== loader || v !== version) return
        loaderVersions = list ?? []
        loaderVersion = loaderVersions[0] ?? ''
      })
      .catch(() => (loaderVersions = []))
      .finally(() => {
        if (l === loader && v === version) loaderLoading = false
      })
  })

  async function create() {
    busy = true
    error = ''
    try {
      oncreated(await CreateInstance(name, version, loader, loaderVersion))
    } catch (e) {
      error = tErr(e)
    } finally {
      busy = false
    }
  }
</script>

<Modal title={t('newInstance')} onclose={() => !importing && onclose()} width={mode === 'modpacks' ? 760 : 460}>
  <div class="seg" role="radiogroup">
    {#each modes as m (m)}
      <button class="plain" class:on={mode === m} role="radio" aria-checked={mode === m} onclick={() => { mode = m; error = '' }} disabled={!!importing}>{t('new.' + m)}</button>
    {/each}
  </div>

  {#if importing}
    <div class="importing">
      <span>{t('import.' + importing.stage)}{#if importing.total}, {importing.done} / {importing.total}{/if}</span>
      <div class="bar" class:indeterminate={!importing.total}><div style="width:{importing.total ? (importing.done / importing.total) * 100 : 30}%"></div></div>
    </div>
  {:else if mode === 'modpacks'}
    <div class="lib">
      <Library types={['modpack']} oninstall={(hit) => runImport(() => InstallModpack(hit.project_id))} />
    </div>
    {#if error}<p class="error-text">{error}</p>{/if}
  {:else if mode === 'import'}
    <p class="muted">{t('importText')}</p>
    <button class="primary" onclick={() => runImport(ImportModpack)}>{t('chooseFile')}</button>
    {#if error}<p class="error-text">{error}</p>{/if}
  {:else if loadError}
    <p class="error-text">{loadError}</p>
  {:else}
    <label for="ni-name">{t('name')}</label>
    <input id="ni-name" bind:value={name} placeholder={version || 'My instance'} />

    <label for="ni-ver">{t('mcVersion')}</label>
    <div class="ver">
      <select id="ni-ver" bind:value={version} disabled={!versions.length}>
        {#if !versions.length}<option>{t('loading')}</option>{/if}
        {#each shown as v (v.id)}
          <option value={v.id}>{v.id}{v.type === 'snapshot' ? ` (${t('snapshot')})` : ''}</option>
        {/each}
      </select>
      <label class="check"><input type="checkbox" bind:checked={snapshots} /> {t('showSnapshots')}</label>
    </div>

    <label for="ni-loader">{t('loader')}</label>
    <div class="ver">
      <select id="ni-loader" bind:value={loader}>
        {#each Object.entries(loaders) as [id, l] (id)}<option value={id}>{l.name}</option>{/each}
      </select>
      {#if loader !== 'vanilla'}
        <select bind:value={loaderVersion} disabled={!loaderVersions.length} aria-label={t('loaderVersion', { loader: loaders[loader].name })}>
          {#if loaderLoading}
            <option value="">{t('loading')}</option>
          {:else if !loaderVersions.length}
            <option value="">{t('noLoaderVersions', { version })}</option>
          {/if}
          {#each loaderVersions as lv (lv)}<option value={lv}>{lv}</option>{/each}
        </select>
      {/if}
    </div>

    {#if error}<p class="error-text">{error}</p>{/if}
    <div class="actions">
      <button onclick={onclose}>{t('cancel')}</button>
      <button class="primary" onclick={create} disabled={busy || !version || (loader !== 'vanilla' && !loaderVersion)}>{t('create')}</button>
    </div>
  {/if}
</Modal>

<style>
  .seg { display: flex; gap: 2px; padding: 2px; margin: 8px 0 6px; border-radius: var(--radius); background: var(--bg); }
  .seg button { flex: 1; min-height: 28px; color: var(--muted); }
  .seg button.on { background: var(--surface); color: var(--text); box-shadow: 0 0 0 1px var(--border); }
  .ver { display: flex; gap: 10px; align-items: center; }
  .ver select { flex: 1; }
  .check { display: flex; align-items: center; gap: 6px; margin: 0; white-space: nowrap; color: var(--text); cursor: pointer; }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 20px; }
  .muted { color: var(--muted); margin: 12px 0; }
  .lib { height: min(440px, calc(100vh - 170px)); display: flex; flex-direction: column; margin-top: 8px; }
  .importing { display: grid; gap: 8px; padding: 20px 0 8px; }
  .bar { height: 4px; border-radius: 2px; background: var(--selected); overflow: hidden; }
  .bar div { height: 100%; background: var(--accent); transition: width .2s; }
  .bar.indeterminate div { animation: slide 1.2s ease-in-out infinite; }
  @keyframes slide { from { transform: translateX(-100%); } to { transform: translateX(340%); } }
</style>
