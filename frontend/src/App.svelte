<script>
  import { onMount } from 'svelte'
  import Avatar from './lib/Avatar.svelte'
  import NewInstance from './lib/NewInstance.svelte'
  import Accounts from './lib/Accounts.svelte'
  import Settings from './lib/Settings.svelte'
  import { GetState, Launch, Stop, SelectInstance, DeleteInstance, OpenInstanceFolder } from '../wailsjs/go/main/App.js'
  import { EventsOn } from '../wailsjs/runtime/runtime.js'

  let app = $state(null)
  let modal = $state('') // '', 'new', 'accounts', 'settings'
  let progress = $state({}) // instance id -> {stage, done, total}
  let status = $state({}) // instance id -> {state, message, log}
  let confirmDelete = $state(false)

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
    refresh()
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

  const loaderName = (i) => (i.loader === 'fabric' ? `Fabric ${i.loaderVersion}` : 'Vanilla')
  const folders = [
    ['mods', 'Моды'],
    ['resourcepacks', 'Ресурспаки'],
    ['shaderpacks', 'Шейдеры'],
    ['saves', 'Миры'],
    ['config', 'Конфиги'],
  ]
  const percent = (p) => (p?.total ? Math.round((p.done / p.total) * 100) : null)
</script>

{#if app}
  <div class="layout">
    <aside>
      <div class="brand">
        <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path fill="currentColor" d="M8.5 2c1.4 0 2 2.6 2 5.4 0 .6 0 1.1-.1 1.6.5-.1 1-.1 1.6-.1s1.1 0 1.6.1c-.1-.5-.1-1-.1-1.6 0-2.8.6-5.4 2-5.4s2 2.6 2 5.4c0 1.2-.2 2.3-.5 3.2C18.8 11.9 20 13.8 20 16c0 3.6-3.6 6-8 6s-8-2.4-8-6c0-2.2 1.2-4.1 3-5.4-.3-.9-.5-2-.5-3.2C6.5 4.6 7.1 2 8.5 2Zm1 12.5a1 1 0 1 0 0 2 1 1 0 0 0 0-2Zm5 0a1 1 0 1 0 0 2 1 1 0 0 0 0-2Z"/></svg>
        Rabbit
      </div>

      <nav>
        {#each app.instances ?? [] as inst (inst.id)}
          <button class="inst" class:sel={selected?.id === inst.id} onclick={() => pick(inst.id)}>
            <span class="name">{inst.name}</span>
            <span class="meta">
              {inst.gameVersion} · {inst.loader === 'fabric' ? 'Fabric' : 'Vanilla'}
              {#if status[inst.id]?.state === 'running'}<i class="dot"></i>{/if}
            </span>
          </button>
        {:else}
          <p class="empty">Сборок пока нет</p>
        {/each}
      </nav>

      <button class="new" onclick={() => (modal = 'new')}>＋ Новая сборка</button>

      <div class="bottom">
        <button class="account" onclick={() => (modal = 'accounts')}>
          <Avatar {account} />
          <span>
            <b>{account?.name ?? 'Войти'}</b>
            <small>{account ? (account.kind === 'elyby' ? 'Ely.by' : 'Офлайн') : 'Ely.by или ник'}</small>
          </span>
        </button>
        <button class="ghost gear" onclick={() => (modal = 'settings')} aria-label="Настройки">
          <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true"><path fill="none" stroke="currentColor" stroke-width="1.8" d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z"/><path fill="none" stroke="currentColor" stroke-width="1.8" d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1Z"/></svg>
        </button>
      </div>
    </aside>

    <main>
      {#if selected}
        <section class="head">
          <h1>{selected.name}</h1>
          <p class="sub">Minecraft {selected.gameVersion} · {loaderName(selected)}</p>
        </section>

        <section class="folders">
          {#each folders as [dir, label] (dir)}
            {#if dir !== 'mods' || selected.loader !== 'vanilla'}
              <button onclick={() => OpenInstanceFolder(selected.id, dir)}>{label}</button>
            {/if}
          {/each}
          <button onclick={() => OpenInstanceFolder(selected.id, '')}>Папка сборки</button>
        </section>

        {#if st?.state === 'error' && st.message}
          <section class="error">
            <b>{st.message}</b>
            {#if st.log?.length}<pre>{st.log.join('\n')}</pre>{/if}
          </section>
        {/if}

        <footer>
          <div class="status">
            {#if st?.state === 'preparing'}
              <span>{pr?.stage ?? 'Подготовка'}{#if percent(pr) !== null} · {pr.done}/{pr.total}{/if}</span>
              <div class="bar"><div class:indeterminate={percent(pr) === null} style="width:{percent(pr) ?? 30}%"></div></div>
            {:else if st?.state === 'running'}
              <span class="ok">Игра запущена</span>
            {:else}
              <button class="ghost danger" onclick={remove} onblur={() => (confirmDelete = false)}>
                {confirmDelete ? 'Точно удалить?' : 'Удалить сборку'}
              </button>
            {/if}
          </div>
          {#if busy}
            <button class="play stop" onclick={() => Stop(selected.id)}>{st.state === 'running' ? 'Закрыть игру' : 'Отмена'}</button>
          {:else}
            <button class="primary play" onclick={play}>Играть</button>
          {/if}
        </footer>
      {:else}
        <div class="welcome">
          <h1>Добро пожаловать в Rabbit</h1>
          <p class="sub">Создайте первую сборку, чтобы начать играть.</p>
          <button class="primary" onclick={() => (modal = 'new')}>Новая сборка</button>
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
  .layout { display: grid; grid-template-columns: 240px 1fr; height: 100%; }

  aside { background: var(--panel); border-right: 1px solid var(--line); display: flex; flex-direction: column; padding: 14px 10px 10px; min-height: 0; }
  .brand { display: flex; align-items: center; gap: 8px; font-weight: 700; font-size: 16px; padding: 2px 8px 14px; color: var(--accent); }
  nav { flex: 1; overflow: auto; display: grid; align-content: start; gap: 2px; }
  .inst { text-align: left; background: transparent; border: 1px solid transparent; display: grid; gap: 1px; padding: 8px 10px; }
  .inst.sel { background: var(--raised); border-color: var(--line); }
  .inst .name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .inst .meta { font-size: 12px; color: var(--muted); display: flex; align-items: center; gap: 6px; }
  .dot { width: 6px; height: 6px; border-radius: 50%; background: var(--accent); display: inline-block; }
  .empty { color: var(--muted); padding: 8px 10px; margin: 0; }
  .new { margin: 8px 0; background: transparent; border-style: dashed; color: var(--muted); }
  .new:hover:not(:disabled) { color: var(--text); }
  .bottom { display: flex; align-items: center; gap: 4px; border-top: 1px solid var(--line); padding-top: 10px; }
  .account { flex: 1; display: flex; align-items: center; gap: 10px; background: transparent; border: none; text-align: left; padding: 4px 6px; min-width: 0; }
  .account span { display: grid; min-width: 0; }
  .account b { font-weight: 600; overflow: hidden; text-overflow: ellipsis; }
  .account small { color: var(--muted); font-size: 12px; }
  .gear { padding: 8px; display: grid; }

  main { display: flex; flex-direction: column; padding: 28px 32px 24px; min-width: 0; min-height: 0; }
  h1 { margin: 0; font-size: 26px; font-weight: 650; letter-spacing: -.01em; }
  .sub { color: var(--muted); margin: 4px 0 0; }
  .folders { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 24px; }
  .error { margin-top: 20px; border: 1px solid #5b2626; background: #1f1414; border-radius: var(--radius); padding: 12px 14px; color: var(--danger); min-height: 0; display: flex; flex-direction: column; user-select: text; }
  .error pre { color: #d4a5a5; font-size: 11px; overflow: auto; margin: 10px 0 0; max-height: 200px; white-space: pre-wrap; }

  footer { margin-top: auto; display: flex; align-items: center; gap: 20px; padding-top: 20px; }
  .status { flex: 1; min-width: 0; color: var(--muted); display: grid; gap: 8px; }
  .status .ok { color: var(--accent); }
  .status .danger { justify-self: start; padding-left: 0; }
  .status .danger:hover:not(:disabled) { background: transparent; color: var(--danger); }
  .bar { height: 4px; background: var(--raised); border-radius: 2px; overflow: hidden; }
  .bar div { height: 100%; background: var(--accent); transition: width .2s; }
  .bar .indeterminate { animation: slide 1.2s ease-in-out infinite; }
  @keyframes slide { from { transform: translateX(-100%); } to { transform: translateX(340%); } }
  .play { min-width: 180px; padding: 14px 24px; font-size: 16px; border-radius: 12px; }
  .stop { border-color: var(--line); }

  .welcome { margin: auto; text-align: center; display: grid; gap: 14px; justify-items: center; }
</style>
