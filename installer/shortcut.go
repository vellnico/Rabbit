package main

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
)

// appID ties the shortcut to the launcher window, so the Start menu entry,
// a pinned icon and the running launcher share one taskbar button. The
// launcher sets the same id at startup (main.go in the project root).
const appID = "vellnico.Rabbit"

var (
	clsidShellLink   = ole.NewGUID("{00021401-0000-0000-C000-000000000046}")
	iidShellLinkW    = ole.NewGUID("{000214F9-0000-0000-C000-000000000046}")
	iidPersistFile   = ole.NewGUID("{0000010B-0000-0000-C000-000000000046}")
	iidPropertyStore = ole.NewGUID("{886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99}")

	pkeyAppUserModelID = propertyKey{*ole.NewGUID("{9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3}"), 5}

	procSHChangeNotify = windows.NewLazySystemDLL("shell32.dll").NewProc("SHChangeNotify")
)

// Vtable slots of the COM methods used here (IUnknown takes 0-2).
const (
	shellLinkSetDescription      = 7
	shellLinkSetWorkingDirectory = 9
	shellLinkSetIconLocation     = 17
	shellLinkSetPath             = 20
	persistFileSave              = 6
	propertyStoreSetValue        = 6
	propertyStoreCommit          = 7
)

type propertyKey struct {
	fmtid ole.GUID
	pid   uint32
}

// propVariant is a PROPVARIANT holding a string (VT_LPWSTR).
type propVariant struct {
	vt  uint16
	_   [3]uint16
	val uintptr
	_   uintptr
}

const vtLPWSTR = 31

// comCall invokes method slot of a COM object.
func comCall(obj *ole.IUnknown, slot int, args ...uintptr) error {
	vtbl := (*[32]uintptr)(unsafe.Pointer(obj.RawVTable))
	hr, _, _ := syscall.SyscallN(vtbl[slot], append([]uintptr{uintptr(unsafe.Pointer(obj))}, args...)...)
	if int32(hr) < 0 {
		return ole.NewError(hr)
	}
	return nil
}

// comCallStr invokes a method whose first argument is a string.
func comCallStr(obj *ole.IUnknown, slot int, s string, args ...uintptr) error {
	p, err := windows.UTF16PtrFromString(s)
	if err != nil {
		return err
	}
	err = comCall(obj, slot, append([]uintptr{uintptr(unsafe.Pointer(p))}, args...)...)
	goruntime.KeepAlive(p)
	return err
}

// createShortcut writes a .lnk to target with IShellLink. Unlike the
// WScript.Shell object it needs no scripting host (often blocked by policies
// and antiviruses) and can store the AppUserModelID.
func createShortcut(lnk, target string) error {
	goruntime.LockOSThread()
	defer goruntime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		if oe, ok := err.(*ole.OleError); !ok || oe.Code() != 1 { // S_FALSE: already initialised
			return err
		}
	}
	defer ole.CoUninitialize()

	if err := os.MkdirAll(filepath.Dir(lnk), 0o755); err != nil {
		return err
	}
	link, err := ole.CreateInstance(clsidShellLink, iidShellLinkW)
	if err != nil {
		return err
	}
	defer link.Release()
	for _, set := range []struct {
		slot  int
		value string
	}{
		{shellLinkSetPath, target},
		{shellLinkSetWorkingDirectory, filepath.Dir(target)},
		{shellLinkSetDescription, "Minecraft launcher"},
	} {
		if err := comCallStr(link, set.slot, set.value); err != nil {
			return err
		}
	}
	if err := comCallStr(link, shellLinkSetIconLocation, target, 0); err != nil {
		return err
	}
	if err := setAppID(link); err != nil {
		return err
	}

	pf, err := link.QueryInterface(iidPersistFile)
	if err != nil {
		return err
	}
	defer pf.Release()
	if err := comCallStr(&pf.IUnknown, persistFileSave, lnk, 1); err != nil {
		return err
	}
	if _, err := os.Stat(lnk); err != nil {
		return err
	}
	notifyShell(shcneCreate, lnk)
	return nil
}

func setAppID(link *ole.IUnknown) error {
	ps, err := link.QueryInterface(iidPropertyStore)
	if err != nil {
		return err
	}
	defer ps.Release()
	id, err := windows.UTF16PtrFromString(appID)
	if err != nil {
		return err
	}
	v := propVariant{vt: vtLPWSTR, val: uintptr(unsafe.Pointer(id))}
	err = comCall(&ps.IUnknown, propertyStoreSetValue, uintptr(unsafe.Pointer(&pkeyAppUserModelID)), uintptr(unsafe.Pointer(&v)))
	if err == nil {
		err = comCall(&ps.IUnknown, propertyStoreCommit)
	}
	goruntime.KeepAlive(id)
	return err
}

const (
	shcneCreate       = 0x00000002
	shcneDelete       = 0x00000004
	shcneAssocChanged = 0x08000000
	shcnfIDList       = 0x0000
	shcnfPathW        = 0x0005
)

// notifyShell tells Explorer that a file was created or deleted, so the
// Start menu shows the change at once.
func notifyShell(event uint32, path string) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return
	}
	procSHChangeNotify.Call(uintptr(event), shcnfPathW, uintptr(unsafe.Pointer(p)), 0)
}

// removeShortcut deletes a shortcut and tells Explorer about it.
func removeShortcut(lnk string) {
	if err := os.Remove(lnk); err == nil {
		notifyShell(shcneDelete, lnk)
	}
}

// refreshShell makes Explorer reload icons and shortcuts.
func refreshShell() {
	procSHChangeNotify.Call(shcneAssocChanged, shcnfIDList, 0, 0)
}
