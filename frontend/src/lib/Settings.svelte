<script>
  import { untrack } from 'svelte'
  import Modal from './Modal.svelte'
  import { t, tErr, languages, setLanguage } from './i18n.svelte.js'
  import { accents, applyAccent } from './theme.js'
  import { SaveSettings, SetAppearance } from '../../wailsjs/go/main/App.js'

  let { app, onclose, onchange } = $props()

  // The form edits a copy of the settings taken when it opens.
  const init = untrack(() => ({ ...app.settings, maxMemory: app.maxMemory }))
  const max = Math.max(1024, Math.floor((init.maxMemory * 0.75) / 512) * 512)
  let memory = $state(Math.min(init.memoryMb, max))
  let closeOnLaunch = $state(init.closeOnLaunch)
  let jvmArgs = $state(init.jvmArgs ?? '')
  let accent = $state(init.accent || 'yellow')
  let language = $state(init.language ?? '')
  let error = $state('')

  // Appearance applies live and is saved right away.
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
    try {
      await SaveSettings(memory, closeOnLaunch, jvmArgs)
      await onchange()
      onclose()
    } catch (e) {
      error = tErr(e)
    }
  }

  async function cancel() {
    await onchange()
    onclose()
  }
</script>

<Modal title={t('settings')} onclose={cancel} width={460}>
  <h4>{t('appearance')}</h4>
  <span class="label">{t('accent')}</span>
  <div class="swatches">
    {#each Object.entries(accents) as [name, hex] (name)}
      <button class="swatch" class:on={accent === name} style="--c:{hex}" onclick={() => setAccent(name)} aria-label={name} title={name}>
        <svg viewBox="0 0 24 24" width="14" height="14" aria-hidden="true"><path d="M5 12.5l4.5 4.5L19 7.5" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" /></svg>
      </button>
    {/each}
  </div>

  <label for="s-lang">{t('language')}</label>
  <select id="s-lang" value={language} onchange={(e) => setLang(e.currentTarget.value)}>
    <option value="">{t('systemLanguage')}</option>
    {#each languages as l (l.code)}<option value={l.code}>{l.name}</option>{/each}
  </select>

  <h4>{t('game')}</h4>
  <label for="s-mem">{t('memory')}: <b class="mem">{(memory / 1024).toFixed(1)} {t('gb')}</b></label>
  <input id="s-mem" type="range" min="1024" {max} step="512" bind:value={memory} style="--p:{((memory - 1024) / Math.max(1, max - 1024)) * 100}%" />

  <label class="check"><input type="checkbox" bind:checked={closeOnLaunch} /> {t('hideOnLaunch')}</label>

  <label for="s-jvm">{t('jvmArgs')}</label>
  <input id="s-jvm" bind:value={jvmArgs} placeholder="-XX:+UseG1GC" spellcheck="false" />

  <p class="hint">{t('dataDir')}: {app.dataDir}</p>
  {#if error}<p class="error-text">{error}</p>{/if}
  <div class="actions">
    <button class="ghost" onclick={cancel}>{t('cancel')}</button>
    <button class="primary" onclick={save}>{t('save')}</button>
  </div>
</Modal>

<style>
  h4 { margin: 18px 0 0; font-size: 11px; letter-spacing: .08em; text-transform: uppercase; color: var(--muted); font-weight: 600; }
  h4:first-of-type { margin-top: 12px; }
  .label { display: block; color: var(--muted); font-size: 12px; margin: 12px 0 8px; }
  .swatches { display: flex; flex-wrap: wrap; gap: 8px; }
  .swatch {
    width: 32px; height: 32px; padding: 0; border-radius: 50%;
    background: var(--c); border: 2px solid transparent;
    display: grid; place-items: center; color: #141208;
  }
  .swatch:hover:not(:disabled) { background: var(--c); transform: scale(1.1); border-color: transparent; }
  .swatch svg { opacity: 0; transform: scale(.4); transition: opacity .2s, transform .3s var(--ease); }
  .swatch.on { box-shadow: 0 0 0 3px var(--panel), 0 0 0 5px var(--c), 0 0 18px var(--c); }
  .swatch.on svg { opacity: 1; transform: scale(1); }
  .mem { color: var(--text); }
  input[type='range'] {
    -webkit-appearance: none; appearance: none; padding: 0; height: 6px; border: none; border-radius: 3px;
    background: linear-gradient(90deg, var(--accent) var(--p), var(--raised-2) var(--p));
  }
  input[type='range']::-webkit-slider-thumb {
    -webkit-appearance: none; width: 18px; height: 18px; border-radius: 50%;
    background: var(--text); border: 3px solid var(--accent); cursor: pointer;
    box-shadow: 0 0 10px rgb(var(--accent-rgb) / .6); transition: transform .15s;
  }
  input[type='range']::-webkit-slider-thumb:hover { transform: scale(1.15); }
  .check { display: flex; align-items: center; gap: 8px; cursor: pointer; margin-top: 16px; color: var(--text); font-size: 13.5px; }
  .check input { width: auto; margin: 0; }
  .hint { color: var(--muted); font-size: 12px; margin: 16px 0 0; user-select: text; word-break: break-all; }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 18px; }
</style>
