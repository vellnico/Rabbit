<script>
  import { t } from './i18n.svelte.js'

  let { title, onclose, children, width = 440 } = $props()

  function key(e) {
    if (e.key === 'Escape') onclose()
  }
</script>

<svelte:window onkeydown={key} />

<!-- svelte-ignore a11y_click_events_have_key_events -->
<div class="backdrop" role="presentation" onclick={(e) => e.target === e.currentTarget && onclose()}>
  <div class="dialog" role="dialog" aria-label={title} style="width:min({width}px, calc(100vw - 32px))">
    <header>
      <h2>{title}</h2>
      <button class="plain close" onclick={onclose} aria-label={t('close')} title={t('close')}>
        <svg viewBox="0 0 16 16" width="12" height="12" aria-hidden="true"><path d="M3 3l10 10M13 3 3 13" stroke="currentColor" stroke-width="1.4" /></svg>
      </button>
    </header>
    {@render children()}
  </div>
</div>

<style>
  .backdrop { position: fixed; inset: 0; z-index: 10; background: rgb(0 0 0 / .35); display: grid; place-items: center; }
  .dialog {
    max-height: calc(100vh - 40px); overflow: auto;
    background: var(--surface); border: 1px solid var(--border); border-radius: 8px;
    padding: 16px 20px 20px; box-shadow: 0 16px 40px rgb(0 0 0 / .25);
  }
  header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px; }
  h2 { margin: 0; font-size: 16px; font-weight: 600; }
  .close { min-height: 28px; width: 28px; padding: 0; display: grid; place-items: center; color: var(--muted); }
</style>
