<script>
  import { untrack } from 'svelte'
  import { t, tErr, languages, setLanguage } from './i18n.svelte.js'
  import { accents, applyAccent } from './theme.js'
  import { SaveSettings, SetAppearance } from '../../wailsjs/go/main/App.js'

  let { app, onchange } = $props()

  // Every change is saved right away.
  const init = untrack(() => ({ ...app.settings, maxMemory: app.maxMemory }))
  const min = 1024
  const max = Math.max(min, Math.floor((init.maxMemory * 0.75) / 512) * 512)
  let memory = $state(Math.min(init.memoryMb, max))
  let closeOnLaunch = $state(init.closeOnLaunch)
  let jvmArgs = $state(init.jvmArgs ?? '')
  let accent = $state(init.accent || 'yellow')
  let language = $state(init.language ?? '')
  let error = $state('')

  function setAccent(name) {
    accent = name
    applyAccent(name)
    SetAppearance(accent, language)
  }
  function setLang(code) {
    language = code
    setLanguage(code)
    SetAppearance(accent, language)
  }

  async function save() {
    error = ''
    try {
      await SaveSettings(memory, closeOnLaunch, jvmArgs)
      await onchange()
    } catch (e) {
      error = tErr(e)
    }
  }
</script>

<header class="head"><h1>{t('settings')}</h1></header>

<div class="page">
  <h2 class="overline">{t('section.appearance')}</h2>
  <div class="row">
    <div class="text"><b>{t('language')}</b><p>{t('languageHint')}</p></div>
    <select class="control" value={language} onchange={(e) => setLang(e.currentTarget.value)} aria-label={t('language')}>
      <option value="">{t('systemLanguage')}</option>
      {#each languages as l (l.code)}<option value={l.code}>{l.name}</option>{/each}
    </select>
  </div>
  <div class="row">
    <div class="text"><b>{t('accent')}</b><p>{t('accentHint')}</p></div>
    <div class="colors" role="radiogroup" aria-label={t('accent')}>
      {#each Object.entries(accents) as [name, hex] (name)}
        <button class="color" class:on={accent === name} role="radio" aria-checked={accent === name} style="--c:{hex}" onclick={() => setAccent(name)} aria-label={name} title={name}></button>
      {/each}
    </div>
  </div>

  <h2 class="overline">{t('section.game')}</h2>
  <div class="row">
    <div class="text"><b>{t('memory')}</b><p>{t('memoryHint')}</p></div>
    <div class="mem">
      <input type="range" {min} {max} step="512" bind:value={memory} onchange={save} aria-label={t('memory')} style="--fill:{((memory - min) / Math.max(1, max - min)) * 100}%" />
      <span class="chip mono">{memory} MB</span>
    </div>
  </div>
  <div class="row">
    <div class="text"><b>{t('jvmArgs')}</b><p>{t('jvmHint')}</p></div>
    <input class="control mono" bind:value={jvmArgs} onchange={save} placeholder="-XX:+UseG1GC" spellcheck="false" aria-label={t('jvmArgs')} />
  </div>

  <h2 class="overline">{t('section.launcher')}</h2>
  <div class="row">
    <div class="text"><b>{t('hideOnLaunch')}</b><p>{t('hideHint')}</p></div>
    <input type="checkbox" class="toggle" bind:checked={closeOnLaunch} onchange={save} aria-label={t('hideOnLaunch')} />
  </div>
  <div class="row">
    <div class="text"><b>{t('dataDir')}</b><p>{t('dataDirHint')}</p></div>
    <span class="path mono selectable">{app.dataDir}</span>
  </div>
  {#if error}<p class="error-text">{error}</p>{/if}
</div>

<style>
  .head { display: flex; align-items: center; min-height: var(--control-h-md); margin-bottom: var(--sp-6); }
  h1 { font-size: var(--fs-24); font-weight: var(--fw-semibold); letter-spacing: var(--tracking-tight); line-height: var(--lh-tight); }
  .page { max-width: var(--content-max); }
  h2 { margin: var(--sp-6) 0 var(--sp-1); }
  h2:first-child { margin-top: 0; }
  .row { display: flex; align-items: center; justify-content: space-between; gap: var(--sp-6); min-height: 56px; padding: var(--sp-2) 0; border-bottom: 1px solid var(--border-subtle); }
  .text { min-width: 0; }
  .text b { font-weight: var(--fw-medium); }
  .text p { font-size: var(--fs-12); color: var(--text-secondary); }
  .control { width: 240px; flex: none; }
  .colors { display: flex; gap: var(--sp-2); }
  .color { width: 20px; height: 20px; padding: 0; border-radius: var(--r-full); background: var(--c); border: 1px solid var(--border-default); }
  .color:hover:not(:disabled) { background: var(--c); }
  .color.on { box-shadow: 0 0 0 2px var(--bg-surface-1), 0 0 0 4px var(--text-secondary); }
  .mem { width: 300px; flex: none; display: flex; align-items: center; gap: var(--sp-3); }
  .mem input { flex: 1; }
  .chip { flex: none; min-width: 80px; padding: var(--sp-1) var(--sp-2); text-align: right; font-size: var(--fs-12); color: var(--text-secondary); background: var(--bg-surface-2); border: 1px solid var(--border-subtle); border-radius: var(--r-xs); }
  .path { max-width: 320px; font-size: var(--fs-12); color: var(--text-secondary); text-align: right; word-break: break-all; }
</style>
