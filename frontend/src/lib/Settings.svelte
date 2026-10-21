<script>
  import { untrack } from 'svelte'
  import Modal from './Modal.svelte'
  import Icon from './Icon.svelte'
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
  <label for="s-lang">{t('language')}</label>
  <select id="s-lang" value={language} onchange={(e) => setLang(e.currentTarget.value)}>
    <option value="">{t('systemLanguage')}</option>
    {#each languages as l (l.code)}<option value={l.code}>{l.name}</option>{/each}
  </select>

  <span class="label">{t('accent')}</span>
  <div class="colors" role="radiogroup">
    {#each Object.entries(accents) as [name, hex] (name)}
      <button class="color" class:on={accent === name} role="radio" aria-checked={accent === name} style="--c:{hex}" onclick={() => setAccent(name)} aria-label={name} title={name}></button>
    {/each}
  </div>

  <label for="s-mem">{t('memory')}</label>
  <div class="mem">
    <input id="s-mem" type="range" min="1024" {max} step="512" bind:value={memory} />
    <span>{(memory / 1024).toFixed(1)} {t('gb')}</span>
  </div>

  <label for="s-jvm">{t('jvmArgs')}</label>
  <input id="s-jvm" bind:value={jvmArgs} placeholder="-XX:+UseG1GC" spellcheck="false" />

  <label class="check"><input type="checkbox" bind:checked={closeOnLaunch} /> {t('hideOnLaunch')}</label>

  <p class="path">{t('dataDir')}: {app.dataDir}</p>
  {#if error}<p class="error-text">{error}</p>{/if}
  <div class="actions">
    <button onclick={cancel}>{t('cancel')}</button>
    <button class="primary" onclick={save}><Icon name="check" />{t('save')}</button>
  </div>
</Modal>

<style>
  .label { display: block; color: var(--muted); margin: 12px 0 6px; }
  .colors { display: flex; gap: 8px; }
  .color { width: 22px; height: 22px; padding: 0; border-radius: 50%; background: var(--c); border: 1px solid rgb(0 0 0 / .15); }
  .color:hover:not(:disabled) { background: var(--c); }
  .color.on { box-shadow: 0 0 0 2px var(--surface), 0 0 0 4px var(--text); }
  .mem { display: flex; align-items: center; gap: 12px; }
  .mem input { flex: 1; }
  .mem span { min-width: 56px; text-align: right; font-variant-numeric: tabular-nums; }
  .check { display: flex; align-items: center; gap: 8px; margin-top: 14px; color: var(--text); cursor: pointer; }
  .path { margin: 14px 0 0; font-size: 12px; color: var(--muted); user-select: text; word-break: break-all; }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 18px; }
</style>
