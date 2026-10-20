<script>
  import { slide } from 'svelte/transition'
  import Modal from './Modal.svelte'
  import { t, tErr } from './i18n.svelte.js'
  import { loaders } from './theme.js'
  import { ListVersions, ListLoaderVersions, CreateInstance } from '../../wailsjs/go/main/App.js'

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

<Modal title={t('newInstance')} {onclose} width={460}>
  {#if loadError}
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
