# Rabbit

Лёгкий минималистичный лаунчер Minecraft для Windows на Go + Wails v2 и Svelte.

- Вход через **Microsoft** для лицензионной игры, через **Ely.by** в браузере (OAuth2, лаунчер не видит пароль) или просто по нику
- Скины Ely.by в игре через [authlib-injector](https://github.com/yushijinhun/authlib-injector), в том числе для офлайн-ника, если он зарегистрирован на Ely.by
- Загрузчики: **Vanilla, Fabric, Quilt, Forge, NeoForge**
- Библиотека [Modrinth](https://modrinth.com) прямо в лаунчере: моды, ресурспаки, шейдеры и модпаки; обязательные зависимости ставятся сами
- Импорт и экспорт модпаков в формате `.mrpack`
- Сборки с отдельными папками `mods`, `resourcepacks`, `shaderpacks`, `saves`, `config`
- Общие `assets` и `libraries` для всех сборок, файлы не скачиваются повторно
- Java подбирается и скачивается автоматически под версию игры (рантаймы Mojang)
- Тёмный интерфейс со своим заголовком окна, выбор главного цвета и языка (русский, English, українська; по умолчанию язык системы)

Данные хранятся в `%APPDATA%\Rabbit`.

## Сборка

Нужны Go 1.25+, Node.js, pnpm и Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

```powershell
./build.ps1
```

Готовый файл: `build/bin/Rabbit.exe`.

Ключи для входа через Microsoft и Ely.by берутся из переменных окружения или из файла `build.local` рядом с `build.ps1` (он в `.gitignore`):

```
MS_CLIENT_ID=...
ELY_CLIENT_ID=...
ELY_CLIENT_SECRET=...
```

Без ключей Ely.by лаунчер показывает обычную форму логина и пароля, без `MS_CLIENT_ID` вход через Microsoft недоступен.

### Microsoft

Лаунчер входит по коду устройства (device code flow): показывает код, открывает страницу Microsoft, а затем получает токены Xbox Live и Minecraft. Нужно своё приложение Azure:

1. На <https://portal.azure.com> откройте **App registrations → New registration**, тип аккаунтов **Personal Microsoft accounts only**, Redirect URI не нужен.
2. В **Authentication** включите **Allow public client flows**.
3. Отправьте Application (client) ID на одобрение Mojang: <https://aka.ms/mce-reviewappid>. Пока приложение не одобрено, Minecraft API отвечает отказом и лаунчер пишет об этом.
4. Запишите ID в `build.local` как `MS_CLIENT_ID`.

Client ID не секретный, секрета у приложения нет.

### Ely.by

1. Создайте приложение на <https://account.ely.by/dev/applications/new>, тип **Website**.
2. Redirect URI: `http://127.0.0.1:25585/callback`.
3. Запишите данные приложения в `build.local` как `ELY_CLIENT_ID` и `ELY_CLIENT_SECRET`.

## Forge и NeoForge

Rabbit запускает официальные установщики Forge и NeoForge (`--installClient`). Если они вам полезны, поддержите их авторов: [Forge](https://www.patreon.com/LexManos/), [NeoForge](https://opencollective.com/neoforged).

## Структура

```
app.go              методы, доступные интерфейсу
content.go          контент Modrinth и модпаки
internal/auth       офлайн-аккаунты, Microsoft, Ely.by (OAuth и пароль), authlib-injector
internal/download   параллельная загрузка с проверкой sha1/sha256/sha512
internal/java       Java-рантаймы Mojang
internal/launch     аргументы запуска и процесс игры
internal/modrinth   поиск и установка контента Modrinth, .mrpack
internal/mojang     версии, установка, Fabric/Quilt, Forge/NeoForge
internal/msg        коды ошибок, которые переводит интерфейс
internal/paths      расположение файлов лаунчера
internal/store      настройки, аккаунты, сборки
frontend/           интерфейс на Svelte 5; токены дизайна в src/styles/tokens.css
```

## Лицензия

MIT. Шрифты [Inter](https://github.com/rsms/inter) и [JetBrains Mono](https://github.com/JetBrains/JetBrainsMono) распространяются по SIL Open Font License, иконки [Lucide](https://lucide.dev) по ISC.
