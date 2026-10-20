<script>
  import { onMount } from 'svelte'
  import { fade, slide } from 'svelte/transition'
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

<Modal title={t('newInstance')} onclose={() => !importing && onclose()} width={mode === 'modpacks' ? 720 : 460}>
  <div class="modes" style="--i:{modes.indexOf(mode)}">
    <span class="pill"></span>
    {#each modes as m (m)}
      <button class:on={mode === m} onclick={() => { mode = m; error = '' }} disabled={!!importing}>{t('new.' + m)}</button>
    {/each}
  </div>

  {#if importing}
    <div class="importing" in:fade>
      <i class="spinner"></i>
      <b>{t('import.' + importing.stage)}</b>
      {#if importing.total}
        <span>{importing.done} / {importing.total}</span>
        <div class="bar"><div style="width:{(importing.done / importing.total) * 100}%"></div></div>
      {/if}
    </div>
  {:else if mode === 'modpacks'}
    <div class="lib" in:fade={{ duration: 150 }}>
      <Library types={['modpack']} oninstall={(hit) => runImport(() => InstallModpack(hit.project_id))} />
    </div>
    {#if error}<p class="error-text">{error}</p>{/if}
  {:else if mode === 'import'}
    <div class="import" in:fade={{ duration: 150 }}>
      <svg viewBox="0 0 24 24" width="34" height="34" aria-hidden="true"><path d="M12 3v12m0 0-4-4m4 4 4-4M5 15v4h14v-4" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" /></svg>
      <b>{t('importTitle')}</b>
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
    <select id="ni-ver" bind:value={version} disabled={!versions.length}>
      {#if !versions.length}<option>{t('loading')}</option>{/if}
      {#each shown as v (v.id)}
        <option value={v.id}>{v.id}{v.type === 'snapshot' ? ` (${t('snapshot')})` : ''}</option>
      {/each}
    </select>
    <label class="check"><input type="checkbox" bind:checked={snapshots} /> {t('showSnapshots')}</label>

    <span class="label">{t('loader')}</span>
    <div class="loaders">
      {#each Object.entries(loaders) as [id, l] (id)}
        <button class="loader" class:on={loader === id} style="--c:{l.color}" onclick={() => (loader = id)}>
          <i></i>{l.name}
        </button>
      {/each}
    </div>

    {#if loader !== 'vanilla'}
      <div transition:slide={{ duration: 200 }}>
        <label for="ni-lv">{t('loaderVersion', { loader: loaders[loader].name })}</label>
        <select id="ni-lv" bind:value={loaderVersion} disabled={!loaderVersions.length}>
          {#if loaderLoading}
            <option value="">{t('loading')}</option>
          {:else if !loaderVersions.length}
            <option value="">{t('noLoaderVersions', { version })}</option>
          {/if}
          {#each loaderVersions as lv (lv)}<option value={lv}>{lv}</option>{/each}
        </select>
      </div>
    {/if}

    {#if error}<p class="error-text" transition:slide>{error}</p>{/if}
    <div class="actions">
      <button class="ghost" onclick={onclose}>{t('cancel')}</button>
      <button class="primary" onclick={create} disabled={busy || !version || (loader !== 'vanilla' && !loaderVersion)}>{t('create')}</button>
    </div>
  {/if}
</Modal>

<style>
  .modes { position: relative; display: grid; grid-template-columns: repeat(3, 1fr); margin: 14px 0 4px; background: var(--bg); padding: 3px; border-radius: 10px; border: 1px solid var(--line); }
  .pill { position: absolute; top: 3px; bottom: 3px; left: 3px; width: calc((100% - 6px) / 3); transform: translateX(calc(var(--i) * 100%)); background: var(--raised-2); border-radius: 8px; transition: transform .3s var(--ease); }
  .modes button { position: relative; border: none; background: transparent; color: var(--muted); padding: 6px; }
  .modes button:hover:not(:disabled) { background: transparent; color: var(--text); }
  .modes button.on { color: var(--text); }
  .lib { height: min(440px, calc(100vh - 190px)); display: flex; flex-direction: column; margin-top: 12px; }
  .import { display: grid; justify-items: center; text-align: center; gap: 6px; padding: 26px 10px 8px; }
  .import svg { color: var(--accent); margin-bottom: 4px; }
  .import p { margin: 0 0 12px; color: var(--muted); font-size: 13px; max-width: 320px; }
  .importing { display: grid; justify-items: center; gap: 8px; padding: 36px 10px 20px; text-align: center; }
  .importing span { color: var(--muted); font-size: 12.5px; font-variant-numeric: tabular-nums; }
  .spinner { width: 34px; height: 34px; border-radius: 50%; border: 3px solid rgb(var(--accent-rgb) / .25); border-top-color: var(--accent); animation: spin .8s linear infinite; margin-bottom: 6px; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .bar { width: 100%; max-width: 320px; height: 6px; background: var(--raised-2); border-radius: 3px; overflow: hidden; }
  .bar div { height: 100%; background: var(--accent); transition: width .2s; box-shadow: 0 0 10px rgb(var(--accent-rgb) / .6); }
  .check { display: flex; align-items: center; gap: 8px; cursor: pointer; margin-top: 10px; }
  .check input { width: auto; margin: 0; }
  .label { display: block; color: var(--muted); font-size: 12px; margin: 14px 0 6px; }
  .loaders { display: grid; grid-template-columns: repeat(5, 1fr); gap: 6px; }
  .loader { display: grid; justify-items: center; gap: 6px; padding: 10px 4px 8px; font-size: 12.5px; color: var(--muted); }
  .loader i { width: 10px; height: 10px; border-radius: 50%; background: var(--c); opacity: .55; transition: transform .25s var(--ease), opacity .2s, box-shadow .25s; }
  .loader.on { color: var(--text); border-color: color-mix(in srgb, var(--c) 70%, transparent); background: color-mix(in srgb, var(--c) 10%, var(--raised)); }
  .loader.on i { opacity: 1; transform: scale(1.25); box-shadow: 0 0 12px var(--c); }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 20px; }
</style>
