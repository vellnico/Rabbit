<script>
  import { untrack } from 'svelte'
  import { fly, slide } from 'svelte/transition'
  import { flip } from 'svelte/animate'
  import Modal from './Modal.svelte'
  import Avatar from './Avatar.svelte'
  import { t, tErr } from './i18n.svelte.js'
  import { AddOfflineAccount, LoginEly, LoginElyBrowser, CancelLogin, RemoveAccount, SelectAccount } from '../../wailsjs/go/main/App.js'
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime.js'

  let { app, onclose, onchange } = $props()

  let tab = $state(untrack(() => app.accounts?.length) ? 'list' : 'ely')
  let nick = $state('')
  let login = $state('')
  let password = $state('')
  let totp = $state('')
  let needTotp = $state(false)
  let busy = $state(false)
  let waiting = $state(false)
  let error = $state('')

  async function run(fn) {
    busy = true
    error = ''
    try {
      await fn()
    } catch (e) {
      error = tErr(e)
    } finally {
      busy = false
    }
  }

  const addOffline = () =>
    run(async () => {
      await AddOfflineAccount(nick)
      nick = ''
      await onchange()
      tab = 'list'
    })

  const addEly = () =>
    run(async () => {
      const r = await LoginEly(login, password, needTotp ? totp : '')
      if (r === 'totp') {
        needTotp = true
        return
      }
      login = password = totp = ''
      needTotp = false
      await onchange()
      tab = 'list'
    })

  const browserLogin = () =>
    run(async () => {
      waiting = true
      try {
        await LoginElyBrowser()
      } finally {
        waiting = false
      }
      await onchange()
      tab = 'list'
    })

  function close() {
    if (waiting) CancelLogin()
    onclose()
  }

  const select = (id) => run(async () => { await SelectAccount(id); await onchange() })
  const remove = (id) => run(async () => { await RemoveAccount(id); await onchange() })
  const tabs = $derived([
    ['list', t('mine')],
    ['ely', 'Ely.by'],
    ['offline', t('byNick')],
  ])
</script>

<Modal title={t('accounts')} onclose={close}>
  <div class="tabs" style="--i:{tabs.findIndex(([id]) => id === tab)}">
    <span class="pill"></span>
    {#each tabs as [id, label] (id)}
      <button class:on={tab === id} onclick={() => { tab = id; error = '' }} disabled={id === 'list' && !app.accounts?.length}>{label}</button>
    {/each}
  </div>

  {#key tab}
    <div class="pane" in:fly={{ y: 8, duration: 220 }}>
      {#if tab === 'list'}
        <ul>
          {#each app.accounts ?? [] as a (a.id)}
            <li class:sel={a.id === app.settings.selectedAccount} animate:flip={{ duration: 200 }} out:slide={{ duration: 180 }}>
              <button class="pick" onclick={() => select(a.id)}>
                <Avatar account={a} size={34} />
                <span>
                  <b>{a.name}</b>
                  <small>{a.kind === 'elyby' ? 'Ely.by' : a.elySkin ? t('offlineSkin') : t('offline')}</small>
                </span>
              </button>
              <button class="ghost" onclick={() => remove(a.id)} aria-label={t('remove')} title={t('remove')}>
                <svg viewBox="0 0 24 24" width="15" height="15" aria-hidden="true"><path d="M6 6l12 12M18 6 6 18" stroke="currentColor" stroke-width="2" stroke-linecap="round" /></svg>
              </button>
            </li>
          {/each}
        </ul>
      {:else if tab === 'ely' && app.elyBrowser}
        <div class="ely">
          <div class="ely-logo" class:pulse={waiting}>
            <svg viewBox="0 0 24 24" width="30" height="30" aria-hidden="true"><path fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4M10 17l5-5-5-5M15 12H3" /></svg>
          </div>
          <h3>{t('elyBrowserTitle')}</h3>
          <p>{waiting ? t('elyWaiting') : t('elyBrowserText')}</p>
          {#if waiting}
            <div class="dots"><i></i><i></i><i></i></div>
            <button class="ghost" onclick={() => CancelLogin()}>{t('cancel')}</button>
          {:else}
            <button class="primary big" onclick={browserLogin} disabled={busy}>{t('elyBrowserButton')}</button>
            <button class="ghost small" onclick={() => BrowserOpenURL('https://account.ely.by/register')}>{t('register')}</button>
          {/if}
        </div>
      {:else if tab === 'ely'}
        <form onsubmit={(e) => { e.preventDefault(); addEly() }}>
          {#if !needTotp}
            <label for="ely-login">{t('loginOrEmail')}</label>
            <input id="ely-login" bind:value={login} autocomplete="username" />
            <label for="ely-pass">{t('password')}</label>
            <input id="ely-pass" type="password" bind:value={password} autocomplete="current-password" />
          {:else}
            <label for="ely-totp">{t('totp')}</label>
            <!-- svelte-ignore a11y_autofocus -->
            <input id="ely-totp" bind:value={totp} inputmode="numeric" maxlength="6" autofocus />
          {/if}
          <div class="actions">
            <button type="button" class="ghost" onclick={() => BrowserOpenURL('https://account.ely.by/register')}>{t('register')}</button>
            <button class="primary" disabled={busy || !login || !password || (needTotp && !totp)}>{busy ? t('signingIn') : t('signIn')}</button>
          </div>
        </form>
      {:else}
        <form onsubmit={(e) => { e.preventDefault(); addOffline() }}>
          <label for="off-nick">{t('nick')}</label>
          <input id="off-nick" bind:value={nick} maxlength="16" placeholder="Steve" />
          <p class="hint">{t('nickHint')}</p>
          <div class="actions">
            <button class="primary" disabled={busy || nick.length < 3}>{t('add')}</button>
          </div>
        </form>
      {/if}
    </div>
  {/key}
  {#if error}<p class="error-text" transition:slide={{ duration: 180 }}>{error}</p>{/if}
</Modal>

<style>
  .tabs { position: relative; display: grid; grid-template-columns: repeat(3, 1fr); margin: 16px 0 6px; background: var(--bg); padding: 3px; border-radius: 10px; border: 1px solid var(--line); }
  .pill {
    position: absolute; top: 3px; bottom: 3px; left: 3px;
    width: calc((100% - 6px) / 3);
    transform: translateX(calc(var(--i) * 100%));
    background: var(--raised-2); border-radius: 8px;
    transition: transform .3s var(--ease);
  }
  .tabs button { position: relative; border: none; background: transparent; color: var(--muted); padding: 6px; }
  .tabs button:hover:not(:disabled) { background: transparent; color: var(--text); }
  .tabs button.on { color: var(--text); }
  ul { list-style: none; margin: 10px 0 0; padding: 0; display: grid; gap: 6px; }
  li { display: flex; align-items: center; gap: 6px; border: 1px solid var(--line); border-radius: var(--radius); padding: 4px; transition: border-color .2s, background .2s; }
  li:hover { background: rgb(255 255 255 / .02); }
  li.sel { border-color: rgb(var(--accent-rgb) / .7); background: rgb(var(--accent-rgb) / .05); }
  .pick { flex: 1; display: flex; align-items: center; gap: 11px; background: transparent; border: none; text-align: left; padding: 4px; }
  .pick:hover:not(:disabled) { background: transparent; }
  .pick span { display: grid; }
  small { color: var(--muted); font-size: 12px; }
  .hint { color: var(--muted); font-size: 12px; margin: 8px 0 0; }
  .actions { display: flex; justify-content: space-between; gap: 8px; margin-top: 18px; }
  .actions .primary { margin-left: auto; }
  .ely { display: grid; justify-items: center; text-align: center; padding: 16px 8px 4px; gap: 4px; }
  .ely-logo { width: 62px; height: 62px; border-radius: 18px; display: grid; place-items: center; color: var(--accent); background: rgb(var(--accent-rgb) / .1); border: 1px solid rgb(var(--accent-rgb) / .25); margin-bottom: 8px; }
  .ely-logo.pulse { animation: pulse 1.6s var(--ease) infinite; }
  @keyframes pulse { 0%, 100% { box-shadow: 0 0 0 0 rgb(var(--accent-rgb) / .35); } 50% { box-shadow: 0 0 0 12px rgb(var(--accent-rgb) / 0); } }
  h3 { margin: 0; font-size: 15px; }
  .ely p { margin: 4px 0 14px; color: var(--muted); font-size: 13px; max-width: 320px; }
  .big { padding: 11px 22px; width: 100%; }
  .small { font-size: 12.5px; margin-top: 4px; }
  .dots { display: flex; gap: 6px; margin-bottom: 10px; }
  .dots i { width: 7px; height: 7px; border-radius: 50%; background: var(--accent); animation: bounce 1s ease-in-out infinite; }
  .dots i:nth-child(2) { animation-delay: .15s; }
  .dots i:nth-child(3) { animation-delay: .3s; }
  @keyframes bounce { 0%, 100% { transform: translateY(0); opacity: .4; } 50% { transform: translateY(-5px); opacity: 1; } }
</style>
