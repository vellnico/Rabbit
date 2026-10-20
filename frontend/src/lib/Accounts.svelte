<script>
  import { untrack } from 'svelte'
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
  <div class="modes">
    {#each tabs as [id, label] (id)}
      <button class:on={tab === id} aria-pressed={tab === id} onclick={() => { tab = id; error = '' }} disabled={id === 'list' && !app.accounts?.length}>{label}</button>
    {/each}
  </div>

  {#if tab === 'list'}
    <div class="slot list">
      {#each app.accounts ?? [] as a (a.id)}
        <div class="acc" class:sel={a.id === app.settings.selectedAccount}>
          <button class="pick flat" onclick={() => select(a.id)}>
            <Avatar account={a} size={40} />
            <span>
              <b>{a.name}</b>
              <small>{a.kind === 'elyby' ? 'Ely.by' : a.elySkin ? t('offlineSkin') : t('offline')}</small>
            </span>
          </button>
          <button class="del" onclick={() => remove(a.id)} aria-label={t('remove')} title={t('remove')}>
            <svg class="px" viewBox="0 0 7 7" width="12" height="12" shape-rendering="crispEdges" aria-hidden="true"><path fill="currentColor" d="M0 0h1v1H0zM1 1h1v1H1zM2 2h1v1H2zM3 3h1v1H3zM4 4h1v1H4zM5 5h1v1H5zM6 6h1v1H6zM6 0h1v1H6zM5 1h1v1H5zM4 2h1v1H4zM2 4h1v1H2zM1 5h1v1H1zM0 6h1v1H0z" /></svg>
          </button>
        </div>
      {/each}
    </div>
  {:else if tab === 'ely' && app.elyBrowser}
    <div class="ely">
      <p>{waiting ? t('elyWaiting') : t('elyBrowserText')}</p>
      {#if waiting}
        <div class="xp"><div class="fill crawl"></div></div>
        <button onclick={() => CancelLogin()}>{t('cancel')}</button>
      {:else}
        <button class="primary" onclick={browserLogin} disabled={busy}>{t('elyBrowserButton')}</button>
        <button class="flat" onclick={() => BrowserOpenURL('https://account.ely.by/register')}>{t('register')}</button>
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
        <button type="button" class="flat" onclick={() => BrowserOpenURL('https://account.ely.by/register')}>{t('register')}</button>
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
  {#if error}<p class="error-text">{error}</p>{/if}
</Modal>

<style>
  .modes { display: flex; gap: 4px; margin: 6px 0 10px; }
  .modes button { flex: 1; font-size: 15px; }
  .modes button.on { background: #6f6f6f; box-shadow: inset 2px 2px 0 #4a4a4a, inset -2px -2px 0 #9a9a9a; outline: 2px solid var(--accent); outline-offset: -4px; }
  .list { padding: 4px; display: grid; gap: 2px; max-height: 300px; overflow: auto; }
  .acc { display: flex; align-items: center; gap: 4px; border: 2px solid transparent; }
  .acc:hover { background: rgb(0 0 0 / .18); }
  .acc.sel { border-color: #fff; background: rgb(255 255 255 / .2); }
  .pick { flex: 1; display: flex; align-items: center; gap: 10px; text-align: left; padding: 4px; font: inherit; color: #fff; }
  .pick:hover:not(:disabled) { background: none; }
  .pick:active:not(:disabled) { padding: 4px; }
  .pick span { display: grid; }
  .pick b { font: 16px var(--pixel); font-weight: 400; text-shadow: 2px 2px 0 #3f3f3f; }
  .pick small { font-size: 12px; color: #ececec; }
  .del { padding: 6px; display: grid; margin-right: 4px; }
  .del:hover:not(:disabled) { color: #ff5555; }
  .ely { display: grid; justify-items: start; gap: 10px; padding: 4px 0; }
  .ely p { margin: 0 0 4px; max-width: 380px; }
  .xp { width: 100%; position: relative; height: 14px; background: #1b1b1b; border: 2px solid #000; overflow: hidden; }
  .fill { width: 20%; height: 100%; background: var(--accent); }
  .fill.crawl { animation: crawl 1.4s steps(10) infinite; }
  @keyframes crawl { from { transform: translateX(-100%); } to { transform: translateX(500%); } }
  .hint { font-size: 13px; color: #555; margin: 6px 0 0; }
  .actions { display: flex; justify-content: space-between; gap: 8px; margin-top: 16px; }
  .actions .primary { margin-left: auto; }
</style>
