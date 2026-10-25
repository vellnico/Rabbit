<script>
  // Frameless window title bar: drag area plus the window buttons. A double
  // click on the drag area maximises natively (it acts as the caption).
  import Logo from './Logo.svelte'
  import Icon from './Icon.svelte'
  import { t } from './i18n.svelte.js'
  import { WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime.js'

  // update: a newer release ({version, ...}) or null; onupdate opens its dialog.
  let { update = null, onupdate = () => {} } = $props()
</script>

<header class="titlebar" style="--wails-draggable: drag">
  <div class="brand"><Logo size={16} /><span>Rabbit</span></div>
  <div class="buttons" style="--wails-draggable: no-drag">
    {#if update}
      <button class="update" onclick={onupdate}><Icon name="download" size={12} />{t('update.available', { v: update.version })}</button>
    {/if}
    <button onclick={WindowMinimise} aria-label={t('minimize')} title={t('minimize')}><Icon name="minus" size={14} stroke={1.5} /></button>
    <button onclick={WindowToggleMaximise} aria-label={t('maximize')} title={t('maximize')}><Icon name="square" size={12} stroke={1.5} /></button>
    <button class="close" onclick={Quit} aria-label={t('close')} title={t('close')}><Icon name="x" size={14} stroke={1.5} /></button>
  </div>
</header>

<style>
  .titlebar { flex: none; height: var(--titlebar-h); display: flex; align-items: center; background: var(--bg-sidebar); }
  .brand { display: flex; align-items: center; gap: var(--sp-2); padding-left: var(--sp-5); font-size: var(--fs-12); font-weight: var(--fw-medium); color: var(--text-secondary); }
  .buttons { margin-left: auto; display: flex; align-items: center; height: 100%; }
  .buttons button {
    width: var(--titlebar-btn-w); height: 100%; padding: 0;
    background: transparent; border: 0; border-radius: 0; color: var(--text-secondary);
  }
  .buttons button:hover:not(:disabled) { background: var(--bg-surface-3); color: var(--text-primary); }
  .buttons button:active:not(:disabled) { transform: none; }
  .buttons .close:hover:not(:disabled) { background: var(--close-hover); color: var(--text-primary); }
  .buttons .update {
    width: auto; height: 22px; margin-right: var(--sp-2); padding: 0 var(--sp-2); gap: var(--sp-1);
    border-radius: var(--r-full); background: var(--accent-a14); color: var(--accent-300);
    font-size: var(--fs-11); font-weight: var(--fw-medium);
  }
  .buttons .update:hover:not(:disabled) { background: var(--accent-a24); color: var(--accent-200); }
</style>
