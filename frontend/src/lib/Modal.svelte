<script>
  import { t } from './i18n.svelte.js'
  import Icon from './Icon.svelte'

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
      <button class="subtle icon" onclick={onclose} aria-label={t('close')} title={t('close')}><Icon name="x" /></button>
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
  header { display: flex; align-items: center; justify-content: space-between; margin: -4px -8px 4px 0; }
  h2 { margin: 0; font-size: 16px; font-weight: 600; }
</style>
