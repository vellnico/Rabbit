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
  <div class="seg" role="radiogroup">
    {#each tabs as [id, label] (id)}
      <button class="plain" class:on={tab === id} role="radio" aria-checked={tab === id} onclick={() => { tab = id; error = '' }} disabled={id === 'list' && !app.accounts?.length}>{label}</button>
    {/each}
  </div>

  {#if tab === 'list'}
    <div class="list">
      {#each app.accounts ?? [] as a (a.id)}
        <div class="acc">
          <button class="plain pick" class:sel={a.id === app.settings.selectedAccount} onclick={() => select(a.id)}>
            <Avatar account={a} size={32} />
            <span>
              <b>{a.name}</b>
              <small>{a.kind === 'elyby' ? 'Ely.by' : a.elySkin ? t('offlineSkin') : t('offline')}</small>
            </span>
            {#if a.id === app.settings.selectedAccount}
              <svg class="tick" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true"><path d="M3 8.5l3 3 7-7" fill="none" stroke="currentColor" stroke-width="1.6" /></svg>
            {/if}
          </button>
          <button class="plain del" onclick={() => remove(a.id)} aria-label={t('remove')} title={t('remove')}>
            <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true"><path d="M3 4.5h10M6.5 4.5V3h3v1.5M4.5 4.5l.6 8.5h5.8l.6-8.5" fill="none" stroke="currentColor" stroke-width="1.2" /></svg>
          </button>
        </div>
      {/each}
    </div>
  {:else if tab === 'ely' && app.elyBrowser}
    <p class="muted">{waiting ? t('elyWaiting') : t('elyBrowserText')}</p>
    {#if waiting}
      <div class="bar"><div></div></div>
      <div class="actions"><button onclick={() => CancelLogin()}>{t('cancel')}</button></div>
    {:else}
      <div class="actions">
        <button class="plain" onclick={() => BrowserOpenURL('https://account.ely.by/register')}>{t('register')}</button>
        <button class="primary" onclick={browserLogin} disabled={busy}>{t('elyBrowserButton')}</button>
      </div>
    {/if}
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
        <button type="button" class="plain" onclick={() => BrowserOpenURL('https://account.ely.by/register')}>{t('register')}</button>
        <button class="primary" disabled={busy || !login || !password || (needTotp && !totp)}>{busy ? t('signingIn') : t('signIn')}</button>
      </div>
    </form>
  {:else}
    <form onsubmit={(e) => { e.preventDefault(); addOffline() }}>
      <label for="off-nick">{t('nick')}</label>
      <input id="off-nick" bind:value={nick} maxlength="16" placeholder="Steve" />
      <p class="muted small">{t('nickHint')}</p>
      <div class="actions">
        <button class="primary" disabled={busy || nick.length < 3}>{t('add')}</button>
      </div>
    </form>
  {/if}
  {#if error}<p class="error-text">{error}</p>{/if}
</Modal>

<style>
  .seg { display: flex; gap: 2px; padding: 2px; margin: 8px 0 10px; border-radius: var(--radius); background: var(--bg); }
  .seg button { flex: 1; min-height: 28px; color: var(--muted); }
  .seg button.on { background: var(--surface); color: var(--text); box-shadow: 0 0 0 1px var(--border); }
  .list { display: grid; gap: 2px; max-height: 300px; overflow: auto; }
  .acc { display: flex; align-items: center; gap: 2px; }
  .pick { flex: 1; display: flex; align-items: center; gap: 10px; text-align: left; padding: 6px 8px; }
  .pick.sel { background: var(--selected); }
  .pick span { flex: 1; display: grid; }
  .pick b { font-weight: 600; }
  .pick small { font-size: 12px; color: var(--muted); }
  .del { min-height: 28px; width: 28px; padding: 0; display: grid; place-items: center; color: var(--muted); }
  .del:hover:not(:disabled) { color: var(--danger); }
  .muted { color: var(--muted); margin: 4px 0 0; }
  .small { font-size: 12px; margin-top: 6px; }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 18px; }
  .bar { height: 4px; margin-top: 14px; border-radius: 2px; background: var(--selected); overflow: hidden; }
  .bar div { width: 30%; height: 100%; background: var(--accent); animation: slide 1.2s ease-in-out infinite; }
  @keyframes slide { from { transform: translateX(-100%); } to { transform: translateX(340%); } }
</style>
