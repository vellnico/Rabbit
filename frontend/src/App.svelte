<script>
  import { onMount } from 'svelte'
  import Titlebar from './lib/Titlebar.svelte'
  import Avatar from './lib/Avatar.svelte'
  import Icon from './lib/Icon.svelte'
  import NewInstance from './lib/NewInstance.svelte'
  import Instances from './lib/Instances.svelte'
  import Accounts from './lib/Accounts.svelte'
  import Settings from './lib/Settings.svelte'
  import Library from './lib/Library.svelte'
  import Installed from './lib/Installed.svelte'
  import { t, tErr, setLanguage, formatDate, accountKind } from './lib/i18n.svelte.js'
  import { applyAccent, loaders } from './lib/theme.js'
  import { GetState, Launch, Stop, SelectInstance, DeleteInstance, OpenInstanceFolder, ExportModpack, ShowInFolder } from '../wailsjs/go/main/App.js'
  import { EventsOn } from '../wailsjs/runtime/runtime.js'

  let app = $state(null)
  let view = $state('home') // home | instances | library | settings
  let modal = $state('') // '', 'new', 'accounts'
  let progress = $state({}) // instance id -> {stage, done, total}
  let status = $state({}) // instance id -> {state, message, log}
  let confirmDelete = $state(false)
  let tab = $state('overview') // overview | content
  let picker = $state(false) // instance menu next to Play
  let contentVersion = $state(0) // bumped when content changes
  let exporting = $state(false)
  let exported = $state('') // path of the last export
  let exportError = $state('')

  let selected = $derived(app?.instances?.find((i) => i.id === app.settings.selectedInstance) ?? app?.instances?.[0])
  let account = $derived(app?.accounts?.find((a) => a.id === app.settings.selectedAccount))
  let st = $derived(selected ? status[selected.id] : null)
  let pr = $derived(selected ? progress[selected.id] : null)

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
    confirmDelete = picker = false
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

  const nav = [
    ['home', 'house'],
    ['instances', 'layout-grid'],
    ['library', 'compass'],
  ]
  const navLabel = (id) => (id === 'library' ? t('tab.library') : t(id))
  // Folder, icon.
  const folders = [
    ['mods', 'blocks'],
    ['resourcepacks', 'image'],
    ['shaderpacks', 'sun'],
    ['saves', 'globe'],
    ['config', 'sliders-horizontal'],
    ['', 'folder-open'],
  ]
  const libraryTypes = (inst) => (inst.loader === 'vanilla' ? ['resourcepack', 'shader'] : ['mod', 'resourcepack', 'shader'])
  const percent = (p) => (p?.total ? Math.round((p.done / p.total) * 100) : null)
  const loaderOf = (inst) => loaders[inst.loader] ?? loaders.vanilla
  const loaderLabel = (inst) => (inst.loaderVersion ? `${loaderOf(inst).name} ${inst.loaderVersion}` : loaderOf(inst).name)
</script>

<svelte:window onclick={() => (picker = false)} oncontextmenu={(e) => !e.target.closest('input, textarea, .selectable') && e.preventDefault()} />

{#if app}
  <div class="window">
    <Titlebar />
    <div class="shell">
      <nav class="rail">
        <div class="group">
          {#each nav as [id, icon] (id)}
            <button class="item" class:on={view === id} onclick={() => (view = id)} aria-label={navLabel(id)} aria-current={view === id ? 'page' : undefined}>
              <Icon name={icon} size={20} />
              <span class="tip">{navLabel(id)}</span>
            </button>
          {/each}
        </div>
        <div class="group">
          <button class="item" class:on={view === 'settings'} onclick={() => (view = 'settings')} aria-label={t('settings')}>
            <Icon name="settings" size={20} />
            <span class="tip">{t('settings')}</span>
          </button>
          <button class="item" onclick={() => (modal = 'accounts')} aria-label={t('accounts')}>
            <Avatar {account} size={28} />
            <span class="tip">{account ? `${account.name} · ${accountKind(account)}` : t('signIn')}</span>
          </button>
        </div>
      </nav>

      <main class="content">
        {#key view}
          <div class="screen">
            {#if view === 'home'}
              {#if selected}
                <section class="hero">
                  <div class="info">
                    <span class="overline">{loaderLabel(selected)} · {selected.gameVersion}</span>
                    <h1>{selected.name}</h1>
                    <p class="meta">
                      {selected.lastPlayed && formatDate(selected.lastPlayed) ? t('lastPlayed', { date: formatDate(selected.lastPlayed) }) : t('neverPlayed')}
                    </p>
                  </div>

                  <div class="launch">
                    {#if st?.state === 'preparing'}
                      <div class="row">
                        <button class="subtle" onclick={() => Stop(selected.id)}>{t('cancel')}</button>
                        <div class="play loading" role="progressbar" aria-valuenow={percent(pr)} aria-valuemin="0" aria-valuemax="100">
                          <div class="fill" class:indeterminate={percent(pr) === null} style="width:{percent(pr) ?? 30}%"></div>
                          {#snippet label()}
                            {#if pr?.stage === 'launch'}<Icon name="loader-circle" size={14} spin />{t('launching')}
                            {:else if percent(pr) !== null}{t('downloadPct', { p: percent(pr) })}
                            {:else}<Icon name="loader-circle" size={14} spin />{t('preparing')}{/if}
                          {/snippet}
                          <span>{@render label()}</span>
                          {#if percent(pr) !== null}
                            <!-- The same label in dark ink, clipped to the filled part. -->
                            <span class="ink" aria-hidden="true" style="clip-path: inset(0 {100 - percent(pr)}% 0 0)">{@render label()}</span>
                          {/if}
                        </div>
                      </div>
                      <span class="stage">{pr?.stage ? t('stage.' + pr.stage) : t('preparing')}{#if pr?.total} · <span class="mono">{pr.done} / {pr.total}</span>{/if}</span>
                    {:else if st?.state === 'running'}
                      <div class="row">
                        <button class="subtle" onclick={() => Stop(selected.id)}><Icon name="square" />{t('stop')}</button>
                        <button class="play running" disabled><span class="dot"></span>{t('runningShort')}</button>
                      </div>
                    {:else}
                      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
                      <div class="play-group" onclick={(e) => e.stopPropagation()}>
                        <button class="primary play" onclick={play}><Icon name="play" size={18} fill />{t('play')}</button>
                        <button class="primary chevron" onclick={() => (picker = !picker)} aria-label={t('pickInstance')} aria-expanded={picker} title={t('pickInstance')}><Icon name="chevron-down" size={18} /></button>
                        {#if picker}
                          <div class="menu" role="menu">
                            {#each app.instances as inst (inst.id)}
                              <button class="subtle" role="menuitem" onclick={() => pick(inst.id)}>
                                <span class="name">{inst.name}</span>
                                <span class="mono">{inst.gameVersion}</span>
                                {#if inst.id === selected.id}<Icon name="check" />{/if}
                              </button>
                            {/each}
                          </div>
                        {/if}
                      </div>
                    {/if}
                  </div>
                </section>

                {#if st?.state === 'error' && st.message}
                  <div class="alert failure">
                    <Icon name="circle-alert" />
                    <div>
                      <b>{tErr(st.message)}</b>
                      {#if st.log?.length}<pre>{st.log.join('\n')}</pre>{/if}
                    </div>
                  </div>
                {/if}

                <div class="tabs" role="tablist">
                  {#each ['overview', 'content'] as id (id)}
                    <button role="tab" aria-selected={tab === id} class:on={tab === id} onclick={() => (tab = id)}>{t('tab.' + id)}</button>
                  {/each}
                </div>

                {#if tab === 'overview'}
                  <h2>{t('openFolder')}</h2>
                  <div class="buttons">
                    {#each folders as [dir, icon] (dir)}
                      {#if dir !== 'mods' || selected.loader !== 'vanilla'}
                        <button onclick={() => OpenInstanceFolder(selected.id, dir)}><Icon name={icon} />{t('folder.' + (dir || 'root'))}</button>
                      {/if}
                    {/each}
                  </div>

                  <h2>Modrinth</h2>
                  <div class="buttons">
                    <button onclick={exportPack} disabled={exporting}>
                      <Icon name={exporting ? 'loader-circle' : 'upload'} spin={exporting} />{exporting ? t('exporting') : t('exportPack')}
                    </button>
                    {#if exported}
                      <span class="done"><Icon name="circle-check" />{t('exported')}</span>
                      <button class="subtle" onclick={() => ShowInFolder(exported)}><Icon name="folder-open" />{t('showInFolder')}</button>
                    {/if}
                  </div>
                  {#if exportError}<p class="error-text">{exportError}</p>{/if}

                  <div class="danger-zone">
                    <div>
                      <b>{t('dangerZone')}</b>
                      <p>{t('deleteHint')}</p>
                    </div>
                    <button class="danger" onclick={remove} onblur={() => (confirmDelete = false)}>
                      <Icon name="trash-2" />{confirmDelete ? t('confirmDelete') : t('deleteInstance')}
                    </button>
                  </div>
                {:else}
                  <Installed instance={selected} version={contentVersion} onlibrary={() => (view = 'library')} />
                {/if}
              {:else}
                <div class="empty-state">
                  <div class="glyph"><Icon name="layout-grid" size={32} /></div>
                  <h3>{t('welcome')}</h3>
                  <p>{t('welcomeSub')}</p>
                  <button class="primary" onclick={() => (modal = 'new')}><Icon name="plus" />{t('newInstance')}</button>
                </div>
              {/if}
            {:else if view === 'instances'}
              <Instances {app} {selected} {status} onpick={(id) => { pick(id); view = 'home' }} oncreate={() => (modal = 'new')} />
            {:else if view === 'library'}
              <header class="head">
                <div>
                  <h1>{t('tab.library')}</h1>
                  {#if selected}<p>{t('librarySub', { name: selected.name })}</p>{/if}
                </div>
              </header>
              {#if selected}
                {#key selected.id}
                  <Library instance={selected} types={libraryTypes(selected)} onchanged={() => contentVersion++} />
                {/key}
              {:else}
                <div class="empty-state">
                  <div class="glyph"><Icon name="compass" size={32} /></div>
                  <h3>{t('welcome')}</h3>
                  <p>{t('noInstanceText')}</p>
                  <button class="primary" onclick={() => (modal = 'new')}><Icon name="plus" />{t('newInstance')}</button>
                </div>
              {/if}
            {:else}
              <Settings {app} onchange={refresh} />
            {/if}
          </div>
        {/key}
      </main>
    </div>
  </div>

  {#if modal === 'new'}
    <NewInstance onclose={() => (modal = '')} oncreated={async () => { modal = ''; tab = 'overview'; view = 'home'; await refresh() }} />
  {:else if modal === 'accounts'}
    <Accounts {app} onclose={() => (modal = '')} onchange={refresh} />
  {/if}
{/if}

<style>
  .window { height: 100%; display: flex; flex-direction: column; background: var(--bg-sidebar); }
  .shell { flex: 1; min-height: 0; display: flex; }

  /* ---- rail (§4.2) ---- */
  .rail { position: relative; z-index: 2; flex: none; width: var(--sidebar-w); display: flex; flex-direction: column; justify-content: space-between; align-items: center; padding: var(--sp-2) 0 var(--sp-3); }
  .group { display: grid; gap: var(--sp-1); }
  .item { position: relative; width: var(--rail-item); height: var(--rail-item); padding: 0; background: transparent; border: 0; color: var(--text-tertiary); }
  .item:hover:not(:disabled) { background: var(--bg-surface-2); color: var(--text-primary); }
  .item.on, .item.on:hover:not(:disabled) { background: var(--accent-a08); color: var(--accent-400); }
  .item.on::before {
    content: ''; position: absolute; top: 50%; width: 3px; height: 16px; margin-top: -8px;
    left: calc((var(--rail-item) - var(--sidebar-w)) / 2); border-radius: var(--r-full); background: var(--accent-400);
  }
  .item :global(.avatar) { border-radius: var(--r-xs); }
  .tip {
    position: absolute; left: calc(100% + var(--sp-2)); top: 50%; transform: translateY(-50%);
    padding: var(--sp-1) var(--sp-2); border-radius: var(--r-xs);
    background: var(--bg-surface-3); box-shadow: var(--shadow-pop);
    font-size: var(--fs-12); font-weight: var(--fw-medium); color: var(--text-primary);
    white-space: nowrap; pointer-events: none; opacity: 0;
    transition: opacity var(--dur-fast);
  }
  .item:hover .tip { opacity: 1; transition-delay: var(--dur-tooltip-delay); }

  /* ---- content sheet (§4.3) ---- */
  .content {
    flex: 1; min-width: 0; display: flex;
    background: var(--bg-surface-1);
    border-top: 1px solid var(--border-subtle); border-left: 1px solid var(--border-subtle);
    border-top-left-radius: var(--r-lg);
  }
  .screen {
    flex: 1; min-width: 0; display: flex; flex-direction: column;
    padding: var(--sp-7) var(--sp-8) var(--sp-6); overflow: auto;
    animation: enter var(--dur-base) var(--ease-out);
  }
  @keyframes enter { from { opacity: 0; transform: translateY(4px); } }

  .head { display: flex; align-items: center; justify-content: space-between; gap: var(--sp-4); min-height: var(--control-h-md); margin-bottom: var(--sp-6); }
  .head h1 { font-size: var(--fs-24); font-weight: var(--fw-semibold); letter-spacing: var(--tracking-tight); line-height: var(--lh-tight); }
  .head p { margin-top: var(--sp-1); color: var(--text-secondary); font-size: var(--fs-12); }

  /* ---- hero (§6) ---- */
  .hero {
    position: relative; flex: none; display: flex; align-items: flex-end; justify-content: space-between; gap: var(--sp-6);
    min-height: clamp(220px, 46vh, 400px); padding: var(--sp-6);
    border-radius: var(--r-lg); background: var(--bg-surface-2); border: 1px solid var(--border-subtle);
  }
  .hero::before {
    /* Faint pixel pattern, the only decoration of the screen. */
    content: ''; position: absolute; inset: 0; border-radius: inherit; pointer-events: none; opacity: 0.035;
    background: repeating-conic-gradient(var(--text-primary) 0 25%, transparent 0 50%) 0 0 / 16px 16px;
    mask-image: linear-gradient(180deg, var(--text-primary), transparent 85%);
  }
  .info { position: relative; min-width: 0; display: grid; gap: var(--sp-1); }
  .info h1 { font-size: var(--fs-32); font-weight: var(--fw-bold); letter-spacing: var(--tracking-tight); line-height: var(--lh-tight); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .meta { font-size: var(--fs-12); color: var(--text-secondary); }

  /* ---- Play (§5.2) ---- */
  .launch { position: relative; flex: none; display: grid; justify-items: end; gap: var(--sp-2); }
  .launch .row { display: flex; align-items: center; gap: var(--sp-2); }
  .play { height: var(--play-h); min-width: var(--play-w); padding: 0 var(--sp-6); font-size: var(--fs-15); font-weight: var(--fw-bold); letter-spacing: var(--tracking-play); }
  .play-group { position: relative; display: flex; border-radius: var(--r-sm); box-shadow: var(--accent-glow); transition: box-shadow var(--dur-fast); }
  .play-group:hover { box-shadow: var(--accent-glow-hover); }
  .play-group .play { border-radius: var(--r-sm) 0 0 var(--r-sm); }
  .play-group button.primary:hover:not(:disabled) { background: var(--accent-300); }
  .play-group button.primary:active:not(:disabled) { background: var(--accent-500); }
  .chevron { width: var(--play-h); height: var(--play-h); padding: 0; border-radius: 0 var(--r-sm) var(--r-sm) 0; border-left: 1px solid var(--group-divider); }
  .menu {
    position: absolute; right: 0; bottom: calc(100% + var(--sp-2)); z-index: 5;
    width: 280px; max-height: 320px; overflow: auto; padding: var(--sp-1);
    background: var(--bg-surface-3); border-radius: var(--r-md); box-shadow: var(--shadow-pop);
    animation: enter var(--dur-base) var(--ease-out);
  }
  .menu button { width: 100%; height: 30px; justify-content: flex-start; padding: 0 var(--sp-2); border-radius: var(--r-xs); color: var(--text-primary); font-weight: var(--fw-regular); }
  .menu button:hover:not(:disabled) { background: var(--bg-surface-4); }
  .menu .name { flex: 1; min-width: 0; text-align: left; overflow: hidden; text-overflow: ellipsis; }
  .menu .mono { font-size: var(--fs-12); color: var(--text-secondary); }
  .menu :global(svg) { color: var(--accent-400); }

  .play.loading { position: relative; display: grid; place-items: center; overflow: hidden; border-radius: var(--r-sm); background: var(--bg-surface-3); }
  .play.loading .fill { position: absolute; left: 0; top: 0; bottom: 0; background: var(--accent-400); transition: width var(--dur-slow); }
  .play.loading .fill.indeterminate { animation: slide var(--dur-indeterminate) var(--ease-inout) infinite; }
  .play.loading span { grid-area: 1 / 1; position: relative; display: inline-flex; align-items: center; justify-content: center; gap: var(--sp-2); color: var(--text-primary); }
  .play.loading .ink { width: 100%; height: 100%; color: var(--text-on-accent); transition: clip-path var(--dur-slow); }
  .play.loading .fill { grid-area: 1 / 1; }
  .stage { font-size: var(--fs-11); color: var(--text-tertiary); }
  .play.running:disabled { opacity: 1; cursor: default; }
  .dot { width: 6px; height: 6px; border-radius: var(--r-full); background: var(--success); }

  .failure { margin-top: var(--sp-4); }
  .failure b { font-weight: var(--fw-semibold); white-space: pre-wrap; }
  .failure pre { margin-top: var(--sp-2); max-height: 140px; overflow: auto; white-space: pre-wrap; font: var(--fs-11)/var(--lh-log) var(--font-mono); color: var(--text-secondary); }

  /* ---- tabs ---- */
  .tabs { display: flex; gap: var(--sp-4); margin: var(--sp-6) 0 var(--sp-1); border-bottom: 1px solid var(--border-subtle); }
  .tabs button { position: relative; height: var(--control-h-md); padding: 0; background: transparent; border: 0; border-radius: 0; color: var(--text-secondary); }
  .tabs button:hover:not(:disabled) { background: transparent; color: var(--text-primary); }
  .tabs button:active:not(:disabled) { transform: none; }
  .tabs button.on { color: var(--text-primary); font-weight: var(--fw-semibold); }
  .tabs button.on::after { content: ''; position: absolute; left: 0; right: 0; bottom: -1px; height: 2px; border-radius: var(--r-full); background: var(--accent-400); }

  h2 { margin: var(--sp-6) 0 var(--sp-3); font-size: var(--fs-14); font-weight: var(--fw-semibold); }
  .buttons { display: flex; flex-wrap: wrap; align-items: center; gap: var(--sp-2); }
  .done { display: inline-flex; align-items: center; gap: 6px; padding: 0 var(--sp-2); color: var(--success); }

  .danger-zone {
    display: flex; align-items: center; justify-content: space-between; gap: var(--sp-4);
    margin-top: var(--sp-8); padding: var(--sp-4); border: 1px solid var(--danger-a12); border-radius: var(--r-md);
  }
  .danger-zone b { font-weight: var(--fw-semibold); }
  .danger-zone p { font-size: var(--fs-12); color: var(--text-secondary); }
</style>
