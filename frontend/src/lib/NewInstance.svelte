<script>
  import Modal from './Modal.svelte'
  import { ListVersions, ListFabricLoaders, CreateInstance } from '../../wailsjs/go/main/App.js'

  let { onclose, oncreated } = $props()

  let versions = $state([])
  let loadError = $state('')
  let snapshots = $state(false)
  let name = $state('')
  let version = $state('')
  let loader = $state('vanilla')
  let loaders = $state([])
  let loaderVersion = $state('')
  let busy = $state(false)
  let error = $state('')

  let shown = $derived(versions.filter((v) => v.type === 'release' || (snapshots && v.type === 'snapshot')))

  ListVersions()
    .then((list) => {
      versions = list
      version = list.find((v) => v.type === 'release')?.id ?? ''
    })
    .catch((e) => (loadError = String(e)))

  $effect(() => {
    if (loader !== 'fabric' || !version) return
    const v = version
    loaders = []
    loaderVersion = ''
    ListFabricLoaders(v)
      .then((list) => {
        if (v !== version) return
        loaders = list ?? []
        loaderVersion = loaders[0] ?? ''
      })
      .catch(() => (loaders = []))
  })

  async function create() {
    busy = true
    error = ''
    try {
      const inst = await CreateInstance(name, version, loader, loaderVersion)
      oncreated(inst)
    } catch (e) {
      error = String(e)
    } finally {
      busy = false
    }
  }
</script>

<Modal title="Новая сборка" {onclose}>
  {#if loadError}
    <p class="error">{loadError}</p>
  {:else}
    <label for="ni-name">Название</label>
    <input id="ni-name" bind:value={name} placeholder={version || 'Моя сборка'} />

    <label for="ni-ver">Версия Minecraft</label>
    <select id="ni-ver" bind:value={version} disabled={!versions.length}>
      {#if !versions.length}<option>Загрузка…</option>{/if}
      {#each shown as v (v.id)}
        <option value={v.id}>{v.id}{v.type === 'snapshot' ? ' (снапшот)' : ''}</option>
      {/each}
    </select>
    <label class="check"><input type="checkbox" bind:checked={snapshots} /> Показывать снапшоты</label>

    <label for="ni-loader">Загрузчик</label>
    <div class="seg" id="ni-loader">
      <button class:on={loader === 'vanilla'} onclick={() => (loader = 'vanilla')}>Vanilla</button>
      <button class:on={loader === 'fabric'} onclick={() => (loader = 'fabric')}>Fabric</button>
    </div>
    {#if loader === 'fabric'}
      <label for="ni-fl">Версия Fabric Loader</label>
      <select id="ni-fl" bind:value={loaderVersion} disabled={!loaders.length}>
        {#if !loaders.length}<option value="">Нет версий для {version}</option>{/if}
        {#each loaders as l (l)}<option value={l}>{l}</option>{/each}
      </select>
    {/if}

    {#if error}<p class="error">{error}</p>{/if}
    <div class="actions">
      <button class="ghost" onclick={onclose}>Отмена</button>
      <button class="primary" onclick={create} disabled={busy || !version || (loader === 'fabric' && !loaderVersion)}>Создать</button>
    </div>
  {/if}
</Modal>

<style>
  .check { display: flex; align-items: center; gap: 8px; cursor: pointer; }
  .check input { width: auto; margin: 0; }
  .seg { display: flex; gap: 6px; }
  .seg button { flex: 1; }
  .seg .on { border-color: var(--accent); color: var(--accent); }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 18px; }
  .error { color: var(--danger); font-size: 13px; margin: 12px 0 0; user-select: text; }
</style>
