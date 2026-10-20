<script>
  import Modal from './Modal.svelte'
  import { SaveSettings } from '../../wailsjs/go/main/App.js'

  let { app, onclose, onchange } = $props()

  const max = Math.max(1024, Math.floor((app.maxMemory * 0.75) / 512) * 512)
  let memory = $state(Math.min(app.settings.memoryMb, max))
  let closeOnLaunch = $state(app.settings.closeOnLaunch)
  let jvmArgs = $state(app.settings.jvmArgs ?? '')
  let error = $state('')

  async function save() {
    try {
      await SaveSettings(memory, closeOnLaunch, jvmArgs)
      await onchange()
      onclose()
    } catch (e) {
      error = String(e)
    }
  }
</script>

<Modal title="Настройки" {onclose}>
  <label for="s-mem">Память для игры: <b>{(memory / 1024).toFixed(1)} ГБ</b></label>
  <input id="s-mem" type="range" min="1024" {max} step="512" bind:value={memory} />

  <label class="check"><input type="checkbox" bind:checked={closeOnLaunch} /> Скрывать лаунчер во время игры</label>

  <label for="s-jvm">Дополнительные аргументы JVM</label>
  <input id="s-jvm" bind:value={jvmArgs} placeholder="-XX:+UseG1GC" spellcheck="false" />

  <p class="hint">Данные: {app.dataDir}</p>
  {#if error}<p class="error">{error}</p>{/if}
  <div class="actions">
    <button class="ghost" onclick={onclose}>Отмена</button>
    <button class="primary" onclick={save}>Сохранить</button>
  </div>
</Modal>

<style>
  input[type='range'] { padding: 0; accent-color: var(--accent); border: none; }
  .check { display: flex; align-items: center; gap: 8px; cursor: pointer; margin-top: 16px; }
  .check input { width: auto; margin: 0; }
  .hint { color: var(--muted); font-size: 12px; margin: 16px 0 0; user-select: text; word-break: break-all; }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 18px; }
  .error { color: var(--danger); font-size: 13px; }
</style>
