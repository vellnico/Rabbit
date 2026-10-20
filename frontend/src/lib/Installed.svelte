<script>
  import { untrack } from 'svelte'
  import { fade, slide } from 'svelte/transition'
  import { flip } from 'svelte/animate'
  import { t, tErr } from './i18n.svelte.js'
  import { ListContent, RemoveContent, SetContentEnabled } from '../../wailsjs/go/main/App.js'

  let { instance, onlibrary, version = 0 } = $props()

  let items = $state([])
  let loading = $state(true)
  let error = $state('')

  async function load() {
    loading = true
    error = ''
    try {
      items = (await ListContent(instance.id)) ?? []
    } catch (e) {
      error = tErr(e)
    } finally {
      loading = false
    }
  }

  $effect(() => {
    instance.id, version
    untrack(load)
  })

  async function toggle(it) {
    try {
      it.path = await SetContentEnabled(instance.id, it.path, !it.enabled)
      it.enabled = !it.enabled
    } catch (e) {
      error = tErr(e)
    }
  }

  async function remove(it) {
    try {
      await RemoveContent(instance.id, it.path)
      items = items.filter((x) => x !== it)
    } catch (e) {
      error = tErr(e)
    }
  }

  let groups = $derived(
    ['mod', 'resourcepack', 'shader']
      .map((type) => [type, items.filter((i) => i.type === type)])
      .filter(([, list]) => list.length),
  )
</script>

<div class="wrap">
  {#if error}<p class="error-text">{error}</p>{/if}
  {#if loading && !items.length}
    <div class="list">{#each Array(3) as _, i (i)}<div class="row skeleton"></div>{/each}</div>
  {:else if !items.length}
    <div class="empty" in:fade>
      <svg viewBox="0 0 24 24" width="34" height="34" aria-hidden="true"><path d="M4 7l8-4 8 4-8 4-8-4Zm0 5 8 4 8-4M4 17l8 4 8-4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
      <b>{t('noContent')}</b>
      <p>{t('noContentText')}</p>
      <button class="primary" onclick={onlibrary}>{t('openLibrary')}</button>
    </div>
  {:else}
    <div class="list">
      {#each groups as [type, list] (type)}
        <span class="group">{t('type.' + type)} <em>{list.length}</em></span>
        {#each list as it (it.type + '/' + it.file)}
          <div class="row" class:off={!it.enabled} animate:flip={{ duration: 200 }} out:slide={{ duration: 180 }}>
            {#if it.iconUrl}
              <img class="icon" src={it.iconUrl} alt="" loading="lazy" />
            {:else}
              <div class="icon letter">{(it.title || it.file)[0]}</div>
            {/if}
            <div class="text">
              <b>{it.title || it.file}</b>
              <small>{it.version || it.file}</small>
            </div>
            <button class="switch" class:on={it.enabled} onclick={() => toggle(it)} title={it.enabled ? t('disable') : t('enable')} aria-label={it.enabled ? t('disable') : t('enable')}><i></i></button>
            <button class="ghost del" onclick={() => remove(it)} title={t('remove')} aria-label={t('remove')}>
              <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true"><path d="M5 7h14M10 7V4h4v3m-7 0 1 13h8l1-13" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" /></svg>
            </button>
          </div>
        {/each}
      {/each}
    </div>
  {/if}
</div>

<style>
  .wrap { display: flex; flex-direction: column; min-height: 0; flex: 1; }
  .list { overflow: auto; min-height: 0; display: grid; align-content: start; gap: 6px; padding-right: 4px; }
  .group { margin: 10px 0 2px; font-size: 11px; letter-spacing: .08em; text-transform: uppercase; color: var(--muted); font-weight: 600; }
  .group:first-child { margin-top: 0; }
  .group em { font-style: normal; color: var(--accent); margin-left: 4px; }
  .row { display: flex; align-items: center; gap: 12px; padding: 8px 10px; border-radius: 11px; background: rgb(30 30 34 / .65); border: 1px solid var(--line); transition: opacity .2s, border-color .2s; }
  .row:hover { border-color: var(--line-2); }
  .row.off { opacity: .5; }
  .icon { width: 36px; height: 36px; border-radius: 9px; object-fit: cover; flex: none; background: var(--raised); }
  .letter { display: grid; place-items: center; font-weight: 700; color: var(--accent); background: rgb(var(--accent-rgb) / .12); text-transform: uppercase; }
  .text { flex: 1; min-width: 0; display: grid; }
  .text b { font-weight: 550; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .text small { color: var(--muted); font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .switch { width: 38px; height: 22px; padding: 0; border-radius: 999px; position: relative; background: var(--raised-2); border-color: var(--line-2); flex: none; }
  .switch i { position: absolute; top: 2px; left: 2px; width: 16px; height: 16px; border-radius: 50%; background: var(--muted); transition: transform .25s var(--ease), background .2s; }
  .switch.on { background: rgb(var(--accent-rgb) / .3); border-color: rgb(var(--accent-rgb) / .6); }
  .switch.on i { transform: translateX(16px); background: var(--accent); }
  .switch:hover:not(:disabled) { background: var(--raised-2); }
  .switch.on:hover:not(:disabled) { background: rgb(var(--accent-rgb) / .38); }
  .del { padding: 7px; display: grid; }
  .del:hover:not(:disabled) { color: var(--danger); }
  .skeleton { height: 54px; background: linear-gradient(90deg, var(--raised) 0%, var(--raised-2) 50%, var(--raised) 100%); background-size: 200% 100%; animation: shimmer 1.2s linear infinite; border-color: transparent; }
  @keyframes shimmer { from { background-position: 200% 0; } to { background-position: -200% 0; } }
  .empty { margin: auto; display: grid; justify-items: center; text-align: center; gap: 6px; color: var(--muted); padding: 20px; }
  .empty svg { color: var(--accent); margin-bottom: 4px; }
  .empty b { color: var(--text); font-size: 15px; }
  .empty p { margin: 0 0 10px; font-size: 13px; max-width: 300px; }
</style>
