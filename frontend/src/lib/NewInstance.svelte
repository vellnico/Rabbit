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

<Modal title={t('newInstance')} onclose={() => !importing && onclose()} width={mode === 'modpacks' ? 760 : 470}>
  <div class="modes">
    {#each modes as m (m)}
      <button class:on={mode === m} aria-pressed={mode === m} onclick={() => { mode = m; error = '' }} disabled={!!importing}>{t('new.' + m)}</button>
    {/each}
  </div>

  {#if importing}
    <div class="importing">
      <b>{t('import.' + importing.stage)}</b>
      <div class="xp"><div class="fill" class:crawl={!importing.total} style="width:{importing.total ? (importing.done / importing.total) * 100 : 20}%"></div></div>
      {#if importing.total}<span>{importing.done} / {importing.total}</span>{/if}
    </div>
  {:else if mode === 'modpacks'}
    <div class="lib">
      <Library types={['modpack']} oninstall={(hit) => runImport(() => InstallModpack(hit.project_id))} />
    </div>
    {#if error}<p class="error-text">{error}</p>{/if}
  {:else if mode === 'import'}
    <div class="import">
      <p>{t('importText')}</p>
      <button class="primary" onclick={() => runImport(ImportModpack)}>{t('chooseFile')}</button>
      {#if error}<p class="error-text">{error}</p>{/if}
    </div>
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

    <span class="label">{t('loader')}</span>
    <div class="loaders">
      {#each Object.entries(loaders) as [id, l] (id)}
        <button class:on={loader === id} aria-pressed={loader === id} onclick={() => (loader = id)}>
          <span class="block px" style="--lc:{l.color}"></span>{l.name}
        </button>
      {/each}
    </div>

    {#if loader !== 'vanilla'}
      <label for="ni-lv">{t('loaderVersion', { loader: loaders[loader].name })}</label>
      <select id="ni-lv" bind:value={loaderVersion} disabled={!loaderVersions.length}>
        {#if loaderLoading}
          <option value="">{t('loading')}</option>
        {:else if !loaderVersions.length}
          <option value="">{t('noLoaderVersions', { version })}</option>
        {/if}
        {#each loaderVersions as lv (lv)}<option value={lv}>{lv}</option>{/each}
      </select>
    {/if}

    {#if error}<p class="error-text">{error}</p>{/if}
    <div class="actions">
      <button class="flat" onclick={onclose}>{t('cancel')}</button>
      <button class="primary" onclick={create} disabled={busy || !version || (loader !== 'vanilla' && !loaderVersion)}>{t('create')}</button>
    </div>
  {/if}
</Modal>

<style>
  .modes { display: flex; gap: 4px; margin: 6px 0 4px; }
  .modes button { flex: 1; font-size: 15px; }
  .modes button.on, .loaders button.on { background: #6f6f6f; box-shadow: inset 2px 2px 0 #4a4a4a, inset -2px -2px 0 #9a9a9a; outline: 2px solid var(--accent); outline-offset: -4px; }
  .ver { display: flex; gap: 12px; align-items: center; }
  .ver select { flex: 1; }
  .check { display: flex; align-items: center; gap: 6px; margin: 0; white-space: nowrap; cursor: pointer; }
  .label { display: block; font-size: 13px; margin: 12px 0 5px; }
  .loaders { display: grid; grid-template-columns: repeat(5, 1fr); gap: 4px; }
  .loaders button { display: grid; justify-items: center; gap: 6px; padding: 8px 2px; font-size: 13px; }
  .block {
    width: 16px; height: 16px; background: var(--lc);
    box-shadow: inset 2px 2px 0 color-mix(in srgb, var(--lc) 60%, #fff), inset -2px -2px 0 color-mix(in srgb, var(--lc) 60%, #000);
  }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 16px; }
  .lib { height: min(440px, calc(100vh - 170px)); display: flex; flex-direction: column; margin-top: 8px; }
  .import { display: grid; justify-items: start; gap: 12px; padding: 8px 0 4px; }
  .import p { margin: 0; max-width: 380px; }
  .importing { display: grid; gap: 8px; padding: 18px 0 8px; }
  .importing b { font: 17px var(--pixel); font-weight: 400; }
  .importing span { font-size: 13px; font-variant-numeric: tabular-nums; }
  .xp { position: relative; height: 14px; background: #1b1b1b; border: 2px solid #000; overflow: hidden; }
  .xp::after { content: ''; position: absolute; inset: 0; background: repeating-linear-gradient(90deg, transparent 0 calc(10% - 2px), #000 calc(10% - 2px) 10%); }
  .fill { height: 100%; background: var(--accent); transition: width .25s steps(4); }
  .fill.crawl { animation: crawl 1.4s steps(10) infinite; }
  @keyframes crawl { from { transform: translateX(-100%); } to { transform: translateX(500%); } }
</style>
