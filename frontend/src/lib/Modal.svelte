<script>
  import { t } from './i18n.svelte.js'

  let { title, onclose, children, width = 440 } = $props()

  function key(e) {
    if (e.key === 'Escape') onclose()
  }
</script>

<svelte:window onkeydown={key} />

<!-- A game-style window: opens instantly over a darkened screen. -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<div class="backdrop" role="presentation" onclick={(e) => e.target === e.currentTarget && onclose()}>
  <div class="window panel" role="dialog" aria-label={title} style="width:min({width}px, calc(100vw - 24px))">
    <header>
      <h2>{title}</h2>
      <button class="close" onclick={onclose} aria-label={t('close')} title={t('close')}>
        <svg class="px" viewBox="0 0 7 7" width="14" height="14" shape-rendering="crispEdges" aria-hidden="true"><path fill="currentColor" d="M0 0h1v1H0zM1 1h1v1H1zM2 2h1v1H2zM3 3h1v1H3zM4 4h1v1H4zM5 5h1v1H5zM6 6h1v1H6zM6 0h1v1H6zM5 1h1v1H5zM4 2h1v1H4zM2 4h1v1H2zM1 5h1v1H1zM0 6h1v1H0z" /></svg>
      </button>
    </header>
    {@render children()}
  </div>
</div>

<style>
  .backdrop {
    position: fixed; inset: 0; z-index: 10;
    background: rgb(16 12 9 / .72);
    display: grid; place-items: center;
  }
  .window { max-height: calc(100vh - 32px); overflow: auto; padding: 12px 16px 16px; }
  header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px; }
  h2 { margin: 0; font: 20px var(--pixel); color: var(--ink); }
  .close { padding: 6px; display: grid; }
</style>
