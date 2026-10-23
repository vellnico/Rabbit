# Rabbit

A lightweight, minimalist Minecraft launcher for Windows, built with Go + Wails v2 and Svelte.

- Sign in with **Ely.by** in the browser (OAuth2, the launcher never sees your password) or just play with a nickname
- Ely.by skins in game through [authlib-injector](https://github.com/yushijinhun/authlib-injector), including offline nicknames that are registered on Ely.by
- Loaders: **Vanilla, Fabric, Quilt, Forge, NeoForge**
- Built-in [Modrinth](https://modrinth.com) library: mods, resource packs, shaders and modpacks; required dependencies are installed automatically
- Import and export of modpacks in the `.mrpack` format
- Instances with their own `mods`, `resourcepacks`, `shaderpacks`, `saves` and `config` folders
- `assets` and `libraries` shared by all instances, so files are never downloaded twice
- Java is picked and downloaded automatically for each game version (Mojang runtimes)
- Dark interface with a custom title bar, a choice of accent color and language (Russian, English, Ukrainian; the system language by default)

Launcher data is stored in `%APPDATA%\Rabbit`.

## Download

Get `Rabbit-Setup.exe` from the [latest release](https://github.com/vellnico/Rabbit/releases/latest) and run it. No administrator rights are needed.

## Building

You need Go 1.25+, Node.js, pnpm and the Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

```powershell
./build.ps1
```

The result is `build/bin/Rabbit.exe`.

### Installer

```powershell
./build.ps1 -Installer
```

This produces `build/bin/Rabbit-Setup.exe`, an installer styled like the launcher (also Wails and Svelte, sharing the design tokens). It installs the launcher for the current user into `%LOCALAPPDATA%\Programs\Rabbit` without administrator rights, adds a Start menu shortcut and, optionally, a desktop shortcut, and registers the uninstaller in Apps. Running a newer setup updates the installed version. The uninstaller asks whether to delete the instances and worlds in `%APPDATA%\Rabbit`; by default they are kept.

Silent mode: `Rabbit-Setup.exe /S [/D=folder]` and `uninstall.exe /S` (launcher data is kept).

Layout: `installer/` is the Go part (a separate module; built with the `setup` tag it embeds `Rabbit.exe` and the uninstaller, without the tag it is the uninstaller itself), `frontend/installer/` is its interface. `build.ps1` reads the version from the root `wails.json` and passes it to the launcher, the installer and `installer/wails.json` (the version in the exe properties).

The Start menu shortcut is created with `IShellLink` and the application id `vellnico.Rabbit` (the launcher sets the same id at startup); the installer then notifies Explorer (`SHChangeNotify`) so the shortcut shows up in the menu right away.

### Ely.by OAuth

Signing in through the browser needs an Ely.by application:

1. Create an application at <https://account.ely.by/dev/applications/new>, type **Website**.
2. Redirect URI: `http://127.0.0.1:25585/callback`.
3. Put the application credentials into `build.local` next to `build.ps1` (it is git-ignored; the old name `elyby.local` is read too):

   ```
   ELY_CLIENT_ID=...
   ELY_CLIENT_SECRET=...
   ```

Without them the launcher shows the regular login and password form.

## Forge and NeoForge

Rabbit runs the official Forge and NeoForge installers (`--installClient`). If you find them useful, please support their authors: [Forge](https://www.patreon.com/LexManos/), [NeoForge](https://opencollective.com/neoforged).

## Structure

```
app.go              methods exposed to the interface
content.go          Modrinth content and modpacks
internal/auth       offline accounts, Ely.by (OAuth and password), authlib-injector
internal/download   parallel downloads with sha1/sha256/sha512 checks
internal/java       Mojang Java runtimes
internal/launch     launch arguments and the game process
internal/modrinth   Modrinth search and content installation, .mrpack
internal/mojang     versions, installation, Fabric/Quilt, Forge/NeoForge
internal/msg        error codes translated by the interface
internal/paths      launcher file locations
internal/store      settings, accounts, instances
frontend/           Svelte 5 interface; design tokens in src/styles/tokens.css
frontend/installer/ installer interface
installer/          installer and uninstaller
```

## License

MIT. The [Inter](https://github.com/rsms/inter) and [JetBrains Mono](https://github.com/JetBrains/JetBrainsMono) fonts are distributed under the SIL Open Font License, the [Lucide](https://lucide.dev) icons under ISC.
