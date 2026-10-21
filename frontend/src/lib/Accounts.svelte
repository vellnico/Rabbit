<script>
  import { untrack, onMount } from 'svelte'
  import Modal from './Modal.svelte'
  import Avatar from './Avatar.svelte'
  import Icon from './Icon.svelte'
  import { t, tErr, accountKind } from './i18n.svelte.js'
  import { AddOfflineAccount, LoginEly, LoginElyBrowser, LoginMicrosoft, CancelLogin, RemoveAccount, SelectAccount } from '../../wailsjs/go/main/App.js'
  import { BrowserOpenURL, ClipboardSetText, EventsOn } from '../../wailsjs/runtime/runtime.js'

  let { app, onclose, onchange } = $props()

  let tab = $state(untrack(() => (app.accounts?.length ? 'list' : app.microsoft ? 'ms' : 'ely')))
  let nick = $state('')
  let login = $state('')
  let password = $state('')
  let totp = $state('')
  let needTotp = $state(false)
  let busy = $state(false)
  let waiting = $state(false)
  let msCode = $state(null) // {code, url} while waiting for Microsoft
  let copied = $state(false)
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

  onMount(() => EventsOn('mscode', (dc) => (msCode = dc)))

  const msLogin = () =>
    run(async () => {
      waiting = true
      try {
        await LoginMicrosoft()
      } finally {
        waiting = false
        msCode = null
        copied = false
      }
      await onchange()
      tab = 'list'
    })

  async function copyCode() {
    await ClipboardSetText(msCode.code)
    copied = true
  }

  function close() {
    if (waiting) CancelLogin()
    onclose()
  }

  const select = (id) => run(async () => { await SelectAccount(id); await onchange() })
  const remove = (id) => run(async () => { await RemoveAccount(id); await onchange() })
  const tabs = $derived([
    ['list', t('mine')],
    ['ms', 'Microsoft'],
    ['ely', 'Ely.by'],
    ['offline', t('byNick')],
  ])
</script>

<Modal title={t('accounts')} onclose={close}>
  <div class="seg" role="radiogroup">
    {#each tabs as [id, label] (id)}
      <button class:on={tab === id} role="radio" aria-checked={tab === id} onclick={() => { tab = id; error = '' }} disabled={(id === 'list' && !app.accounts?.length) || (waiting && id !== tab)}>{label}</button>
    {/each}
  </div>

  {#if tab === 'list'}
    <div class="list">
      {#each app.accounts ?? [] as a (a.id)}
        <div class="acc">
          <button class="subtle pick" class:sel={a.id === app.settings.selectedAccount} onclick={() => select(a.id)}>
            <Avatar account={a} size={32} />
            <span>
              <b>{a.name}</b>
              <span class="badge">{accountKind(a)}</span>
            </span>
            {#if a.id === app.settings.selectedAccount}
              <Icon name="check" />
            {/if}
          </button>
          <button class="subtle icon sm danger" onclick={() => remove(a.id)} aria-label={t('remove')} title={t('remove')}><Icon name="trash-2" /></button>
        </div>
      {/each}
    </div>
  {:else if tab === 'ms'}
    {#if msCode}
      <p class="muted">{t('msWaiting')}</p>
      <div class="code">
        <span>{msCode.code}</span>
        <button class="subtle" onclick={copyCode}><Icon name={copied ? 'check' : 'copy'} />{copied ? t('copied') : t('copy')}</button>
      </div>
      <div class="bar indeterminate"><div></div></div>
      <div class="actions">
        <button class="subtle" onclick={() => BrowserOpenURL(msCode.url)}><Icon name="external-link" />{t('msOpenPage')}</button>
        <button onclick={() => CancelLogin()}>{t('cancel')}</button>
      </div>
    {:else}
      <p class="muted">{app.microsoft ? t('msText') : t('err.msNotConfigured')}</p>
      <div class="actions">
        <button class="subtle" onclick={() => BrowserOpenURL('https://www.minecraft.net/store/minecraft-java-bedrock-edition-pc')}><Icon name="external-link" />{t('buyGame')}</button>
        <button class="primary" onclick={msLogin} disabled={busy || !app.microsoft}><Icon name={busy ? 'loader-circle' : 'log-in'} spin={busy} />{t('msButton')}</button>
      </div>
    {/if}
  {:else if tab === 'ely' && app.elyBrowser}
    <p class="muted">{waiting ? t('elyWaiting') : t('elyBrowserText')}</p>
    {#if waiting}
      <div class="bar indeterminate"><div></div></div>
      <div class="actions"><button onclick={() => CancelLogin()}>{t('cancel')}</button></div>
    {:else}
      <div class="actions">
        <button class="subtle" onclick={() => BrowserOpenURL('https://account.ely.by/register')}><Icon name="external-link" />{t('register')}</button>
        <button class="primary" onclick={browserLogin} disabled={busy}><Icon name="log-in" />{t('elyBrowserButton')}</button>
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
        <button type="button" class="subtle" onclick={() => BrowserOpenURL('https://account.ely.by/register')}><Icon name="external-link" />{t('register')}</button>
        <button class="primary" disabled={busy || !login || !password || (needTotp && !totp)}><Icon name={busy ? 'loader-circle' : 'log-in'} spin={busy} />{busy ? t('signingIn') : t('signIn')}</button>
      </div>
    </form>
  {:else}
    <form onsubmit={(e) => { e.preventDefault(); addOffline() }}>
      <label for="off-nick">{t('nick')}</label>
      <input id="off-nick" bind:value={nick} maxlength="16" placeholder="Steve" />
      <p class="hint">{t('nickHint')}</p>
      <div class="actions">
        <button class="primary" disabled={busy || nick.length < 3}><Icon name="user-plus" />{t('add')}</button>
      </div>
    </form>
  {/if}
  {#if error}<p class="error-text">{error}</p>{/if}
</Modal>

<style>
  .seg { margin-bottom: var(--sp-3); }
  .list { display: grid; gap: var(--sp-half); max-height: 300px; overflow: auto; }
  .acc { display: flex; align-items: center; gap: var(--sp-1); }
  .pick { flex: 1; height: auto; justify-content: flex-start; gap: var(--sp-3); text-align: left; padding: var(--sp-2); font-weight: var(--fw-regular); color: var(--text-primary); }
  .pick.sel { background: var(--bg-surface-4); }
  .pick span:not(.badge) { flex: 1; min-width: 0; display: flex; align-items: center; gap: var(--sp-2); }
  .pick b { font-weight: var(--fw-semibold); overflow: hidden; text-overflow: ellipsis; }
  .pick > :global(svg) { color: var(--accent-400); }
  .muted { color: var(--text-secondary); }
  .code { display: flex; align-items: center; justify-content: space-between; gap: var(--sp-2); margin-top: var(--sp-3); padding: var(--sp-2) var(--sp-2) var(--sp-2) var(--sp-4); border: 1px solid var(--border-default); border-radius: var(--r-sm); background: var(--bg-surface-1); }
  .code span { font: var(--fw-semibold) var(--fs-24) / var(--lh-tight) var(--font-mono); letter-spacing: var(--tracking-caps); user-select: text; }
  .actions { display: flex; justify-content: flex-end; gap: var(--sp-2); margin-top: var(--sp-6); }
  .actions :global(.subtle) { margin-right: auto; }
  .bar { margin-top: var(--sp-4); }
</style>
