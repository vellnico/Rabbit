<script>
  import { onMount } from 'svelte'
  import Logo from '../src/lib/Logo.svelte'
  import Icon from '../src/lib/Icon.svelte'
  import { applyAccent } from '../src/lib/theme.js'
  import { t, tErr, setLanguage } from './strings.svelte.js'
  import { EventsOn, WindowMinimise, Quit } from '../wailsjs/runtime/runtime.js'

  // Bound Go methods (no generated bindings: the setup is its own app).
  const api = () => window.go.main.Setup

  let info = $state(null)
  let step = $state('ready') // ready | working | done
  let dir = $state('')
  let desktop = $state(true)
  let removeData = $state(false)
  let progress = $state(null) // {done, total}
  let error = $state('')

  // install | update | uninstall
  let kind = $derived(info?.mode === 'uninstall' ? 'uninstall' : info?.installedVersion ? 'update' : 'install')
  let percent = $derived(progress?.total ? Math.min(100, Math.round((progress.done / progress.total) * 100)) : null)

  onMount(() => {
    api().Info().then((i) => {
      info = i
      dir = i.dir
      desktop = i.desktop
      applyAccent(i.accent)
      setLanguage(i.language)
    })
    return EventsOn('progress', (e) => (progress = e))
  })

  async function pick() {
    dir = await api().PickDir(dir)
  }

  async function run() {
    error = ''
    progress = null
    step = 'working'
    try {
      if (kind === 'uninstall') await api().Uninstall(removeData)
      else await api().Install(dir, desktop)
      step = 'done'
    } catch (e) {
      error = tErr(e)
      step = 'ready'
    }
  }
</script>

<svelte:window oncontextmenu={(e) => !e.target.closest('input, .selectable') && e.preventDefault()} />

<div class="window">
  <header class="titlebar" style="--wails-draggable: drag">
    <div class="brand"><Logo size={16} /><span>Rabbit</span></div>
    <div class="buttons" style="--wails-draggable: no-drag">
      <button onclick={WindowMinimise} aria-label={t('minimize')} title={t('minimize')}><Icon name="minus" size={14} stroke={1.5} /></button>
      <button class="close" onclick={Quit} disabled={step === 'working'} aria-label={t('close')} title={t('close')}><Icon name="x" size={14} stroke={1.5} /></button>
    </div>
  </header>

  {#if info}
    <main class="content">
      {#if step === 'done'}
        <section class="done">
          <div class="glyph"><Icon name="circle-check" size={28} /></div>
          <h1>{t('done.' + kind)}</h1>
          <p>
            {#if kind === 'uninstall'}{removeData ? t('doneSub.uninstallAll') : t('doneSub.uninstallKept', { dir: info.dataDir })}
            {:else}{t('doneSub.install')}{/if}
          </p>
        </section>
        <footer>
          <button class="subtle lg" onclick={Quit}>{t('close')}</button>
          {#if kind !== 'uninstall'}
            <div class="glow"><button class="primary lg" onclick={() => api().Launch()}><Icon name="play" size={16} fill />{t('launch')}</button></div>
          {/if}
        </footer>
      {:else}
        <section class="hero">
          <div class="mark"><Logo size={60} /></div>
          <div class="info">
            <span class="overline mono">
              {kind === 'update' ? `${info.installedVersion} → ${info.version}` : t('version', { v: kind === 'uninstall' && info.installedVersion ? info.installedVersion : info.version })}
            </span>
            <h1>{t('title.' + kind)}</h1>
            <p>{kind === 'uninstall' ? t('uninstallSub') : kind === 'update' ? t('updateSub') : t('installSub')}</p>
          </div>
        </section>

        <fieldset class="options" disabled={step === 'working'}>
          {#if kind === 'uninstall'}
            <div class="row">
              <div class="text">
                <b>{t('removeData')}</b>
                <p class:warn={removeData}>{t('removeDataHint', { dir: info.dataDir })}</p>
              </div>
              <input type="checkbox" class="toggle" bind:checked={removeData} aria-label={t('removeData')} />
            </div>
          {:else}
            <div class="row">
              <div class="text">
                <b>{t('folder')}</b>
                <p class="mono path selectable" title={dir}>{dir}</p>
              </div>
              {#if kind === 'install'}
                <button onclick={pick}><Icon name="folder-open" />{t('change')}</button>
              {/if}
            </div>
            <div class="row">
              <div class="text"><b>{t('desktop')}</b><p>{t('desktopHint')}</p></div>
              <input type="checkbox" class="toggle" bind:checked={desktop} aria-label={t('desktop')} />
            </div>
          {/if}
        </fieldset>

        {#if error}
          <div class="alert"><Icon name="circle-alert" /><span>{error}</span></div>
        {/if}

        <footer>
          {#if step === 'working'}
            <div class="bar-btn" role="progressbar" aria-valuenow={percent} aria-valuemin="0" aria-valuemax="100">
              <div class="fill" class:indeterminate={percent === null} style="width:{percent ?? 30}%"></div>
              {#snippet label()}
                {#if kind === 'uninstall'}<Icon name="loader-circle" size={14} spin />{t('uninstalling')}
                {:else}{t('installing')}{#if percent !== null}<span class="mono">{percent}%</span>{/if}{/if}
              {/snippet}
              <span>{@render label()}</span>
              {#if percent !== null}
                <!-- The same label in dark ink, clipped to the filled part. -->
                <span class="ink" aria-hidden="true" style="clip-path: inset(0 {100 - percent}% 0 0)">{@render label()}</span>
              {/if}
            </div>
          {:else}
            <button class="subtle lg" onclick={Quit}>{t('cancel')}</button>
            {#if kind === 'uninstall'}
              <button class="danger lg main" onclick={run}><Icon name="trash-2" />{error ? t('retry') : t('uninstall')}</button>
            {:else}
              <div class="glow">
                <button class="primary lg main" onclick={run}><Icon name="download" />{error ? t('retry') : t(kind)}</button>
              </div>
            {/if}
          {/if}
        </footer>
      {/if}
    </main>
  {/if}
</div>

<style>
  .window { height: 100%; display: flex; flex-direction: column; background: var(--bg-sidebar); }

  /* ---- title bar, as in the launcher ---- */
  .titlebar { flex: none; height: var(--titlebar-h); display: flex; align-items: center; }
  .brand { display: flex; align-items: center; gap: var(--sp-2); padding-left: var(--sp-5); font-size: var(--fs-12); font-weight: var(--fw-medium); color: var(--text-secondary); }
  .buttons { margin-left: auto; display: flex; height: 100%; }
  .buttons button { width: var(--titlebar-btn-w); height: 100%; padding: 0; background: transparent; border: 0; border-radius: 0; color: var(--text-secondary); }
  .buttons button:hover:not(:disabled) { background: var(--bg-surface-3); color: var(--text-primary); }
  .buttons button:active:not(:disabled) { transform: none; }
  .buttons .close:hover:not(:disabled) { background: var(--close-hover); color: var(--text-primary); }

  /* ---- content sheet ---- */
  .content {
    flex: 1; min-height: 0; display: flex; flex-direction: column;
    padding: var(--sp-6) var(--sp-7);
    background: var(--bg-surface-1); border-top: 1px solid var(--border-subtle);
    animation: enter var(--dur-base) var(--ease-out);
  }
  @keyframes enter { from { opacity: 0; transform: translateY(4px); } }

  .hero {
    position: relative; flex: none; display: flex; align-items: center; gap: var(--sp-5);
    padding: var(--sp-5) var(--sp-6);
    border-radius: var(--r-lg); background: var(--bg-surface-2); border: 1px solid var(--border-subtle);
  }
  .hero::before {
    /* The launcher's faint pixel pattern. */
    content: ''; position: absolute; inset: 0; border-radius: inherit; pointer-events: none; opacity: 0.035;
    background: repeating-conic-gradient(var(--text-primary) 0 25%, transparent 0 50%) 0 0 / 16px 16px;
    mask-image: linear-gradient(90deg, transparent, var(--text-primary) 85%);
  }
  .mark { position: relative; flex: none; width: 84px; height: 84px; display: grid; place-items: center; border-radius: var(--r-md); background: var(--bg-surface-3); }
  .info { position: relative; min-width: 0; display: grid; gap: var(--sp-1); }
  .info .overline { color: var(--accent-400); }
  .info h1 { font-size: var(--fs-24); font-weight: var(--fw-bold); letter-spacing: var(--tracking-tight); line-height: var(--lh-tight); }
  .info p { color: var(--text-secondary); font-size: var(--fs-12); }

  .options { border: 0; padding: 0; margin: var(--sp-3) 0 0; min-width: 0; }
  .options:disabled { opacity: 0.6; }
  .row { display: flex; align-items: center; justify-content: space-between; gap: var(--sp-6); min-height: 56px; padding: var(--sp-2) 0; border-bottom: 1px solid var(--border-subtle); }
  .text { min-width: 0; }
  .text b { font-weight: var(--fw-medium); }
  .text p { font-size: var(--fs-12); color: var(--text-secondary); }
  .text p.warn { color: var(--danger); }
  .path { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  .alert { margin-top: var(--sp-4); align-items: center; }

  footer { margin-top: auto; display: flex; justify-content: flex-end; align-items: center; gap: var(--sp-2); padding-top: var(--sp-4); }
  .main { min-width: 160px; }
  .glow { display: flex; border-radius: var(--r-sm); box-shadow: var(--accent-glow); transition: box-shadow var(--dur-fast); }
  .glow:hover { box-shadow: var(--accent-glow-hover); }
  .glow button.primary:hover:not(:disabled) { background: var(--accent-300); }
  .glow button.primary:active:not(:disabled) { background: var(--accent-500); }

  /* Progress drawn like the launcher's Play button while it downloads. */
  .bar-btn {
    position: relative; flex: 1; height: var(--control-h-lg); display: grid; place-items: center; overflow: hidden;
    border-radius: var(--r-sm); background: var(--bg-surface-3);
    font-weight: var(--fw-semibold);
  }
  .bar-btn .fill { position: absolute; left: 0; top: 0; bottom: 0; background: var(--accent-400); transition: width var(--dur-slow); }
  .bar-btn .fill.indeterminate { animation: slide var(--dur-indeterminate) var(--ease-inout) infinite; }
  .bar-btn span { grid-area: 1 / 1; position: relative; display: inline-flex; align-items: center; justify-content: center; gap: var(--sp-2); color: var(--text-primary); }
  .bar-btn .ink { grid-area: auto; position: absolute; inset: 0; color: var(--text-on-accent); transition: clip-path var(--dur-slow); }

  .done { margin: auto; display: grid; justify-items: center; text-align: center; gap: var(--sp-2); max-width: 400px; }
  .done .glyph { width: 64px; height: 64px; display: grid; place-items: center; margin-bottom: var(--sp-2); border-radius: var(--r-md); background: var(--success-a12); color: var(--success); }
  .done h1 { font-size: var(--fs-20); font-weight: var(--fw-semibold); letter-spacing: var(--tracking-tight); }
  .done p { color: var(--text-secondary); word-break: break-word; }
</style>
