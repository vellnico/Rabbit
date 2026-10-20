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

<Modal title={t('settings')} onclose={cancel} width={470}>
  <span class="label">{t('accent')}</span>
  <div class="dyes">
    {#each Object.entries(accents) as [name, hex] (name)}
      <button class="slot dye" class:on={accent === name} aria-pressed={accent === name} onclick={() => setAccent(name)} aria-label={name} title={name}>
        <span class="px" style="--c:{hex}"></span>
      </button>
    {/each}
  </div>

  <label for="s-lang">{t('language')}</label>
  <select id="s-lang" value={language} onchange={(e) => setLang(e.currentTarget.value)}>
    <option value="">{t('systemLanguage')}</option>
    {#each languages as l (l.code)}<option value={l.code}>{l.name}</option>{/each}
  </select>

  <!-- A game-style slider: the value is written on the track. -->
  <label for="s-mem">{t('memory')}</label>
  <div class="slider">
    <input id="s-mem" type="range" min="1024" {max} step="512" bind:value={memory} style="--p:{(memory - 1024) / Math.max(1, max - 1024)}" />
    <span>{(memory / 1024).toFixed(1)} {t('gb')}</span>
  </div>

  <label class="check"><input type="checkbox" bind:checked={closeOnLaunch} /> {t('hideOnLaunch')}</label>

  <label for="s-jvm">{t('jvmArgs')}</label>
  <input id="s-jvm" bind:value={jvmArgs} placeholder="-XX:+UseG1GC" spellcheck="false" />

  <p class="hint">{t('dataDir')}: {app.dataDir}</p>
  {#if error}<p class="error-text">{error}</p>{/if}
  <div class="actions">
    <button class="flat" onclick={cancel}>{t('cancel')}</button>
    <button class="primary" onclick={save}>{t('save')}</button>
  </div>
</Modal>

<style>
  .label { display: block; font-size: 13px; margin: 8px 0 5px; }
  .dyes { display: flex; flex-wrap: wrap; gap: 4px; }
  .dye { width: 36px; height: 36px; padding: 0; border: none; display: grid; place-items: center; }
  .dye span {
    width: 20px; height: 20px; background: var(--c);
    box-shadow: inset 2px 2px 0 color-mix(in srgb, var(--c) 55%, #fff), inset -2px -2px 0 color-mix(in srgb, var(--c) 60%, #000);
  }
  .dye:hover:not(:disabled) { background: #9f9f9f; outline: none; }
  .dye.on { outline: 2px solid #fff; outline-offset: -2px; }
  .dye:active:not(:disabled) { padding: 0; }

  .slider { position: relative; height: 40px; }
  .slider input {
    -webkit-appearance: none; appearance: none; width: 100%; height: 100%; margin: 0; padding: 0;
    background: #1b1b1b; border: 2px solid #000; box-shadow: inset 2px 2px 0 #373737; cursor: pointer;
  }
  .slider input::-webkit-slider-thumb {
    -webkit-appearance: none; width: 16px; height: 36px;
    background: #8b8b8b; border: 2px solid #000; box-shadow: inset 2px 2px 0 #b5b5b5, inset -2px -2px 0 #5a5a5a;
  }
  .slider input:hover::-webkit-slider-thumb { outline: 2px solid var(--accent); outline-offset: -4px; }
  .slider span {
    position: absolute; inset: 0; display: grid; place-items: center; pointer-events: none;
    font: 16px var(--pixel); color: #fff; text-shadow: 2px 2px 0 #3f3f3f;
  }
  .check { display: flex; align-items: center; gap: 8px; cursor: pointer; margin-top: 14px; }
  .hint { font-size: 12px; color: #555; margin: 14px 0 0; user-select: text; word-break: break-all; }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 16px; }
</style>
