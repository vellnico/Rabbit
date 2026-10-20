<script>
  import Modal from './Modal.svelte'
  import Avatar from './Avatar.svelte'
  import { AddOfflineAccount, LoginEly, RemoveAccount, SelectAccount } from '../../wailsjs/go/main/App.js'
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime.js'

  let { app, onclose, onchange } = $props()

  let tab = $state(app.accounts?.length ? 'list' : 'ely')
  let nick = $state('')
  let login = $state('')
  let password = $state('')
  let totp = $state('')
  let needTotp = $state(false)
  let busy = $state(false)
  let error = $state('')

  async function run(fn) {
    busy = true
    error = ''
    try {
      await fn()
    } catch (e) {
      error = String(e)
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

  const select = (id) => run(async () => { await SelectAccount(id); await onchange() })
  const remove = (id) => run(async () => { await RemoveAccount(id); await onchange() })
</script>

<Modal title="Аккаунты" {onclose}>
  <div class="tabs">
    <button class:on={tab === 'list'} onclick={() => (tab = 'list')} disabled={!app.accounts?.length}>Мои</button>
    <button class:on={tab === 'ely'} onclick={() => (tab = 'ely')}>Ely.by</button>
    <button class:on={tab === 'offline'} onclick={() => (tab = 'offline')}>По нику</button>
  </div>

  {#if tab === 'list'}
    <ul>
      {#each app.accounts ?? [] as a (a.id)}
        <li class:sel={a.id === app.settings.selectedAccount}>
          <button class="pick" onclick={() => select(a.id)}>
            <Avatar account={a} size={32} />
            <span>
              <b>{a.name}</b>
              <small>{a.kind === 'elyby' ? 'Ely.by' : a.elySkin ? 'Офлайн · скин Ely.by' : 'Офлайн'}</small>
            </span>
          </button>
          <button class="ghost" onclick={() => remove(a.id)} aria-label="Удалить">✕</button>
        </li>
      {/each}
    </ul>
  {:else if tab === 'ely'}
    <form onsubmit={(e) => { e.preventDefault(); addEly() }}>
      {#if !needTotp}
        <label for="ely-login">Ник или e-mail</label>
        <input id="ely-login" bind:value={login} autocomplete="username" />
        <label for="ely-pass">Пароль</label>
        <input id="ely-pass" type="password" bind:value={password} autocomplete="current-password" />
      {:else}
        <label for="ely-totp">Код двухфакторной аутентификации</label>
        <!-- svelte-ignore a11y_autofocus -->
        <input id="ely-totp" bind:value={totp} inputmode="numeric" maxlength="6" autofocus />
      {/if}
      {#if error}<p class="error">{error}</p>{/if}
      <div class="actions">
        <button type="button" class="ghost link" onclick={() => BrowserOpenURL('https://account.ely.by/register')}>Регистрация</button>
        <button class="primary" disabled={busy || !login || !password || (needTotp && !totp)}>{busy ? 'Вход…' : 'Войти'}</button>
      </div>
    </form>
  {:else}
    <form onsubmit={(e) => { e.preventDefault(); addOffline() }}>
      <label for="off-nick">Ник</label>
      <input id="off-nick" bind:value={nick} maxlength="16" placeholder="Steve" />
      <p class="hint">Если такой ник есть на Ely.by, в игре будет его скин.</p>
      {#if error}<p class="error">{error}</p>{/if}
      <div class="actions">
        <button class="primary" disabled={busy || nick.length < 3}>Добавить</button>
      </div>
    </form>
  {/if}
  {#if error && tab === 'list'}<p class="error">{error}</p>{/if}
</Modal>

<style>
  .tabs { display: flex; gap: 4px; margin: 14px 0 4px; background: var(--bg); padding: 3px; border-radius: 9px; }
  .tabs button { flex: 1; border: none; background: transparent; color: var(--muted); padding: 6px; }
  .tabs button.on { background: var(--raised); color: var(--text); }
  ul { list-style: none; margin: 10px 0 0; padding: 0; display: grid; gap: 6px; }
  li { display: flex; align-items: center; gap: 6px; border: 1px solid var(--line); border-radius: var(--radius); padding: 4px; }
  li.sel { border-color: var(--accent); }
  .pick { flex: 1; display: flex; align-items: center; gap: 10px; background: transparent; border: none; text-align: left; padding: 4px; }
  .pick:hover:not(:disabled) { background: transparent; }
  .pick span { display: grid; }
  small { color: var(--muted); font-size: 12px; }
  .hint { color: var(--muted); font-size: 12px; margin: 8px 0 0; }
  .actions { display: flex; justify-content: space-between; gap: 8px; margin-top: 18px; }
  .actions .primary { margin-left: auto; }
  .error { color: var(--danger); font-size: 13px; margin: 12px 0 0; user-select: text; }
</style>
