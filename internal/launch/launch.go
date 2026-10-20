// Package launch builds the Minecraft command line and starts the game.
package launch

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/vellnico/rabbit/internal/auth"
	"github.com/vellnico/rabbit/internal/mojang"
	"github.com/vellnico/rabbit/internal/paths"
)

type Options struct {
	Version  *mojang.Version
	Java     string
	GameDir  string
	Account  auth.Account
	Injector string // path to authlib-injector.jar, empty to disable
	MemoryMB int
	ExtraJVM string
}

// Args returns the full java argument list (without the executable).
func Args(o Options) []string {
	v := o.Version
	cp := strings.Join(mojang.LibraryClasspath(v), string(os.PathListSeparator))
	natives := mojang.NativesDir(v.ID)
	index := v.AssetsIndexName()

	userType := "legacy"
	if o.Account.Kind == auth.ElyBy {
		userType = "mojang"
	}
	vars := map[string]string{
		"auth_player_name":    o.Account.Name,
		"version_name":        v.ID,
		"game_directory":      o.GameDir,
		"assets_root":         paths.Assets(),
		"game_assets":         mojang.VirtualAssetsDir(index),
		"assets_index_name":   index,
		"auth_uuid":           o.Account.UUID,
		"auth_access_token":   o.Account.AccessToken,
		"auth_session":        o.Account.AccessToken,
		"auth_xuid":           "0",
		"clientid":            "0",
		"user_type":           userType,
		"user_properties":     "{}",
		"version_type":        "Rabbit",
		"natives_directory":   natives,
		"launcher_name":       "Rabbit",
		"launcher_version":    "0.1.0",
		"classpath":           cp,
		"classpath_separator": string(os.PathListSeparator),
		"library_directory":   paths.Libraries(),
	}
	expand := func(s string) string {
		return os.Expand(s, func(k string) string {
			if val, ok := vars[k]; ok {
				return val
			}
			return "${" + k + "}"
		})
	}

	args := []string{
		"-Xmx" + strconv.Itoa(o.MemoryMB) + "M",
		"-Xms512M",
		"-Dfile.encoding=UTF-8",
		"-Dstdout.encoding=UTF-8",
	}
	if o.Injector != "" {
		args = append(args, "-javaagent:"+o.Injector+"="+auth.InjectorAPI)
	}
	args = append(args, strings.Fields(o.ExtraJVM)...)

	if v.Arguments != nil && len(v.Arguments.JVM) > 0 {
		for _, a := range v.Arguments.JVM {
			if mojang.Allowed(a.Rules) {
				for _, s := range a.Value {
					args = append(args, expand(s))
				}
			}
		}
	} else {
		args = append(args,
			"-Djava.library.path="+natives,
			"-XX:HeapDumpPath=MojangTricksIntelDriversForPerformance_javaw.exe_minecraft.exe.heapdump",
			"-cp", cp)
	}
	if lc, ok := v.Logging["client"]; ok && lc.Argument != "" {
		args = append(args, strings.ReplaceAll(lc.Argument, "${path}", mojang.LogConfig(lc.File.ID)))
	}

	args = append(args, v.MainClass)
	if v.Arguments != nil && len(v.Arguments.Game) > 0 {
		for _, a := range v.Arguments.Game {
			if mojang.Allowed(a.Rules) {
				for _, s := range a.Value {
					args = append(args, expand(s))
				}
			}
		}
	} else {
		for _, s := range strings.Fields(v.MinecraftArguments) {
			args = append(args, expand(s))
		}
	}
	return args
}

// Process is a running game.
type Process struct {
	cmd *exec.Cmd
	mu  sync.Mutex
	log []string
}

const logTail = 300

// Start runs the game. onExit is called when the process ends.
func Start(o Options, onExit func(code int, tail []string)) (*Process, error) {
	cmd := exec.Command(o.Java, Args(o)...)
	cmd.Dir = o.GameDir
	// No console, but never HideWindow: LWJGL 2 (1.12.2 and older) honours
	// the inherited SW_HIDE and the game window would stay invisible.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	cmd.Env = append(os.Environ(), "JAVA_TOOL_OPTIONS=") // ignore global java agents
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p := &Process{cmd: cmd}

	os.MkdirAll(filepath.Join(o.GameDir, "logs"), 0o755)
	logFile, _ := os.Create(filepath.Join(o.GameDir, "logs", "rabbit-launch.log"))
	go func() {
		sc := bufio.NewScanner(out)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Text()
			if logFile != nil {
				logFile.WriteString(line + "\n")
			}
			if line = plainLine(line); line == "" {
				continue
			}
			p.mu.Lock()
			p.log = append(p.log, line)
			if len(p.log) > logTail {
				p.log = p.log[len(p.log)-logTail:]
			}
			p.mu.Unlock()
		}
		err := cmd.Wait()
		if logFile != nil {
			logFile.Close()
		}
		code := 0
		if err != nil {
			code = -1
			if ee, ok := err.(*exec.ExitError); ok {
				code = ee.ExitCode()
			}
		}
		p.mu.Lock()
		tail := append([]string(nil), p.log...)
		p.mu.Unlock()
		onExit(code, tail)
	}()
	return p, nil
}

// Kill stops the game.
func (p *Process) Kill() {
	if p != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
}

// plainLine turns log4j XML output into plain text, dropping markup lines.
func plainLine(line string) string {
	t := strings.TrimSpace(line)
	if i := strings.Index(t, "<![CDATA["); i >= 0 {
		t = strings.TrimSuffix(t[i+9:], "]]></log4j:Message>")
		t = strings.TrimSuffix(t, "]]></log4j:Throwable>")
		return strings.TrimSuffix(t, "]]>")
	}
	if strings.HasPrefix(t, "<log4j:") || strings.HasPrefix(t, "</log4j:") || t == "" {
		return ""
	}
	return line
}
