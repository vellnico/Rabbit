<script>
  import { onMount } from 'svelte'
  import { fade, fly, slide } from 'svelte/transition'
  import { flip } from 'svelte/animate'
  import { cubicOut } from 'svelte/easing'
  import Avatar from './lib/Avatar.svelte'
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

  const folders = [
    ['mods', 'M4 7l8-4 8 4-8 4-8-4Zm0 5 8 4 8-4M4 17l8 4 8-4'],
    ['resourcepacks', 'M4 5h16v14H4zM4 15l5-5 4 4 3-3 4 4'],
    ['shaderpacks', 'M12 3v2m0 14v2M5 12H3m18 0h-2M6.3 6.3 4.9 4.9m14.2 14.2-1.4-1.4M6.3 17.7l-1.4 1.4M19.1 4.9l-1.4 1.4M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8Z'],
    ['saves', 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18Zm-9 9h18M12 3c2.5 2.6 3.8 5.6 3.8 9s-1.3 6.4-3.8 9c-2.5-2.6-3.8-5.6-3.8-9S9.5 5.6 12 3Z'],
    ['config', 'M4 6h10m4 0h2M4 12h4m4 0h8M4 18h12m4 0h0M14 4v4M8 10v4M16 16v4'],
    ['', 'M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7Z'],
  ]
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
  const libraryTypes = (inst) => (inst.loader === 'vanilla' ? ['resourcepack', 'shader'] : ['mod', 'resourcepack', 'shader'])
  const percent = (p) => (p?.total ? Math.round((p.done / p.total) * 100) : null)
  const stageText = (p) => (p?.stage ? t('stage.' + p.stage) : t('preparing'))
</script>

{#if app}
  <div class="layout" in:fade={{ duration: 300 }}>
    <aside>
      <div class="brand">
        <span class="logo">
          <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true"><path fill="currentColor" d="M8.5 2c1.4 0 2 2.6 2 5.4 0 .6 0 1.1-.1 1.6.5-.1 1-.1 1.6-.1s1.1 0 1.6.1c-.1-.5-.1-1-.1-1.6 0-2.8.6-5.4 2-5.4s2 2.6 2 5.4c0 1.2-.2 2.3-.5 3.2C18.8 11.9 20 13.8 20 16c0 3.6-3.6 6-8 6s-8-2.4-8-6c0-2.2 1.2-4.1 3-5.4-.3-.9-.5-2-.5-3.2C6.5 4.6 7.1 2 8.5 2Zm1 12.5a1 1 0 1 0 0 2 1 1 0 0 0 0-2Zm5 0a1 1 0 1 0 0 2 1 1 0 0 0 0-2Z"/></svg>
        </span>
        Rabbit
      </div>

      <nav>
        {#each app.instances ?? [] as inst, i (inst.id)}
          <button
            class="inst"
            class:sel={selected?.id === inst.id}
            style="--lc:{(loaders[inst.loader] ?? loaders.vanilla).color}"
            onclick={() => pick(inst.id)}
            animate:flip={{ duration: 250 }}
            in:fly={{ x: -14, duration: 260, delay: i * 30 }}
            out:slide={{ duration: 200 }}
          >
            <span class="badge">{inst.gameVersion.split('.').slice(0, 2).join('.')}</span>
            <span class="text">
              <span class="name">{inst.name}</span>
              <span class="meta">{inst.gameVersion} · {(loaders[inst.loader] ?? loaders.vanilla).name}</span>
            </span>
            {#if status[inst.id]?.state === 'running' || status[inst.id]?.state === 'preparing'}
              <i class="dot" class:prep={status[inst.id]?.state === 'preparing'} transition:fade></i>
            {/if}
          </button>
        {:else}
          <p class="empty">{t('noInstances')}</p>
        {/each}
      </nav>

      <button class="new" onclick={() => (modal = 'new')}>
        <svg viewBox="0 0 24 24" width="15" height="15" aria-hidden="true"><path d="M12 5v14M5 12h14" stroke="currentColor" stroke-width="2" stroke-linecap="round" /></svg>
        {t('newInstance')}
      </button>

      <div class="bottom">
        <button class="account" onclick={() => (modal = 'accounts')}>
          <Avatar {account} size={32} />
          <span>
            <b>{account?.name ?? t('signIn')}</b>
            <small>{account ? (account.kind === 'elyby' ? 'Ely.by' : t('offline')) : t('signInSub')}</small>
          </span>
        </button>
        <button class="ghost gear" onclick={() => (modal = 'settings')} aria-label={t('settings')} title={t('settings')}>
          <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true"><path fill="none" stroke="currentColor" stroke-width="1.8" d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z"/><path fill="none" stroke="currentColor" stroke-width="1.8" d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1Z"/></svg>
        </button>
      </div>
    </aside>

    <main style="--lc:{loader.color}">
      <div class="glow" aria-hidden="true"></div>
      {#if selected}
        {#key selected.id}
          <div class="content" in:fly={{ y: 12, duration: 320, easing: cubicOut }}>
            <section class="hero">
              <div class="chips">
                <span class="chip loader-chip"><i></i>{loader.name}{selected.loaderVersion ? ' ' + selected.loaderVersion : ''}</span>
                <span class="chip">Minecraft {selected.gameVersion}</span>
              </div>
              <h1>{selected.name}</h1>
              <p class="sub">
                {selected.lastPlayed && formatDate(selected.lastPlayed) ? t('lastPlayed', { date: formatDate(selected.lastPlayed) }) : t('neverPlayed')}
              </p>
            </section>

            <div class="tabs" style="--i:{tabs.indexOf(tab)}">
              {#each tabs as id (id)}
                <button class:on={tab === id} onclick={() => (tab = id)}>{t('tab.' + id)}</button>
              {/each}
              <span class="underline"></span>
            </div>

            {#if tab === 'overview'}
              <div class="panel" in:fade={{ duration: 180 }}>
                <span class="section-title">{t('folders')}</span>
                <section class="folders">
                  {#each folders as [dir, icon], i (dir)}
                    {#if dir !== 'mods' || selected.loader !== 'vanilla'}
                      <button class="folder" onclick={() => OpenInstanceFolder(selected.id, dir)} in:fly={{ y: 8, duration: 260, delay: 60 + i * 35 }}>
                        <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path d={icon} fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" /></svg>
                        {t('folder.' + (dir || 'root'))}
                      </button>
                    {/if}
                  {/each}
                </section>
                <span class="section-title">Modrinth</span>
                <div class="export">
                  <button onclick={exportPack} disabled={exporting}>
                    <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true"><path d="M12 15V3m0 0-4 4m4-4 4 4M5 13v6h14v-6" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" /></svg>
                    {exporting ? t('exporting') : t('exportPack')}
                  </button>
                  {#if exported}
                    <span class="ok-text" in:fade>{t('exported')}</span>
                    <button class="ghost" onclick={() => ShowInFolder(exported)} in:fade>{t('showInFolder')}</button>
                  {/if}
                  {#if exportError}<span class="error-inline">{exportError}</span>{/if}
                </div>
              </div>
            {:else if tab === 'content'}
              <div class="panel grow" in:fade={{ duration: 180 }}>
                <Installed instance={selected} version={contentVersion} onlibrary={() => (tab = 'library')} />
              </div>
            {:else}
              <div class="panel grow" in:fade={{ duration: 180 }}>
                <Library instance={selected} types={libraryTypes(selected)} onchanged={() => contentVersion++} />
              </div>
            {/if}

            {#if st?.state === 'error' && st.message}
              <section class="error" transition:slide={{ duration: 220 }}>
                <b>{tErr(st.message)}</b>
                {#if st.log?.length}<pre>{st.log.join('\n')}</pre>{/if}
              </section>
            {/if}
          </div>
        {/key}

        <footer>
          <div class="status">
            {#if st?.state === 'preparing'}
              <div class="prog" in:fade={{ duration: 150 }}>
                <span>{stageText(pr)}{#if percent(pr) !== null}<em>{percent(pr)}%</em>{/if}</span>
                <div class="bar"><div class:indeterminate={percent(pr) === null} style="width:{percent(pr) ?? 35}%"></div></div>
              </div>
            {:else if st?.state === 'running'}
              <span class="ok" in:fade><i class="dot"></i>{t('running')}</span>
            {:else}
              <button class="ghost danger" onclick={remove} onblur={() => (confirmDelete = false)} in:fade={{ duration: 150 }}>
                {confirmDelete ? t('confirmDelete') : t('deleteInstance')}
              </button>
            {/if}
          </div>
          {#if busy}
            <button class="play stop" onclick={() => Stop(selected.id)} in:fade={{ duration: 150 }}>
              {st.state === 'running' ? t('closeGame') : t('cancel')}
            </button>
          {:else}
            <button class="primary play" onclick={play} in:fade={{ duration: 150 }}>
              <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true"><path d="M7 4.5v15a1 1 0 0 0 1.5.9l12-7.5a1 1 0 0 0 0-1.8l-12-7.5A1 1 0 0 0 7 4.5Z" fill="currentColor" /></svg>
              {t('play')}
            </button>
          {/if}
        </footer>
      {:else}
        <div class="welcome" in:fly={{ y: 16, duration: 400 }}>
          <div class="welcome-logo">
            <svg viewBox="0 0 24 24" width="44" height="44" aria-hidden="true"><path fill="currentColor" d="M8.5 2c1.4 0 2 2.6 2 5.4 0 .6 0 1.1-.1 1.6.5-.1 1-.1 1.6-.1s1.1 0 1.6.1c-.1-.5-.1-1-.1-1.6 0-2.8.6-5.4 2-5.4s2 2.6 2 5.4c0 1.2-.2 2.3-.5 3.2C18.8 11.9 20 13.8 20 16c0 3.6-3.6 6-8 6s-8-2.4-8-6c0-2.2 1.2-4.1 3-5.4-.3-.9-.5-2-.5-3.2C6.5 4.6 7.1 2 8.5 2Zm1 12.5a1 1 0 1 0 0 2 1 1 0 0 0 0-2Zm5 0a1 1 0 1 0 0 2 1 1 0 0 0 0-2Z"/></svg>
          </div>
          <h1>{t('welcome')}</h1>
          <p class="sub">{t('welcomeSub')}</p>
          <button class="primary" onclick={() => (modal = 'new')}>{t('newInstance')}</button>
        </div>
      {/if}
    </main>
  </div>

  {#if modal === 'new'}
    <NewInstance onclose={() => (modal = '')} oncreated={async () => { modal = ''; await refresh() }} />
  {:else if modal === 'accounts'}
    <Accounts {app} onclose={() => (modal = '')} onchange={refresh} />
  {:else if modal === 'settings'}
    <Settings {app} onclose={() => (modal = '')} onchange={refresh} />
  {/if}
{/if}

<style>
  .layout { display: grid; grid-template-columns: 250px 1fr; height: 100%; }

  /* ---- sidebar ---- */
  aside {
    background: linear-gradient(180deg, #131316, #111113);
    border-right: 1px solid var(--line);
    display: flex; flex-direction: column; padding: 16px 10px 10px; min-height: 0;
    position: relative; z-index: 1;
  }
  .brand { display: flex; align-items: center; gap: 10px; font-weight: 700; font-size: 16px; padding: 0 8px 18px; letter-spacing: .01em; }
  .logo {
    width: 30px; height: 30px; border-radius: 9px; display: grid; place-items: center;
    color: var(--accent-ink); background: var(--accent);
    box-shadow: 0 4px 16px -4px rgb(var(--accent-rgb) / .7);
    transition: background .4s, box-shadow .4s;
  }
  nav { flex: 1; overflow: auto; display: grid; align-content: start; gap: 3px; padding: 0 2px; }
  .inst {
    position: relative; text-align: left; background: transparent; border: 1px solid transparent;
    display: flex; align-items: center; gap: 10px; padding: 8px 10px 8px 8px;
  }
  .inst::before {
    content: ''; position: absolute; left: -2px; top: 50%; width: 3px; height: 0; border-radius: 2px;
    background: var(--accent); transform: translateY(-50%); transition: height .25s var(--ease);
  }
  .inst:hover:not(:disabled) { background: rgb(255 255 255 / .03); border-color: transparent; }
  .inst.sel { background: var(--raised); border-color: var(--line); }
  .inst.sel::before { height: 60%; }
  .badge {
    flex: none; width: 36px; height: 36px; border-radius: 10px; display: grid; place-items: center;
    font-size: 11px; font-weight: 700; color: var(--lc);
    background: color-mix(in srgb, var(--lc) 13%, transparent);
    border: 1px solid color-mix(in srgb, var(--lc) 25%, transparent);
  }
  .text { display: grid; min-width: 0; gap: 1px; }
  .inst .name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 500; }
  .inst .meta { font-size: 12px; color: var(--muted); }
  .inst .dot { margin-left: auto; }
  .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--accent); display: inline-block; flex: none; animation: beat 1.8s ease-in-out infinite; }
  .dot.prep { animation-duration: .8s; }
  @keyframes beat { 0%, 100% { box-shadow: 0 0 0 0 rgb(var(--accent-rgb) / .5); } 50% { box-shadow: 0 0 0 5px rgb(var(--accent-rgb) / 0); } }
  .empty { color: var(--muted); padding: 8px 10px; margin: 0; }
  .new { margin: 8px 2px; background: transparent; border-style: dashed; color: var(--muted); display: flex; align-items: center; justify-content: center; gap: 6px; }
  .new:hover:not(:disabled) { color: var(--accent); border-color: rgb(var(--accent-rgb) / .5); background: rgb(var(--accent-rgb) / .05); }
  .bottom { display: flex; align-items: center; gap: 4px; border-top: 1px solid var(--line); padding-top: 10px; }
  .account { flex: 1; display: flex; align-items: center; gap: 10px; background: transparent; border: none; text-align: left; padding: 5px 6px; min-width: 0; }
  .account:hover:not(:disabled) { background: var(--raised); }
  .account span { display: grid; min-width: 0; }
  .account b { font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .account small { color: var(--muted); font-size: 12px; }
  .gear { padding: 8px; display: grid; }
  .gear svg { transition: transform .5s var(--ease); }
  .gear:hover:not(:disabled) svg { transform: rotate(90deg); }

  /* ---- main ---- */
  /* clip, not hidden: the glow must never make main scrollable */
  main { position: relative; display: flex; flex-direction: column; padding: 30px 34px 26px; min-width: 0; min-height: 0; overflow: clip; }
  main::before {
    content: ''; position: absolute; inset: 0; pointer-events: none;
    background-image: radial-gradient(rgb(255 255 255 / .045) 1px, transparent 1px);
    background-size: 22px 22px;
    mask-image: linear-gradient(180deg, #000, transparent 70%);
  }
  .glow {
    position: absolute; width: 560px; height: 560px; right: -180px; top: -260px; border-radius: 50%; pointer-events: none;
    background: radial-gradient(circle, rgb(var(--accent-rgb) / .22), transparent 65%);
    filter: blur(10px);
    animation: drift 18s ease-in-out infinite alternate;
    transition: background .6s;
  }
  .glow::after {
    content: ''; position: absolute; width: 380px; height: 380px; left: -260px; top: 330px; border-radius: 50%;
    background: radial-gradient(circle, color-mix(in srgb, var(--lc) 16%, transparent), transparent 65%);
    transition: background .6s;
  }
  @keyframes drift { from { transform: translate(0, 0) scale(1); } to { transform: translate(-60px, 40px) scale(1.12); } }
  .content { position: relative; min-height: 0; flex: 1; display: flex; flex-direction: column; }
  .tabs { position: relative; display: flex; gap: 4px; margin: 18px 0 14px; border-bottom: 1px solid var(--line); }
  .tabs button { background: transparent; border: none; border-radius: 8px 8px 0 0; color: var(--muted); padding: 8px 14px 10px; width: 120px; }
  .tabs button:hover:not(:disabled) { background: transparent; color: var(--text); }
  .tabs button.on { color: var(--text); }
  .underline {
    position: absolute; bottom: -1px; left: 0; width: 120px; height: 2px; border-radius: 2px;
    background: var(--accent); box-shadow: 0 0 10px rgb(var(--accent-rgb) / .7);
    transform: translateX(calc(var(--i) * 124px)); transition: transform .3s var(--ease);
  }
  .panel { min-height: 0; }
  .panel.grow { flex: 1; display: flex; flex-direction: column; }
  .export { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
  .export > button:first-child { display: flex; align-items: center; gap: 8px; padding: 10px 14px; border-radius: 12px; }
  .ok-text { color: var(--accent); font-size: 13px; }
  .error-inline { color: var(--danger); font-size: 13px; user-select: text; }

  .hero { position: relative; }
  .chips { display: flex; gap: 6px; flex-wrap: wrap; }
  .chip {
    display: inline-flex; align-items: center; gap: 6px; font-size: 12px; color: var(--muted);
    padding: 3px 10px; border-radius: 999px; background: rgb(255 255 255 / .04); border: 1px solid var(--line);
  }
  .loader-chip { color: var(--lc); border-color: color-mix(in srgb, var(--lc) 35%, transparent); background: color-mix(in srgb, var(--lc) 9%, transparent); }
  .loader-chip i { width: 6px; height: 6px; border-radius: 50%; background: var(--lc); box-shadow: 0 0 8px var(--lc); }
  h1 { margin: 10px 0 0; font-size: 30px; font-weight: 750; letter-spacing: -.02em; line-height: 1.1; overflow-wrap: anywhere; }
  .sub { color: var(--muted); margin: 6px 0 0; }

  .section-title { display: block; margin: 4px 0 10px; font-size: 11px; letter-spacing: .08em; text-transform: uppercase; color: var(--muted); font-weight: 600; }
  .folders + .section-title { margin-top: 22px; }
  .folders { display: grid; grid-template-columns: repeat(auto-fill, minmax(128px, 1fr)); gap: 8px; }
  .folder {
    display: flex; align-items: center; gap: 10px; padding: 12px 14px; text-align: left;
    background: rgb(30 30 34 / .7); backdrop-filter: blur(4px); border-radius: 12px;
  }
  .folder svg { color: var(--muted); flex: none; transition: color .2s, transform .25s var(--ease); }
  .folder:hover:not(:disabled) {
    transform: translateY(-2px); border-color: rgb(var(--accent-rgb) / .45);
    box-shadow: 0 10px 24px -14px rgb(var(--accent-rgb) / .6);
  }
  .folder:hover:not(:disabled) svg { color: var(--accent); transform: scale(1.1); }

  .error {
    margin-top: 20px; border: 1px solid #5b2626; background: rgb(40 18 18 / .8); border-radius: var(--radius);
    padding: 12px 14px; color: var(--danger); min-height: 0; display: flex; flex-direction: column; user-select: text;
  }
  .error b { font-weight: 600; white-space: pre-wrap; }
  .error pre { color: #d4a5a5; font-size: 11px; overflow: auto; margin: 10px 0 0; max-height: 170px; white-space: pre-wrap; }

  footer { position: relative; margin-top: auto; display: flex; align-items: center; gap: 22px; padding-top: 20px; }
  .status { flex: 1; min-width: 0; color: var(--muted); display: grid; }
  .prog { display: grid; gap: 9px; }
  .prog span { display: flex; justify-content: space-between; font-size: 13px; color: var(--text); }
  .prog em { font-style: normal; color: var(--accent); font-variant-numeric: tabular-nums; }
  .ok { color: var(--accent); display: flex; align-items: center; gap: 10px; font-weight: 500; }
  .status .danger { justify-self: start; padding-left: 0; }
  .status .danger:hover:not(:disabled) { background: transparent; color: var(--danger); }
  .bar { height: 6px; background: var(--raised-2); border-radius: 3px; overflow: hidden; }
  .bar div {
    height: 100%; border-radius: 3px; transition: width .3s var(--ease);
    background: linear-gradient(90deg, rgb(var(--accent-rgb) / .7), var(--accent));
    background-image: linear-gradient(90deg, rgb(var(--accent-rgb) / .75), var(--accent)),
      repeating-linear-gradient(45deg, rgb(255 255 255 / .18) 0 8px, transparent 8px 16px);
    background-blend-mode: overlay;
    box-shadow: 0 0 12px rgb(var(--accent-rgb) / .6);
  }
  .bar .indeterminate { animation: slide 1.3s ease-in-out infinite; }
  @keyframes slide { from { transform: translateX(-100%); } to { transform: translateX(300%); } }

  .play {
    position: relative; overflow: hidden; min-width: 200px; padding: 15px 26px; font-size: 17px; border-radius: 14px;
    display: flex; align-items: center; justify-content: center; gap: 10px; letter-spacing: .01em;
  }
  .play.primary { box-shadow: 0 10px 34px -10px rgb(var(--accent-rgb) / .85); }
  .play.primary::after {
    content: ''; position: absolute; top: 0; bottom: 0; width: 60px; left: -80px;
    background: linear-gradient(100deg, transparent, rgb(255 255 255 / .45), transparent);
    transform: skewX(-20deg); animation: shine 3.5s ease-in-out infinite;
  }
  .play.primary:hover:not(:disabled) { transform: translateY(-2px); box-shadow: 0 16px 40px -10px rgb(var(--accent-rgb) / 1); }
  @keyframes shine { 0%, 60% { left: -80px; } 100% { left: calc(100% + 80px); } }
  .stop { border-color: var(--line-2); }

  .welcome { position: relative; margin: auto; text-align: center; display: grid; gap: 12px; justify-items: center; }
  .welcome-logo {
    width: 84px; height: 84px; border-radius: 24px; display: grid; place-items: center; margin-bottom: 6px;
    color: var(--accent-ink); background: var(--accent); box-shadow: 0 18px 50px -12px rgb(var(--accent-rgb) / .8);
    animation: float 4s ease-in-out infinite;
  }
  @keyframes float { 0%, 100% { transform: translateY(0); } 50% { transform: translateY(-8px); } }
  .welcome h1 { margin: 0; font-size: 28px; }
</style>
