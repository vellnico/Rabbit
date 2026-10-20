# Rabbit

Лёгкий минималистичный лаунчер Minecraft для Windows на Go + Wails v2 и Svelte.

- Вход через **Ely.by** (с поддержкой 2FA) или просто по нику
- Скины Ely.by в игре через [authlib-injector](https://github.com/yushijinhun/authlib-injector) — в том числе для офлайн-ника, если он зарегистрирован на Ely.by
- Сборки с отдельными папками `mods`, `resourcepacks`, `shaderpacks`, `saves`, `config`
- Vanilla и Fabric
- Общие `assets` и `libraries` для всех сборок — файлы не скачиваются повторно
- Java подбирается и скачивается автоматически под версию игры (рантаймы Mojang)

Данные хранятся в `%APPDATA%\Rabbit`.

## Сборка

Нужны Go 1.25+, Node.js и pnpm, Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

```bash
wails build -clean -trimpath -ldflags "-s -w"
```

Готовый файл: `build/bin/Rabbit.exe`.

## Структура

```
app.go              методы, доступные интерфейсу
internal/auth       офлайн-аккаунты, Ely.by, authlib-injector
internal/download   параллельная загрузка с проверкой sha1/sha256
internal/java       Java-рантаймы Mojang
internal/launch     аргументы запуска и процесс игры
internal/mojang     манифест версий, установка, Fabric
internal/store      настройки, аккаунты, сборки
frontend/           интерфейс на Svelte 5
```

## Лицензия

MIT
