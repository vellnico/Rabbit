# Rabbit

Лёгкий минималистичный лаунчер Minecraft для Windows на Go + Wails v2 и Svelte.

- Вход через **Ely.by** в браузере (OAuth2, лаунчер не видит пароль) или просто по нику
- Скины Ely.by в игре через [authlib-injector](https://github.com/yushijinhun/authlib-injector), в том числе для офлайн-ника, если он зарегистрирован на Ely.by
- Загрузчики: **Vanilla, Fabric, Quilt, Forge, NeoForge**
- Сборки с отдельными папками `mods`, `resourcepacks`, `shaderpacks`, `saves`, `config`
- Общие `assets` и `libraries` для всех сборок, файлы не скачиваются повторно
- Java подбирается и скачивается автоматически под версию игры (рантаймы Mojang)
- Выбор главного цвета и языка интерфейса (русский, English, українська; по умолчанию язык системы)

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

### Ely.by OAuth

Для входа через браузер нужно приложение Ely.by:

1. Создайте приложение на <https://account.ely.by/dev/applications/new>, тип **Website**.
2. Redirect URI: `http://127.0.0.1:25585/callback`.
3. Положите данные приложения в файл `elyby.local` рядом с `build.ps1` (он в `.gitignore`):

   ```
   ELY_CLIENT_ID=...
   ELY_CLIENT_SECRET=...
   ```

Без этих данных лаунчер показывает обычную форму логина и пароля.

## Forge и NeoForge

Rabbit запускает официальные установщики Forge и NeoForge (`--installClient`). Если они вам полезны, поддержите их авторов: [Forge](https://www.patreon.com/LexManos/), [NeoForge](https://opencollective.com/neoforged).

## Структура

```
app.go              методы, доступные интерфейсу
internal/auth       офлайн-аккаунты, Ely.by (OAuth и пароль), authlib-injector
internal/download   параллельная загрузка с проверкой sha1/sha256
internal/java       Java-рантаймы Mojang
internal/launch     аргументы запуска и процесс игры
internal/mojang     версии, установка, Fabric/Quilt, Forge/NeoForge
internal/msg        коды ошибок, которые переводит интерфейс
internal/store      настройки, аккаунты, сборки
frontend/           интерфейс на Svelte 5
```

## Лицензия

MIT
