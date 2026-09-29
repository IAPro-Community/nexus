//go:build windows

package protocol

import (
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	modKernel32Console             = syscall.NewLazyDLL("kernel32.dll")
	procGetStdHandle               = modKernel32Console.NewProc("GetStdHandle")
	procGetConsoleScreenBufferInfo = modKernel32Console.NewProc("GetConsoleScreenBufferInfo")
)

const stdOutputHandle = ^uint32(10) // (DWORD)-11, STD_OUTPUT_HANDLE

type consoleCoord struct {
	X int16
	Y int16
}

type consoleSmallRect struct {
	Left   int16
	Top    int16
	Right  int16
	Bottom int16
}

type consoleScreenBufferInfo struct {
	Size              consoleCoord
	CursorPosition    consoleCoord
	Attributes        uint16
	Window            consoleSmallRect
	MaximumWindowSize consoleCoord
}

type consoleResizeSignal struct{}

func (consoleResizeSignal) String() string { return "console resize" }
func (consoleResizeSignal) Signal()        {}

// NotifyWinSizeChange polls the Windows console viewport because Windows has
// no SIGWINCH equivalent. The cleanup function prevents attach/reconnect
// cycles from leaking a watcher goroutine.
func NotifyWinSizeChange(ch chan os.Signal) func() {
	if ch == nil {
		return func() {}
	}
	stop := make(chan struct{})
	var once sync.Once
	cleanup := func() { once.Do(func() { close(stop) }) }

	go func() {
		lastRows, lastCols, available := consoleViewport()
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				rows, cols, ok := consoleViewport()
				if !ok {
					continue
				}
				if !available || rows != lastRows || cols != lastCols {
					lastRows, lastCols, available = rows, cols, true
					select {
					case ch <- consoleResizeSignal{}:
					default:
					}
				}
			}
		}
	}()

	return cleanup
}

func consoleViewport() (rows, cols int, ok bool) {
	h, _, _ := procGetStdHandle.Call(uintptr(stdOutputHandle))
	if h == 0 || h == ^uintptr(0) {
		return 0, 0, false
	}
	var info consoleScreenBufferInfo
	ret, _, _ := procGetConsoleScreenBufferInfo.Call(h, uintptr(unsafe.Pointer(&info)))
	if ret == 0 || info.Window.Right < info.Window.Left || info.Window.Bottom < info.Window.Top {
		return 0, 0, false
	}
	return int(info.Window.Bottom-info.Window.Top) + 1, int(info.Window.Right-info.Window.Left) + 1, true
}
