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
  let loader = $derived(loaders[selected?.loader] ?? loaders.vanilla)

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
  const stageText = (p) => (p?.stage ? t('stage.' + p.stage) : t('preparing'))
  const loaderOf = (inst) => loaders[inst.loader] ?? loaders.vanilla
</script>

{#if app}
  <div class="layout">
    <aside class="panel side">
      <div class="brand"><Logo size={28} /><span>Rabbit</span></div>

      <nav class="slot list">
        {#each app.instances ?? [] as inst (inst.id)}
          <button class="inst" class:sel={selected?.id === inst.id} onclick={() => pick(inst.id)}>
            <span class="block px" style="--lc:{loaderOf(inst).color}"></span>
            <span class="text">
              <span class="name">{inst.name}</span>
              <span class="meta">{inst.gameVersion} {loaderOf(inst).name}</span>
            </span>
            {#if status[inst.id]?.state === 'running'}<span class="live" title={t('running')}></span>{/if}
          </button>
        {:else}
          <p class="empty">{t('noInstances')}</p>
        {/each}
      </nav>

      <button class="new" onclick={() => (modal = 'new')}>{t('newInstance')}</button>

      <div class="who">
        <button class="account flat" onclick={() => (modal = 'accounts')}>
          <Avatar {account} size={36} />
          <span>
            <b>{account?.name ?? t('signIn')}</b>
            <small>{account ? (account.kind === 'elyby' ? 'Ely.by' : t('offline')) : t('signInSub')}</small>
          </span>
        </button>
        <button class="gear" onclick={() => (modal = 'settings')} aria-label={t('settings')} title={t('settings')}>
          <svg class="px" viewBox="0 0 8 8" width="16" height="16" shape-rendering="crispEdges" aria-hidden="true"><path fill="currentColor" d="M3 0h2v1h1v1h1v1H6v2h1v1H6v1H5v1H3V7H2V6H1V5h1V3H1V2h1V1h1zM3 3v2h2V3z" /></svg>
        </button>
      </div>
    </aside>

    <main>
      {#if selected}
        <div class="tabs" role="tablist">
          {#each tabs as id (id)}
            <button role="tab" aria-selected={tab === id} class:on={tab === id} onclick={() => (tab = id)}>{t('tab.' + id)}</button>
          {/each}
        </div>

        <section class="panel sheet">
          <header class="head">
            <h1>{selected.name}</h1>
            <p class="facts">
              <span>Minecraft {selected.gameVersion}</span>
              <span style="color:{loader.color}">{loader.name}{selected.loaderVersion ? ' ' + selected.loaderVersion : ''}</span>
              <span class="muted">
                {selected.lastPlayed && formatDate(selected.lastPlayed) ? t('lastPlayed', { date: formatDate(selected.lastPlayed) }) : t('neverPlayed')}
              </span>
            </p>
          </header>

          {#if tab === 'overview'}
            <div class="overview">
              <div class="folders">
                {#each folders as dir (dir)}
                  {#if dir !== 'mods' || selected.loader !== 'vanilla'}
                    <button onclick={() => OpenInstanceFolder(selected.id, dir)}>{t('folder.' + (dir || 'root'))}</button>
                  {/if}
                {/each}
              </div>
              <div class="row">
                <button onclick={exportPack} disabled={exporting}>{exporting ? t('exporting') : t('exportPack')}</button>
                {#if exported}
                  <span class="ok">{t('exported')}</span>
                  <button class="flat" onclick={() => ShowInFolder(exported)}>{t('showInFolder')}</button>
                {/if}
                {#if exportError}<span class="error-text inline">{exportError}</span>{/if}
                <button class="flat danger push" onclick={remove} onblur={() => (confirmDelete = false)}>
                  {confirmDelete ? t('confirmDelete') : t('deleteInstance')}
                </button>
              </div>
            </div>
          {:else if tab === 'content'}
            <Installed instance={selected} version={contentVersion} onlibrary={() => (tab = 'library')} />
          {:else}
            {#key selected.id}
              <Library instance={selected} types={libraryTypes(selected)} onchanged={() => contentVersion++} />
            {/key}
          {/if}

          {#if st?.state === 'error' && st.message}
            <div class="failure">
              <b>{tErr(st.message)}</b>
              {#if st.log?.length}<pre>{st.log.join('\n')}</pre>{/if}
            </div>
          {/if}
        </section>

        <footer>
          <div class="status">
            <span class="label">
              {#if st?.state === 'preparing'}
                {stageText(pr)}{#if percent(pr) !== null}&nbsp; {pr.done}/{pr.total}{/if}
              {:else if st?.state === 'running'}
                {t('running')}
              {:else}
                &nbsp;
              {/if}
            </span>
            <div class="xp" class:idle={!busy}>
              {#if st?.state === 'preparing'}
                <div class="fill" class:crawl={percent(pr) === null} style="width:{percent(pr) ?? 20}%"></div>
              {:else if st?.state === 'running'}
                <div class="fill" style="width:100%"></div>
              {/if}
            </div>
          </div>
          {#if busy}
            <button class="play" onclick={() => Stop(selected.id)}>{st.state === 'running' ? t('closeGame') : t('cancel')}</button>
          {:else}
            <button class="play primary" onclick={play}>{t('play')}</button>
          {/if}
        </footer>
      {:else}
        <section class="panel welcome">
          <Logo size={72} />
          <h1>{t('welcome')}</h1>
          <p>{t('welcomeSub')}</p>
          <button class="primary" onclick={() => (modal = 'new')}>{t('newInstance')}</button>
        </section>
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
  .layout { display: grid; grid-template-columns: 248px 1fr; gap: 12px; height: 100%; padding: 12px; }

  /* ---- sidebar ---- */
  .side { display: flex; flex-direction: column; padding: 10px; min-height: 0; }
  .brand { display: flex; align-items: center; gap: 10px; font: 22px var(--pixel); color: var(--ink); padding: 0 2px 10px; }
  .list { flex: 1; overflow: auto; padding: 4px; display: grid; align-content: start; gap: 2px; min-height: 0; }
  .inst {
    display: flex; align-items: center; gap: 8px; text-align: left; width: 100%;
    padding: 6px; background: none; border: 2px solid transparent; box-shadow: none; text-shadow: none;
    font: inherit; color: #fff;
  }
  .inst:hover:not(:disabled) { background: rgb(255 255 255 / .18); outline: none; }
  .inst:active:not(:disabled) { padding: 6px; }
  .inst.sel { background: rgb(255 255 255 / .28); border-color: #fff; }
  /* A tiny block coloured by the loader, like an item in a slot. */
  .block {
    flex: none; width: 24px; height: 24px; background: var(--lc);
    box-shadow: inset 2px 2px 0 color-mix(in srgb, var(--lc) 60%, #fff), inset -2px -2px 0 color-mix(in srgb, var(--lc) 60%, #000);
  }
  .text { display: grid; min-width: 0; }
  .name { font: 15px var(--pixel); text-shadow: 2px 2px 0 #3f3f3f; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .meta { font-size: 12px; color: #e8e8e8; }
  .live { margin-left: auto; flex: none; width: 8px; height: 8px; background: #55ff55; box-shadow: 0 0 0 2px #1f5f1f; }
  .empty { color: #fff; text-shadow: 1px 1px 0 #3f3f3f; padding: 8px; margin: 0; font-size: 13px; }
  .new { margin: 10px 0; }
  .who { display: flex; align-items: center; gap: 6px; }
  .account { flex: 1; display: flex; align-items: center; gap: 10px; text-align: left; padding: 2px; min-width: 0; font: inherit; }
  .account:active:not(:disabled) { padding: 2px; }
  .account span { display: grid; min-width: 0; }
  .account b { font: 15px var(--pixel); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .account small { font-size: 12px; color: #5a5a5a; }
  .gear { padding: 8px; display: grid; }

  /* ---- main ---- */
  main { display: flex; flex-direction: column; min-width: 0; min-height: 0; }

  /* Creative-inventory tabs sitting on top of the sheet. */
  .tabs { display: flex; gap: 4px; padding-left: 8px; position: relative; z-index: 1; margin-bottom: -2px; }
  .tabs button {
    font-size: 15px; padding: 8px 16px 7px; color: #e8e8e8;
    background: #8b8b8b; border-bottom: none;
    box-shadow: inset 2px 2px 0 #a8a8a8, inset -2px 0 0 #5a5a5a;
  }
  .tabs button.on {
    background: var(--stone); color: var(--ink); text-shadow: none; padding-bottom: 9px;
    box-shadow: inset 2px 2px 0 var(--hi), inset -2px 0 0 var(--lo);
  }
  .tabs button:hover:not(:disabled) { outline-offset: -4px; }
  .tabs button.on:hover:not(:disabled) { outline: none; background: var(--stone); }

  .sheet { flex: 1; min-height: 0; display: flex; flex-direction: column; padding: 14px 16px 16px; }
  .head h1 { margin: 0; font: 30px/1.1 var(--pixel); color: var(--ink); overflow-wrap: anywhere; }
  .facts { display: flex; flex-wrap: wrap; gap: 4px 16px; margin: 6px 0 14px; font-size: 13px; color: var(--ink); }
  .facts span:nth-child(2) { font-weight: 600; filter: brightness(.8); }
  .muted { color: #626262; }

  .overview { display: grid; gap: 14px; align-content: start; }
  .folders { display: grid; grid-template-columns: repeat(auto-fill, minmax(176px, 1fr)); gap: 6px; }
  .folders button { white-space: nowrap; }
  .row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
  .ok { color: #2f6b1f; font-size: 13px; font-weight: 600; }
  .inline { margin: 0; }
  .push { margin-left: auto; }

  .failure { margin-top: 12px; padding: 10px 12px; background: #2a1414; border: 2px solid #000; color: #ff7f7f; user-select: text; min-height: 0; display: flex; flex-direction: column; }
  .failure b { font-weight: 600; white-space: pre-wrap; }
  .failure pre { color: #e0b4b4; font: 11px/1.4 Consolas, monospace; overflow: auto; margin: 8px 0 0; max-height: 150px; white-space: pre-wrap; }

  /* ---- footer: experience bar + Play ---- */
  footer { display: flex; align-items: flex-end; gap: 16px; padding-top: 12px; }
  .status { flex: 1; min-width: 0; display: grid; gap: 6px; }
  .label { font: 15px var(--pixel); color: #fff; text-shadow: 2px 2px 0 #000; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .xp { position: relative; height: 14px; background: #1b1b1b; border: 2px solid #000; overflow: hidden; }
  .xp.idle { opacity: .55; }
  /* Segment dividers drawn over the fill, like the experience bar. */
  .xp::after {
    content: ''; position: absolute; inset: 0; pointer-events: none;
    background: repeating-linear-gradient(90deg, transparent 0 calc(10% - 2px), #000 calc(10% - 2px) 10%);
  }
  .fill {
    height: 100%; background: var(--accent);
    box-shadow: inset 0 2px 0 color-mix(in srgb, var(--accent) 50%, #fff), inset 0 -2px 0 color-mix(in srgb, var(--accent) 60%, #000);
    transition: width .25s steps(4);
  }
  .fill.crawl { animation: crawl 1.4s steps(10) infinite; }
  @keyframes crawl { from { transform: translateX(-100%); } to { transform: translateX(500%); } }

  .play { min-width: 210px; padding: 16px 24px 14px; font-size: 24px; }
  .play:active:not(:disabled) { padding: 17px 24px 13px; }

  .welcome { margin: auto; padding: 28px 36px; display: grid; justify-items: center; gap: 10px; text-align: center; max-width: 440px; }
  .welcome h1 { margin: 6px 0 0; font: 26px var(--pixel); }
  .welcome p { margin: 0 0 8px; }
</style>
