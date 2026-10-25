<script>
  // Release notes of a newer version and the button that installs it.
  import { onMount } from 'svelte'
  import Modal from './Modal.svelte'
  import Icon from './Icon.svelte'
  import { t, tErr } from './i18n.svelte.js'
  import { InstallUpdate } from '../../wailsjs/go/main/App.js'
  import { EventsOn, BrowserOpenURL } from '../../wailsjs/runtime/runtime.js'

  let { update, onclose } = $props()

  let busy = $state(false)
  let progress = $state(null) // {done, total} while downloading
  let error = $state('')

  onMount(() => EventsOn('update', (e) => (progress = e)))

  let percent = $derived(progress?.total ? Math.round((progress.done / progress.total) * 100) : 0)
  let done = $derived(progress?.total > 0 && progress.done >= progress.total)

  // The notes are markdown: headings, list items and paragraphs are enough.
  const inline = (s) => s.replace(/\*\*(.+?)\*\*/g, '$1').replace(/`([^`]+)`/g, '$1').replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
  let blocks = $derived(
    (update.notes ?? '')
      .split(/\r?\n/)
      .map((l) => l.trim())
      .filter(Boolean)
      .map((l) => {
        if (l.startsWith('#')) return { kind: 'h', text: inline(l.replace(/^#+\s*/, '')) }
        if (/^[-*]\s/.test(l)) return { kind: 'li', text: inline(l.slice(2)) }
        return { kind: 'p', text: inline(l) }
      }),
  )

  async function install() {
    busy = true
    error = ''
    progress = { done: 0, total: 0 }
    try {
      await InstallUpdate() // the launcher quits when it succeeds
    } catch (e) {
      error = tErr(e)
      busy = false
      progress = null
    }
  }
</script>

<Modal title={t('update.title')} onclose={() => !busy && onclose()} width="md">
  <p class="versions mono">{update.current} → {update.version}</p>

  {#if blocks.length}
    <div class="notes">
      {#each blocks as b, i (i)}
        {#if b.kind === 'h'}<h3 class="overline">{b.text}</h3>
        {:else if b.kind === 'li'}<p class="li">{b.text}</p>
        {:else}<p>{b.text}</p>{/if}
      {/each}
    </div>
  {/if}

  {#if progress}
    <div class="progress">
      <span>{done ? t('update.restarting') : t('update.downloading', { p: percent })}</span>
      <div class="bar" class:indeterminate={!progress.total}><div style="width:{progress.total ? percent : 30}%"></div></div>
    </div>
  {/if}
  {#if error}<p class="error-text">{error}</p>{/if}

  <div class="actions">
    <button class="subtle" onclick={() => BrowserOpenURL(update.url)}><Icon name="external-link" />{t('update.github')}</button>
    <span class="grow"></span>
    <button onclick={onclose} disabled={busy}>{t('update.later')}</button>
    <button class="primary" onclick={install} disabled={busy}>
      <Icon name={busy ? 'loader-circle' : 'download'} spin={busy} />{t('update.install')}
    </button>
  </div>
</Modal>

<style>
  .versions { color: var(--text-secondary); font-size: var(--fs-12); }
  .notes {
    margin-top: var(--sp-4); max-height: 260px; overflow: auto; padding: var(--sp-3) var(--sp-4);
    background: var(--bg-surface-1); border: 1px solid var(--border-subtle); border-radius: var(--r-md);
    display: grid; gap: var(--sp-1); font-size: var(--fs-13); color: var(--text-secondary);
  }
  .notes h3 { margin-top: var(--sp-2); }
  .notes h3:first-child { margin-top: 0; }
  .notes p { overflow-wrap: anywhere; }
  .li { position: relative; padding-left: var(--sp-4); }
  .li::before { content: ''; position: absolute; left: 4px; top: 0.6em; width: 4px; height: 4px; border-radius: var(--r-full); background: var(--text-tertiary); }
  .progress { display: grid; gap: var(--sp-2); margin-top: var(--sp-4); color: var(--text-secondary); font-size: var(--fs-12); }
  .actions { display: flex; align-items: center; gap: var(--sp-2); margin-top: var(--sp-6); }
  .grow { flex: 1; }
</style>
