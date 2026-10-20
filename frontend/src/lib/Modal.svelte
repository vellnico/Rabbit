<script>
  import { fade, scale } from 'svelte/transition'
  import { cubicOut } from 'svelte/easing'
  import { t } from './i18n.svelte.js'

  let { title, onclose, children, width = 420 } = $props()

  function key(e) {
    if (e.key === 'Escape') onclose()
  }
</script>

<svelte:window onkeydown={key} />

<!-- svelte-ignore a11y_click_events_have_key_events -->
<div class="backdrop" role="presentation" transition:fade={{ duration: 160 }} onclick={(e) => e.target === e.currentTarget && onclose()}>
  <div class="modal" style="width:min({width}px, calc(100vw - 32px))" transition:scale={{ duration: 220, start: 0.94, easing: cubicOut }}>
    <header>
      <h2>{title}</h2>
      <button class="ghost close" onclick={onclose} aria-label={t('close')}>
        <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true"><path d="M6 6l12 12M18 6 6 18" stroke="currentColor" stroke-width="2" stroke-linecap="round" /></svg>
      </button>
    </header>
    {@render children()}
  </div>
</div>

<style>
  .backdrop {
    position: fixed; inset: 0;
    background: rgb(5 5 7 / .6);
    backdrop-filter: blur(6px);
    display: grid; place-items: center;
    z-index: 10;
  }
  .modal {
    max-height: calc(100vh - 48px);
    overflow: auto;
    background: linear-gradient(180deg, #1a1a1e, var(--panel));
    border: 1px solid var(--line);
    border-radius: 16px;
    padding: 18px 20px 20px;
    box-shadow: 0 30px 80px -20px rgb(0 0 0 / .7), 0 0 0 1px rgb(255 255 255 / .02) inset;
  }
  header { display: flex; align-items: center; justify-content: space-between; }
  h2 { margin: 0; font-size: 17px; font-weight: 650; }
  .close { padding: 6px; display: grid; }
  .close:hover:not(:disabled) svg { transform: rotate(90deg); }
  .close svg { transition: transform .25s var(--ease); }
</style>
