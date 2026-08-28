//go:build linux && cgo

// Package globalhk registers a system-wide X11 hotkey without panicking when
// DISPLAY is unavailable (unlike golang.design/x/hotkey's init). Soft-fail so
// the CLI half of the binary stays usable over SSH / headless CI.
package globalhk

/*
#cgo LDFLAGS: -lX11
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <stdint.h>
#include <string.h>
#include <stdlib.h>

extern void mhtodoHotkeyFire(uintptr_t handle);

static int lastGrabError = 0;

static int grabErrorHandler(Display *d, XErrorEvent *e) {
	(void)d;
	lastGrabError = e->error_code;
	return 0;
}

static Display *hkOpenDisplay(void) {
	return XOpenDisplay(NULL);
}

static int hkGrabKey(Display *d, unsigned int *mods, int nmods, KeySym keysym) {
	lastGrabError = 0;
	int keycode = XKeysymToKeycode(d, keysym);
	if (keycode == 0) {
		return -1;
	}
	XErrorHandler old = XSetErrorHandler(grabErrorHandler);
	for (int i = 0; i < nmods; i++) {
		XGrabKey(d, keycode, mods[i], DefaultRootWindow(d), False,
			GrabModeAsync, GrabModeAsync);
	}
	XSync(d, False);
	XSetErrorHandler(old);
	if (lastGrabError == BadAccess) {
		for (int i = 0; i < nmods; i++) {
			XUngrabKey(d, keycode, mods[i], DefaultRootWindow(d));
		}
		return 1;
	}
	// Deliver grabbed KeyPress events on this Display connection.
	XSelectInput(d, DefaultRootWindow(d), KeyPressMask);
	return 0;
}

static void hkUngrabKey(Display *d, unsigned int *mods, int nmods, KeySym keysym) {
	int keycode = XKeysymToKeycode(d, keysym);
	if (keycode == 0) {
		return;
	}
	for (int i = 0; i < nmods; i++) {
		XUngrabKey(d, keycode, mods[i], DefaultRootWindow(d));
	}
	XSync(d, False);
}

static Window hkCreateCancelWindow(Display *d) {
	XSetWindowAttributes attr;
	memset(&attr, 0, sizeof(attr));
	return XCreateWindow(d, DefaultRootWindow(d), 0, 0, 1, 1, 0, 0,
		InputOnly, DefaultVisual(d, 0), 0, &attr);
}

static void hkPump(uintptr_t handle, Display *d, Window w) {
	Atom cancel = XInternAtom(d, "mhtodo_hotkey_cancel", False);
	XSelectInput(d, w, 0);
	XEvent ev;
	for (;;) {
		XNextEvent(d, &ev);
		if (ev.type == KeyPress) {
			mhtodoHotkeyFire(handle);
			continue;
		}
		if (ev.type == ClientMessage && ev.xclient.message_type == cancel &&
			ev.xclient.window == w) {
			return;
		}
	}
}

static void hkSendCancel(Display *d, Window w) {
	Atom cancel = XInternAtom(d, "mhtodo_hotkey_cancel", False);
	XClientMessageEvent cm;
	memset(&cm, 0, sizeof(cm));
	cm.type = ClientMessage;
	cm.window = w;
	cm.message_type = cancel;
	cm.format = 32;
	XEvent ev;
	memset(&ev, 0, sizeof(ev));
	ev.type = ClientMessage;
	ev.xclient = cm;
	XSendEvent(d, w, False, 0, &ev);
	XFlush(d);
}

static void hkClose(Display *d, Window w) {
	XDestroyWindow(d, w);
	XCloseDisplay(d);
}
*/
import "C"
import (
	"errors"
	"fmt"
	"log"
	"runtime/cgo"
	"sync"
	"time"
	"unsafe"
)

// ErrConflict means another client already owns this combination.
var ErrConflict = errors.New("hotkey already grabbed by another application")

// Modifier is an X11 modifier mask bit.
type Modifier uint32

// Key is an X11 KeySym.
type Key uint32

// Common modifiers / keys (see X11/X.h and keysymdef.h).
const (
	ModCtrl  Modifier = 1 << 2
	ModShift Modifier = 1 << 0
	ModAlt   Modifier = 1 << 3 // Mod1 — Alt on typical layouts
	KeyT     Key      = 0x0074
)

const (
	lockCaps = 1 << 1 // LockMask
	lockNum  = 1 << 4 // Mod2Mask (NumLock on most setups)
)

// Handle owns a registered hotkey and its event loop.
type Handle struct {
	mu       sync.Mutex
	display  *C.Display
	window   C.Window
	mods     []C.uint
	key      C.KeySym
	cbHandle cgo.Handle
	done     chan struct{}
	stop     chan struct{} // closes to end the re-grab watchdog
}

type callbackBox struct {
	fn func()
}

//export mhtodoHotkeyFire
func mhtodoHotkeyFire(h C.uintptr_t) {
	box, ok := cgo.Handle(h).Value().(*callbackBox)
	if !ok || box.fn == nil {
		return
	}
	box.fn()
}

func lockVariants(base Modifier) []Modifier {
	return []Modifier{
		base,
		base | lockCaps,
		base | lockNum,
		base | lockCaps | lockNum,
	}
}

func debounce(fn func(), d time.Duration) func() {
	var mu sync.Mutex
	var last time.Time
	return func() {
		mu.Lock()
		defer mu.Unlock()
		now := time.Now()
		if !last.IsZero() && now.Sub(last) < d {
			return
		}
		last = now
		fn()
	}
}

// Register grabs mods+key globally and invokes onFire on each KeyPress
// (debounced ~300ms against X autorepeat). Returns an error (never panics)
// when DISPLAY is missing or the grab conflicts.
func Register(mods []Modifier, key Key, onFire func()) (*Handle, error) {
	if onFire == nil {
		return nil, errors.New("hotkey: nil callback")
	}
	C.XInitThreads()

	d := C.hkOpenDisplay()
	if d == nil {
		return nil, errors.New("hotkey: cannot open X11 display")
	}

	var mask Modifier
	for _, m := range mods {
		mask |= m
	}
	variants := lockVariants(mask)
	cmods := make([]C.uint, len(variants))
	for i, v := range variants {
		cmods[i] = C.uint(v)
	}

	keysym := C.KeySym(key)
	rc := C.hkGrabKey(d, (*C.uint)(unsafe.Pointer(&cmods[0])), C.int(len(cmods)), keysym)
	if rc != 0 {
		C.XCloseDisplay(d)
		if rc == 1 {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("hotkey: grab failed (%d)", int(rc))
	}

	win := C.hkCreateCancelWindow(d)
	box := &callbackBox{fn: debounce(onFire, 300*time.Millisecond)}
	cb := cgo.NewHandle(box)

	h := &Handle{
		display:  d,
		window:   win,
		mods:     append([]C.uint(nil), cmods...),
		key:      keysym,
		cbHandle: cb,
		done:     make(chan struct{}),
		stop:     make(chan struct{}),
	}

	go func() {
		defer close(h.done)
		C.hkPump(C.uintptr_t(cb), d, win)
	}()

	h.startWatchdog()

	log.Printf("global hotkey registered (ctrl+shift+alt+key)")
	return h, nil
}

const regrabInterval = 45 * time.Second

func (h *Handle) startWatchdog() {
	go func() {
		t := time.NewTicker(regrabInterval)
		defer t.Stop()
		for {
			select {
			case <-h.stop:
				return
			case <-t.C:
				if err := h.Regrab(); err != nil {
					log.Printf("global hotkey re-grab: %v", err)
				}
			}
		}
	}()
}

// Regrab renews the X11 key grab. Compositors and screen lock often drop passive
// grabs; periodic re-grab keeps Ctrl+Shift+Alt+T working on long-lived sessions.
func (h *Handle) Regrab() error {
	if h == nil {
		return errors.New("hotkey: nil handle")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.display == nil || len(h.mods) == 0 {
		return errors.New("hotkey: closed")
	}
	rc := C.hkGrabKey(h.display, (*C.uint)(unsafe.Pointer(&h.mods[0])), C.int(len(h.mods)), h.key)
	if rc != 0 {
		if rc == 1 {
			return ErrConflict
		}
		return fmt.Errorf("hotkey: grab failed (%d)", int(rc))
	}
	return nil
}

// Close unregisters the hotkey and stops the event loop.
func (h *Handle) Close() {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.display == nil {
		return
	}
	select {
	case <-h.stop:
	default:
		close(h.stop)
	}
	if len(h.mods) > 0 {
		C.hkUngrabKey(h.display, (*C.uint)(unsafe.Pointer(&h.mods[0])), C.int(len(h.mods)), h.key)
	}
	C.hkSendCancel(h.display, h.window)
	<-h.done
	C.hkClose(h.display, h.window)
	h.display = nil
	h.cbHandle.Delete()
}
