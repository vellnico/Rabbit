<script>
  import { onMount } from 'svelte'
  import Avatar from './lib/Avatar.svelte'
  import Logo from './lib/Logo.svelte'
  import NewInstance from './lib/NewInstance.svelte'
  import Accounts from './lib/Accounts.svelte'
  import Settings from './lib/Settings.svelte'
  import Library from './lib/Library.svelte'
  import Installed from './lib/Installed.svelte'
  import { t, tErr, setLanguage, formatDate } from './lib/i18n.svelte.js'
  import { applyAccent, loaders } from './lib/theme.js'
  import { GetState, Launch, Stop, SelectInstance, DeleteInstance, OpenInstanceFolder, ExportModpack, ShowInFolder } from '../wailsjs/go/main/App.js'
  import { EventsOn } from '../wailsjs/runtime/runtime.js'

  let app = $state(null)
  let modal = $state('') // '', 'new', 'accounts', 'settings'
  let progress = $state({}) // instance id -> {stage, done, total}
  let status = $state({}) // instance id -> {state, message, log}
  let confirmDelete = $state(false)
  let tab = $state('overview') // overview | content | library
  let contentVersion = $state(0) // bumped when content changes
  let exporting = $state(false)
  let exported = $state('') // path of the last export
  let exportError = $state('')

  let selected = $derived(app?.instances?.find((i) => i.id === app.settings.selectedInstance) ?? app?.instances?.[0])
  let account = $derived(app?.accounts?.find((a) => a.id === app.settings.selectedAccount))
  let st = $derived(selected ? status[selected.id] : null)
  let pr = $derived(selected ? progress[selected.id] : null)
  let busy = $derived(st?.state === 'preparing' || st?.state === 'running')

  async function refresh() {
    app = await GetState()
    for (const id of app.running ?? []) status[id] ??= { state: 'running' }
  }

  onMount(() => {
    refresh().then(() => {
      applyAccent(app.settings.accent)
      setLanguage(app.settings.language)
    })
    const off1 = EventsOn('progress', (e) => (progress[e.instance] = e))
    const off2 = EventsOn('state', (e) => {
      status[e.instance] = e
      if (e.state !== 'preparing') delete progress[e.instance]
      if (e.state === 'running' || e.state === 'stopped') refresh()
    })
    return () => { off1(); off2() }
  })

  function pick(id) {
    app.settings.selectedInstance = id
    confirmDelete = false
    exported = exportError = ''
    SelectInstance(id)
  }

  async function play() {
    if (!account) {
      modal = 'accounts'
      return
    }
    status[selected.id] = { state: 'preparing' }
    try {
      await Launch(selected.id)
    } catch (e) {
      status[selected.id] = { state: 'error', message: String(e) }
    }
  }

  async function remove() {
    if (!confirmDelete) {
      confirmDelete = true
      return
    }
    confirmDelete = false
    try {
      await DeleteInstance(selected.id)
    } catch (e) {
      status[selected.id] = { state: 'error', message: String(e) }
    }
    await refresh()
  }

  async function exportPack() {
    exporting = true
    exported = exportError = ''
    try {
      exported = await ExportModpack(selected.id)
    } catch (e) {
      exportError = tErr(e)
    } finally {
      exporting = false
    }
  }

  const tabs = ['overview', 'content', 'library']
  const folders = ['mods', 'resourcepacks', 'shaderpacks', 'saves', 'config', '']
  const libraryTypes = (inst) => (inst.loader === 'vanilla' ? ['resourcepack', 'shader'] : ['mod', 'resourcepack', 'shader'])
  const percent = (p) => (p?.total ? Math.round((p.done / p.total) * 100) : null)
  const loaderLabel = (inst) => {
    const name = (loaders[inst.loader] ?? loaders.vanilla).name
    return inst.loaderVersion ? `${name} ${inst.loaderVersion}` : name
  }
</script>

{#if app}
  <div class="layout">
    <aside>
      <div class="brand"><Logo size={20} /><span>Rabbit</span></div>

      <div class="section">
        <span>{t('instances')}</span>
        <button class="plain icon" onclick={() => (modal = 'new')} title={t('newInstance')} aria-label={t('newInstance')}>
          <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true"><path d="M8 3v10M3 8h10" stroke="currentColor" stroke-width="1.5" /></svg>
        </button>
      </div>

      <nav>
        {#each app.instances ?? [] as inst (inst.id)}
          <button class="plain inst" class:sel={selected?.id === inst.id} onclick={() => pick(inst.id)}>
            <span class="name">{inst.name}</span>
            <span class="meta">{inst.gameVersion}, {(loaders[inst.loader] ?? loaders.vanilla).name}</span>
            {#if status[inst.id]?.state === 'running'}<span class="live" title={t('running')}></span>{/if}
          </button>
        {/each}
      </nav>

      <div class="who">
        <button class="plain account" onclick={() => (modal = 'accounts')}>
          <Avatar {account} size={28} />
          <span>
            <b>{account?.name ?? t('signIn')}</b>
            <small>{account ? (account.kind === 'elyby' ? 'Ely.by' : t('offline')) : t('signInSub')}</small>
          </span>
        </button>
        <button class="plain icon" onclick={() => (modal = 'settings')} aria-label={t('settings')} title={t('settings')}>
          <svg viewBox="0 0 16 16" width="16" height="16" aria-hidden="true"><circle cx="8" cy="8" r="2.2" fill="none" stroke="currentColor" stroke-width="1.2" /><path fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round" d="M6.9 1.8h2.2l.4 1.7 1.2.7 1.7-.5 1.1 1.9-1.3 1.2v1.4l1.3 1.2-1.1 1.9-1.7-.5-1.2.7-.4 1.7H6.9l-.4-1.7-1.2-.7-1.7.5-1.1-1.9 1.3-1.2V7.3L2.5 6.1l1.1-1.9 1.7.5 1.2-.7z" /></svg>
        </button>
      </div>
    </aside>

    <main>
      {#if selected}
        <header>
          <div class="title">
            <h1>{selected.name}</h1>
            <p>
              {#if st?.state === 'preparing'}
                {pr?.stage ? t('stage.' + pr.stage) : t('preparing')}{#if percent(pr) !== null}, {percent(pr)}%{/if}
              {:else if st?.state === 'running'}
                {t('running')}
              {:else}
                Minecraft {selected.gameVersion}, {loaderLabel(selected)}
              {/if}
            </p>
          </div>
          {#if busy}
            <button class="play" onclick={() => Stop(selected.id)}>{st.state === 'running' ? t('closeGame') : t('cancel')}</button>
          {:else}
            <button class="primary play" onclick={play}>{t('play')}</button>
          {/if}
          {#if st?.state === 'preparing'}
            <div class="progress" class:indeterminate={percent(pr) === null}><div style="width:{percent(pr) ?? 30}%"></div></div>
          {/if}
        </header>

        <div class="tabs" role="tablist">
          {#each tabs as id (id)}
            <button class="plain" role="tab" aria-selected={tab === id} class:on={tab === id} onclick={() => (tab = id)}>{t('tab.' + id)}</button>
          {/each}
        </div>

        {#if st?.state === 'error' && st.message}
          <div class="failure">
            <b>{tErr(st.message)}</b>
            {#if st.log?.length}<pre>{st.log.join('\n')}</pre>{/if}
          </div>
        {/if}

        <section class="body">
          {#if tab === 'overview'}
            <p class="muted">
              {selected.lastPlayed && formatDate(selected.lastPlayed) ? t('lastPlayed', { date: formatDate(selected.lastPlayed) }) : t('neverPlayed')}
            </p>

            <h2>{t('openFolder')}</h2>
            <div class="row">
              {#each folders as dir (dir)}
                {#if dir !== 'mods' || selected.loader !== 'vanilla'}
                  <button onclick={() => OpenInstanceFolder(selected.id, dir)}>{t('folder.' + (dir || 'root'))}</button>
                {/if}
              {/each}
            </div>

            <h2>Modrinth</h2>
            <div class="row">
              <button onclick={exportPack} disabled={exporting}>{exporting ? t('exporting') : t('exportPack')}</button>
              {#if exported}
                <span>{t('exported')}</span>
                <button class="plain" onclick={() => ShowInFolder(exported)}>{t('showInFolder')}</button>
              {/if}
            </div>
            {#if exportError}<p class="error-text">{exportError}</p>{/if}

            <div class="danger-zone">
              <button class="plain danger" onclick={remove} onblur={() => (confirmDelete = false)}>
                {confirmDelete ? t('confirmDelete') : t('deleteInstance')}
              </button>
            </div>
          {:else if tab === 'content'}
            <Installed instance={selected} version={contentVersion} onlibrary={() => (tab = 'library')} />
          {:else}
            {#key selected.id}
              <Library instance={selected} types={libraryTypes(selected)} onchanged={() => contentVersion++} />
            {/key}
          {/if}
        </section>
      {:else}
        <div class="welcome">
          <h1>{t('welcome')}</h1>
          <p class="muted">{t('welcomeSub')}</p>
          <button class="primary" onclick={() => (modal = 'new')}>{t('newInstance')}</button>
        </div>
      {/if}
    </main>
  </div>

  {#if modal === 'new'}
    <NewInstance onclose={() => (modal = '')} oncreated={async () => { modal = ''; tab = 'overview'; await refresh() }} />
  {:else if modal === 'accounts'}
    <Accounts {app} onclose={() => (modal = '')} onchange={refresh} />
  {:else if modal === 'settings'}
    <Settings {app} onclose={() => (modal = '')} onchange={refresh} />
  {/if}
{/if}

<style>
  .layout { display: grid; grid-template-columns: 230px 1fr; height: 100%; }

  /* ---- sidebar ---- */
  aside { display: flex; flex-direction: column; min-height: 0; padding: 12px 8px 8px; }
  .brand { display: flex; align-items: center; gap: 8px; padding: 0 8px 14px; font-size: 14px; font-weight: 600; }
  .section { display: flex; align-items: center; justify-content: space-between; padding: 0 2px 4px 8px; color: var(--muted); }
  .icon { min-height: 28px; width: 28px; padding: 0; display: grid; place-items: center; color: var(--muted); }
  .icon:hover:not(:disabled) { color: var(--text); }
  nav { flex: 1; overflow: auto; display: grid; align-content: start; gap: 1px; min-height: 0; }
  .inst { position: relative; display: grid; text-align: left; padding: 6px 8px 6px 12px; border-radius: var(--radius); }
  .inst.sel { background: var(--selected); }
  .inst.sel::before { content: ''; position: absolute; left: 2px; top: 30%; bottom: 30%; width: 3px; border-radius: 2px; background: var(--text); }
  .name { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .meta { font-size: 12px; color: var(--muted); }
  .live { position: absolute; right: 10px; top: 50%; width: 7px; height: 7px; margin-top: -3px; border-radius: 50%; background: #2fa84f; }
  .who { display: flex; align-items: center; gap: 2px; border-top: 1px solid var(--border); padding-top: 8px; }
  .account { flex: 1; display: flex; align-items: center; gap: 8px; text-align: left; padding: 4px 6px; min-width: 0; }
  .account span { display: grid; min-width: 0; }
  .account b { font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .account small { font-size: 12px; color: var(--muted); }

  /* ---- main: a raised layer, like Windows 11 apps ---- */
  main {
    display: flex; flex-direction: column; min-width: 0; min-height: 0;
    background: var(--surface); border-left: 1px solid var(--border); border-top: 1px solid var(--border);
    border-top-left-radius: 8px; margin-top: 8px;
  }
  header { position: relative; display: flex; align-items: center; gap: 16px; padding: 18px 24px 14px; }
  .title { flex: 1; min-width: 0; }
  h1 { margin: 0; font-size: 20px; font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .title p { margin: 2px 0 0; color: var(--muted); }
  .play { min-width: 128px; min-height: 36px; font-size: 14px; }
  .progress { position: absolute; left: 0; right: 0; bottom: 0; height: 2px; overflow: hidden; }
  .progress div { height: 100%; background: var(--accent); transition: width .2s; }
  .progress.indeterminate div { animation: slide 1.2s ease-in-out infinite; }
  @keyframes slide { from { transform: translateX(-100%); } to { transform: translateX(340%); } }

  .tabs { display: flex; gap: 4px; padding: 0 20px; border-bottom: 1px solid var(--border); }
  .tabs button { position: relative; border-radius: var(--radius) var(--radius) 0 0; color: var(--muted); padding: 6px 10px 8px; }
  .tabs button.on { color: var(--text); font-weight: 600; }
  .tabs button.on::after { content: ''; position: absolute; left: 10px; right: 10px; bottom: -1px; height: 2px; background: var(--text); }

  .failure { margin: 12px 24px 0; padding: 10px 12px; border: 1px solid color-mix(in srgb, var(--danger) 45%, transparent); border-radius: var(--radius); color: var(--danger); user-select: text; }
  .failure b { font-weight: 600; white-space: pre-wrap; }
  .failure pre { color: var(--muted); font: 11px/1.4 Consolas, monospace; overflow: auto; margin: 8px 0 0; max-height: 140px; white-space: pre-wrap; }

  .body { flex: 1; min-height: 0; display: flex; flex-direction: column; padding: 16px 24px 20px; overflow: auto; }
  .muted { color: var(--muted); margin: 0; }
  h2 { margin: 20px 0 8px; font-size: 13px; font-weight: 600; }
  .row { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; }
  .danger-zone { margin-top: auto; padding-top: 24px; }
  .danger-zone button { padding-left: 0; }
  .danger-zone button:hover:not(:disabled) { background: none; text-decoration: underline; }

  .welcome { margin: auto; display: grid; justify-items: center; gap: 8px; text-align: center; }
  .welcome h1 { font-size: 18px; }
  .welcome button { margin-top: 8px; }
</style>
