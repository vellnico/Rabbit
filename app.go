package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/vellnico/rabbit/internal/auth"
	"github.com/vellnico/rabbit/internal/java"
	"github.com/vellnico/rabbit/internal/launch"
	"github.com/vellnico/rabbit/internal/mojang"
	"github.com/vellnico/rabbit/internal/msg"
	"github.com/vellnico/rabbit/internal/paths"
	"github.com/vellnico/rabbit/internal/store"
)

// App is bound to the frontend; every exported method is callable from JS.
// Errors are either plain text or "@key|detail" codes the frontend translates.
type App struct {
	ctx context.Context

	mu       sync.Mutex
	settings store.Settings
	accounts []auth.Account
	running  map[string]*session
	login    context.CancelFunc // pending browser login
}

type session struct {
	cancel context.CancelFunc
	proc   *launch.Process
}

func NewApp() *App { return &App{running: map[string]*session{}} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	os.MkdirAll(paths.Root(), 0o755)
	a.settings = store.LoadSettings()
	a.accounts = store.LoadAccounts()
}

// AccountView is an account without secrets.
type AccountView struct {
	ID      string    `json:"id"`
	Kind    auth.Kind `json:"kind"`
	Name    string    `json:"name"`
	UUID    string    `json:"uuid"`
	ElySkin bool      `json:"elySkin"`
	Skin    string    `json:"skin,omitempty"`
}

type State struct {
	Settings   store.Settings   `json:"settings"`
	Accounts   []AccountView    `json:"accounts"`
	Instances  []store.Instance `json:"instances"`
	Running    []string         `json:"running"`
	MaxMemory  int              `json:"maxMemory"`
	DataDir    string           `json:"dataDir"`
	ElyBrowser bool             `json:"elyBrowser"` // OAuth login available
	Microsoft  bool             `json:"microsoft"`  // Microsoft login available
}

func (a *App) GetState() State {
	a.mu.Lock()
	defer a.mu.Unlock()
	s := State{
		Settings:   a.settings,
		Instances:  store.LoadInstances(),
		MaxMemory:  totalMemoryMB(),
		DataDir:    paths.Root(),
		ElyBrowser: auth.OAuthConfigured(),
		Microsoft:  auth.MSConfigured(),
	}
	for _, acc := range a.accounts {
		s.Accounts = append(s.Accounts, AccountView{acc.ID, acc.Kind, acc.Name, acc.UUID, acc.ElySkin, acc.Skin})
	}
	for id := range a.running {
		s.Running = append(s.Running, id)
	}
	return s
}

// ---- versions & instances ----

type VersionInfo struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	ReleaseTime string `json:"releaseTime"`
}

func (a *App) ListVersions() ([]VersionInfo, error) {
	m, err := mojang.FetchManifest(a.ctx)
	if err != nil {
		return nil, msg.Wrap("versionsUnavailable", err)
	}
	out := make([]VersionInfo, 0, len(m.Versions))
	for _, v := range m.Versions {
		out = append(out, VersionInfo{v.ID, v.Type, v.ReleaseTime.Format("2006-01-02")})
	}
	return out, nil
}

// ListLoaderVersions returns versions of a mod loader for a game version,
// newest first.
func (a *App) ListLoaderVersions(loader, game string) ([]string, error) {
	switch loader {
	case "forge":
		return mojang.ForgeVersions(a.ctx, game)
	case "neoforge":
		return mojang.NeoForgeVersions(a.ctx, game)
	case "fabric", "quilt":
		return mojang.ProfileLoaderVersions(a.ctx, loader, game)
	}
	return nil, nil
}

func validLoader(l string) bool {
	return l == "vanilla" || mojang.IsProfileLoader(l) || mojang.IsInstallerLoader(l)
}

func (a *App) CreateInstance(name, game, loader, loaderVersion string) (store.Instance, error) {
	if game == "" {
		return store.Instance{}, msg.E("pickVersion")
	}
	if !validLoader(loader) {
		loader = "vanilla"
	}
	if loader == "vanilla" {
		loaderVersion = ""
	} else if loaderVersion == "" {
		list, err := a.ListLoaderVersions(loader, game)
		if err != nil || len(list) == 0 {
			return store.Instance{}, msg.E("loaderUnavailable")
		}
		loaderVersion = list[0]
	}
	in, err := store.CreateInstance(store.Instance{Name: name, GameVersion: game, Loader: loader, LoaderVersion: loaderVersion})
	if err == nil {
		a.SelectInstance(in.ID)
	}
	return in, err
}

func (a *App) DeleteInstance(id string) error {
	a.mu.Lock()
	_, busy := a.running[id]
	a.mu.Unlock()
	if busy {
		return msg.E("closeGameFirst")
	}
	return store.DeleteInstance(id)
}

func (a *App) OpenInstanceFolder(id string, sub string) error {
	dir := paths.Instance(id)
	if sub != "" && !strings.ContainsAny(sub, `/\.`) {
		dir += `\` + sub
		os.MkdirAll(dir, 0o755)
	}
	return exec.Command("explorer.exe", dir).Start()
}

func (a *App) SelectInstance(id string) {
	a.mu.Lock()
	a.settings.SelectedInstance = id
	store.SaveSettings(a.settings)
	a.mu.Unlock()
}

func (a *App) SaveSettings(memoryMB int, closeOnLaunch bool, jvmArgs string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if memoryMB < 512 {
		memoryMB = 512
	}
	a.settings.MemoryMB = memoryMB
	a.settings.CloseOnLaunch = closeOnLaunch
	a.settings.JVMArgs = strings.TrimSpace(jvmArgs)
	return store.SaveSettings(a.settings)
}

// SetAppearance stores the accent color and language ("" = system).
func (a *App) SetAppearance(accent, language string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.settings.Accent = accent
	a.settings.Language = language
	return store.SaveSettings(a.settings)
}

// ---- accounts ----

func (a *App) addAccount(acc *auth.Account) {
	a.mu.Lock()
	defer a.mu.Unlock()
	list := a.accounts[:0]
	for _, x := range a.accounts {
		if x.ID != acc.ID {
			list = append(list, x)
		}
	}
	a.accounts = append(list, *acc)
	a.settings.SelectedAccount = acc.ID
	store.SaveAccounts(a.accounts)
	store.SaveSettings(a.settings)
}

func (a *App) AddOfflineAccount(nick string) error {
	acc, err := auth.NewOffline(a.ctx, nick)
	if err != nil {
		return err
	}
	a.addAccount(acc)
	return nil
}

// loginContext starts a browser login; a new one or CancelLogin aborts it.
func (a *App) loginContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock()
	if a.login != nil {
		a.login()
	}
	a.login = cancel
	a.mu.Unlock()
	return ctx, cancel
}

// LoginElyBrowser opens Ely.by in the system browser and waits for the user
// to confirm access. CancelLogin aborts the wait.
func (a *App) LoginElyBrowser() error {
	ctx, cancel := a.loginContext()
	defer cancel()

	acc, err := auth.LoginElyBrowser(ctx, func(url string) { runtime.BrowserOpenURL(a.ctx, url) })
	runtime.WindowShow(a.ctx) // bring the launcher back from behind the browser
	if err != nil {
		return err
	}
	a.addAccount(acc)
	return nil
}

// LoginMicrosoft signs in to a Microsoft account with a licensed game. The
// code to enter is sent to the frontend as the "mscode" event and copied to
// the clipboard; the sign-in page opens in the browser.
func (a *App) LoginMicrosoft() error {
	ctx, cancel := a.loginContext()
	defer cancel()

	acc, err := auth.LoginMicrosoft(ctx, func(dc auth.DeviceCode) {
		runtime.ClipboardSetText(a.ctx, dc.Code)
		runtime.EventsEmit(a.ctx, "mscode", dc)
		runtime.BrowserOpenURL(a.ctx, dc.URL)
	})
	runtime.WindowShow(a.ctx)
	if err != nil {
		return err
	}
	a.addAccount(acc)
	return nil
}

func (a *App) CancelLogin() {
	a.mu.Lock()
	if a.login != nil {
		a.login()
		a.login = nil
	}
	a.mu.Unlock()
}

// LoginEly signs in with a password (used when the build has no OAuth
// credentials). Returns "totp" when a two-factor code is required.
func (a *App) LoginEly(login, password, totp string) (string, error) {
	acc, err := auth.LoginEly(a.ctx, login, password, totp)
	if errors.Is(err, auth.ErrTwoFactor) {
		if totp != "" {
			return "totp", msg.E("badTotp")
		}
		return "totp", nil
	}
	if err != nil {
		return "", err
	}
	a.addAccount(acc)
	return "ok", nil
}

func (a *App) RemoveAccount(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	list := a.accounts[:0]
	for _, x := range a.accounts {
		if x.ID == id {
			go auth.Invalidate(context.Background(), &x)
			continue
		}
		list = append(list, x)
	}
	a.accounts = list
	if a.settings.SelectedAccount == id {
		a.settings.SelectedAccount = ""
		if len(list) > 0 {
			a.settings.SelectedAccount = list[0].ID
		}
	}
	store.SaveAccounts(a.accounts)
	store.SaveSettings(a.settings)
}

func (a *App) SelectAccount(id string) {
	a.mu.Lock()
	a.settings.SelectedAccount = id
	store.SaveSettings(a.settings)
	a.mu.Unlock()
}

// ---- launching ----

type progressEvent struct {
	Instance string `json:"instance"`
	Stage    string `json:"stage"` // translated by the frontend
	Done     int    `json:"done"`
	Total    int    `json:"total"`
}

type stateEvent struct {
	Instance string   `json:"instance"`
	State    string   `json:"state"` // preparing | running | stopped | error
	Message  string   `json:"message,omitempty"`
	Log      []string `json:"log,omitempty"`
}

func (a *App) emitState(e stateEvent) { runtime.EventsEmit(a.ctx, "state", e) }

func (a *App) Launch(id string) error {
	in, err := store.GetInstance(id)
	if err != nil {
		return err
	}
	a.mu.Lock()
	if _, busy := a.running[id]; busy {
		a.mu.Unlock()
		return msg.E("alreadyRunning")
	}
	var acc *auth.Account
	for i := range a.accounts {
		if a.accounts[i].ID == a.settings.SelectedAccount {
			acc = &a.accounts[i]
		}
	}
	if acc == nil {
		a.mu.Unlock()
		return msg.E("addAccount")
	}
	account := *acc
	settings := a.settings
	ctx, cancel := context.WithCancel(a.ctx)
	sess := &session{cancel: cancel}
	a.running[id] = sess
	a.mu.Unlock()

	go func() {
		a.emitState(stateEvent{Instance: id, State: "preparing"})
		opts, err := a.prepare(ctx, in, &account)
		if err == nil {
			opts.MemoryMB = settings.MemoryMB
			opts.ExtraJVM = settings.JVMArgs
			var proc *launch.Process
			proc, err = launch.Start(opts, func(code int, tail []string) {
				a.finish(id, code, tail)
			})
			a.mu.Lock()
			sess.proc = proc
			a.mu.Unlock()
		}
		if err != nil {
			a.mu.Lock()
			delete(a.running, id)
			a.mu.Unlock()
			text := err.Error()
			if ctx.Err() != nil {
				text = ""
			}
			a.emitState(stateEvent{Instance: id, State: "error", Message: text})
			return
		}
		in.LastPlayed = time.Now()
		store.SaveInstance(in)
		a.emitState(stateEvent{Instance: id, State: "running"})
		if settings.CloseOnLaunch {
			runtime.WindowHide(a.ctx)
		}
	}()
	return nil
}

func (a *App) finish(id string, code int, tail []string) {
	a.mu.Lock()
	delete(a.running, id)
	a.mu.Unlock()
	runtime.WindowShow(a.ctx)
	e := stateEvent{Instance: id, State: "stopped"}
	if code != 0 && code != 1 { // 1 is also used for normal "Stop" from the launcher
		e.State = "error"
		e.Message = msg.E("crashed", strconv.Itoa(code)).Error()
		if len(tail) > 40 {
			tail = tail[len(tail)-40:]
		}
		e.Log = tail
	}
	a.emitState(e)
}

func (a *App) prepare(ctx context.Context, in store.Instance, acc *auth.Account) (launch.Options, error) {
	report := func(stage string, done, total int) {
		runtime.EventsEmit(a.ctx, "progress", progressEvent{in.ID, stage, done, total})
	}
	report("account", 0, 0)
	if acc.Kind == auth.ElyBy || acc.Kind == auth.Microsoft {
		ok, err := auth.Refresh(ctx, acc)
		service := "ely"
		if acc.Kind == auth.Microsoft {
			service = "ms"
		}
		if err != nil {
			if msg.IsCoded(err) {
				return launch.Options{}, err
			}
			return launch.Options{}, msg.Wrap(service+"Unavailable", err)
		}
		if !ok {
			return launch.Options{}, msg.E(service + "Expired")
		}
		a.mu.Lock()
		for i := range a.accounts {
			if a.accounts[i].ID == acc.ID {
				a.accounts[i] = *acc
			}
		}
		store.SaveAccounts(a.accounts)
		a.mu.Unlock()
	}

	// Vanilla first: loaders inherit from it and their installers need it.
	report("version", 0, 0)
	if err := mojang.EnsureVanilla(ctx, in.GameVersion); err != nil {
		return launch.Options{}, err
	}
	v, err := mojang.Resolve(in.GameVersion)
	if err != nil {
		return launch.Options{}, err
	}
	if err := mojang.Install(ctx, v, report); err != nil {
		return launch.Options{}, err
	}

	report("java", 0, 0)
	javaExe, err := java.Ensure(ctx, v.JavaComponent(), func(d, t int) { report("java", d, t) })
	if err != nil {
		return launch.Options{}, msg.Wrap("javaFailed", err)
	}

	if in.Loader != "vanilla" && in.Loader != "" {
		report("loader", 0, 0)
		var id string
		switch {
		case mojang.IsProfileLoader(in.Loader):
			id, err = mojang.EnsureProfileLoader(ctx, in.Loader, in.GameVersion, in.LoaderVersion)
		case mojang.IsInstallerLoader(in.Loader):
			id, err = mojang.EnsureInstallerLoader(ctx, in.Loader, in.GameVersion, in.LoaderVersion, javaExe)
		default:
			err = errors.New("unknown loader " + in.Loader)
		}
		if err != nil {
			return launch.Options{}, msg.Wrap("loaderFailed", err)
		}
		if v, err = mojang.Resolve(id); err != nil {
			return launch.Options{}, err
		}
		if err := mojang.Install(ctx, v, report); err != nil {
			return launch.Options{}, err
		}
	}

	var injector string
	if acc.Kind == auth.ElyBy || acc.ElySkin {
		report("injector", 0, 0)
		injector, err = auth.EnsureInjector(ctx)
		if err != nil {
			return launch.Options{}, err
		}
	}
	report("launch", 0, 0)
	return launch.Options{Version: v, Java: javaExe, GameDir: paths.Instance(in.ID), Account: *acc, Injector: injector}, nil
}

func (a *App) Stop(id string) {
	a.mu.Lock()
	s := a.running[id]
	var proc *launch.Process
	if s != nil {
		proc = s.proc
	}
	a.mu.Unlock()
	if s == nil {
		return
	}
	s.cancel()
	proc.Kill()
}

// ---- misc ----

type memoryStatusEx struct {
	length, memoryLoad                  uint32
	totalPhys, availPhys                uint64
	totalPageFile, availPageFile        uint64
	totalVirtual, availVirtual, extVirt uint64
}

func totalMemoryMB() int {
	var m memoryStatusEx
	m.length = uint32(unsafe.Sizeof(m))
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")
	if r, _, _ := proc.Call(uintptr(unsafe.Pointer(&m))); r == 0 {
		return 8192
	}
	return int(m.totalPhys / 1024 / 1024)
}

// startExplorer runs explorer.exe with a flag and a quoted path; explorer
// does not understand Go's argument quoting for "/select,".
func startExplorer(flag, path string) error {
	cmd := exec.Command("explorer.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `explorer.exe ` + flag + `"` + path + `"`}
	return cmd.Start()
}
