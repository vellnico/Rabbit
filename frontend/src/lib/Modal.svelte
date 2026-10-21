<script>
  import { t } from './i18n.svelte.js'
  import Icon from './Icon.svelte'

  // width: 'sm' (confirmations), 'md' (forms) or 'lg'.
  let { title, onclose, children, width = 'md' } = $props()

  function key(e) {
    if (e.key === 'Escape') onclose()
  }
</script>

<svelte:window onkeydown={key} />

<!-- svelte-ignore a11y_click_events_have_key_events -->
<div class="backdrop" role="presentation" onclick={(e) => e.target === e.currentTarget && onclose()}>
  <div class="dialog" role="dialog" aria-modal="true" aria-label={title} style="width:min(var(--modal-{width}), calc(100vw - var(--sp-8)))">
    <header>
      <h2>{title}</h2>
      <button class="subtle icon sm" onclick={onclose} aria-label={t('close')} title={t('close')}><Icon name="x" /></button>
    </header>
    {@render children()}
  </div>
</div>

<style>
  .backdrop {
    position: fixed; inset: 0; z-index: 10; display: grid; place-items: center;
    background: var(--bg-overlay); backdrop-filter: blur(4px);
    animation: fade var(--dur-base) var(--ease-out);
  }
  .dialog {
    max-height: calc(100vh - var(--sp-10)); overflow: auto;
    padding: var(--sp-6); border-radius: var(--r-lg);
    background: var(--bg-surface-2); box-shadow: var(--shadow-modal);
    animation: pop var(--dur-base) var(--ease-out);
  }
  header { display: flex; align-items: center; justify-content: space-between; gap: var(--sp-4); margin: calc(-1 * var(--sp-1)) calc(-1 * var(--sp-2)) var(--sp-3) 0; }
  h2 { font-size: var(--fs-16); font-weight: var(--fw-semibold); }
  @keyframes fade { from { opacity: 0; } }
  @keyframes pop { from { opacity: 0; transform: scale(0.97); } }
</style>
