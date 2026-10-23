// Setup strings. The language follows the launcher settings, else the system.
const ru = {
  'title.install': 'Установка Rabbit',
  'title.update': 'Обновление Rabbit',
  'title.uninstall': 'Удаление Rabbit',
  installSub: 'Лёгкий лаунчер Minecraft. Ставится для текущего пользователя, права администратора не нужны.',
  updateSub: 'Сборки, миры и аккаунты останутся на месте.',
  uninstallSub: 'Будут удалены лаунчер, его ярлыки и запись в «Приложениях».',
  version: 'Версия {v}',
  folder: 'Папка установки',
  change: 'Изменить',
  desktop: 'Ярлык на рабочем столе',
  desktopHint: 'Ярлык в меню «Пуск» добавляется всегда',
  removeData: 'Удалить данные лаунчера',
  removeDataHint: 'Сборки, миры, аккаунты и скачанные файлы в {dir}',
  install: 'Установить',
  update: 'Обновить',
  uninstall: 'Удалить',
  installing: 'Установка',
  uninstalling: 'Удаление…',
  cancel: 'Отмена',
  close: 'Закрыть',
  minimize: 'Свернуть',
  launch: 'Запустить',
  retry: 'Повторить',
  'done.install': 'Rabbit установлен',
  'done.update': 'Rabbit обновлён',
  'done.uninstall': 'Rabbit удалён',
  'doneSub.install': 'Лаунчер появился в меню «Пуск».',
  'doneSub.uninstallKept': 'Данные лаунчера остались в {dir}.',
  'doneSub.uninstallAll': 'Данные лаунчера тоже удалены.',
  'err.running': 'Rabbit сейчас запущен. Закройте его и попробуйте снова.',
  'err.dataInUse': 'Не все данные удалось удалить. Закройте игру и удалите папку вручную',
  'err.notInstalled': 'Rabbit не установлен.',
  'err.shortcut': 'Не удалось создать ярлык',
}

const en = {
  'title.install': 'Install Rabbit',
  'title.update': 'Update Rabbit',
  'title.uninstall': 'Uninstall Rabbit',
  installSub: 'A lightweight Minecraft launcher. Installs for the current user, no admin rights needed.',
  updateSub: 'Instances, worlds and accounts stay in place.',
  uninstallSub: 'The launcher, its shortcuts and its entry in Apps will be removed.',
  version: 'Version {v}',
  folder: 'Install folder',
  change: 'Change',
  desktop: 'Desktop shortcut',
  desktopHint: 'A Start menu shortcut is always added',
  removeData: 'Delete launcher data',
  removeDataHint: 'Instances, worlds, accounts and downloaded files in {dir}',
  install: 'Install',
  update: 'Update',
  uninstall: 'Uninstall',
  installing: 'Installing',
  uninstalling: 'Uninstalling…',
  cancel: 'Cancel',
  close: 'Close',
  minimize: 'Minimize',
  launch: 'Launch',
  retry: 'Retry',
  'done.install': 'Rabbit is installed',
  'done.update': 'Rabbit is updated',
  'done.uninstall': 'Rabbit is uninstalled',
  'doneSub.install': 'You will find the launcher in the Start menu.',
  'doneSub.uninstallKept': 'Launcher data is kept in {dir}.',
  'doneSub.uninstallAll': 'Launcher data was deleted too.',
  'err.running': 'Rabbit is running. Close it and try again.',
  'err.dataInUse': 'Some data could not be deleted. Close the game and delete the folder by hand',
  'err.notInstalled': 'Rabbit is not installed.',
  'err.shortcut': 'Could not create the shortcut',
}

const uk = {
  'title.install': 'Встановлення Rabbit',
  'title.update': 'Оновлення Rabbit',
  'title.uninstall': 'Видалення Rabbit',
  installSub: 'Легкий лаунчер Minecraft. Встановлюється для поточного користувача, права адміністратора не потрібні.',
  updateSub: 'Збірки, світи та акаунти залишаться на місці.',
  uninstallSub: 'Буде видалено лаунчер, його ярлики та запис у «Програмах».',
  version: 'Версія {v}',
  folder: 'Тека встановлення',
  change: 'Змінити',
  desktop: 'Ярлик на робочому столі',
  desktopHint: 'Ярлик у меню «Пуск» додається завжди',
  removeData: 'Видалити дані лаунчера',
  removeDataHint: 'Збірки, світи, акаунти та завантажені файли в {dir}',
  install: 'Встановити',
  update: 'Оновити',
  uninstall: 'Видалити',
  installing: 'Встановлення',
  uninstalling: 'Видалення…',
  cancel: 'Скасувати',
  close: 'Закрити',
  minimize: 'Згорнути',
  launch: 'Запустити',
  retry: 'Повторити',
  'done.install': 'Rabbit встановлено',
  'done.update': 'Rabbit оновлено',
  'done.uninstall': 'Rabbit видалено',
  'doneSub.install': 'Лаунчер з’явився в меню «Пуск».',
  'doneSub.uninstallKept': 'Дані лаунчера залишилися в {dir}.',
  'doneSub.uninstallAll': 'Дані лаунчера теж видалено.',
  'err.running': 'Rabbit зараз запущено. Закрийте його та спробуйте знову.',
  'err.dataInUse': 'Не всі дані вдалося видалити. Закрийте гру та видаліть теку вручну',
  'err.notInstalled': 'Rabbit не встановлено.',
  'err.shortcut': 'Не вдалося створити ярлик',
}

const dicts = { ru, en, uk }

function systemLanguage() {
  for (const l of navigator.languages ?? [navigator.language]) {
    const code = l.slice(0, 2).toLowerCase()
    if (dicts[code]) return code
    if (code === 'be' || code === 'kk') return 'ru'
  }
  return 'en'
}

export const i18n = $state({ lang: systemLanguage() })

/** setLanguage('') follows the system language. */
export function setLanguage(code) {
  i18n.lang = dicts[code] ? code : systemLanguage()
  document.documentElement.lang = i18n.lang
}

export function t(key, params) {
  let s = dicts[i18n.lang][key] ?? en[key] ?? key
  if (params) for (const [k, v] of Object.entries(params)) s = s.replaceAll(`{${k}}`, v)
  return s
}

/** Translates a backend error: "@key|detail" or plain text. */
export function tErr(e) {
  const s = String(e?.message ?? e)
  if (!s.startsWith('@')) return s
  const i = s.indexOf('|')
  const key = i < 0 ? s.slice(1) : s.slice(1, i)
  return t('err.' + key) + (i < 0 ? '' : ': ' + s.slice(i + 1))
}
