//go:build windows

package terminal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// Windows Console Pseudoterminal (ConPTY) backend.
//
// Uses the real Win32 pseudo-console API:
//
//	CreatePseudoConsole / ResizePseudoConsole / ClosePseudoConsole
//
// and launches the child via CreateProcessW with the
// PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE attribute. If ConPTY is unavailable
// (e.g. Windows < 10.0.17763), Start falls back to standard pipes and reports
// the backend truthfully as "standard pipes" (no resize / no raw mode).
// Standard pipes are the default until ConPTY is explicitly enabled with
// AI_CLI_ENABLE_CONPTY=1, because some Windows environments report a usable
// ConPTY API while returning no bytes from the output channel.

const (
	procThreadAttributePseudoConsole = 0x00020016
	extendedStartupInfoPresent       = 0x00080000
	createUnicodeEnvironment         = 0x00000400
	stillActive                      = 0x00000103 // 259
	numAttributesForConPTY           = 1
)

var (
	modKernel32             = syscall.NewLazyDLL("kernel32.dll")
	procCreatePseudoConsole = modKernel32.NewProc("CreatePseudoConsole")
	procResizePseudoConsole = modKernel32.NewProc("ResizePseudoConsole")
	procClosePseudoConsole  = modKernel32.NewProc("ClosePseudoConsole")

	procCreatePipe                        = modKernel32.NewProc("CreatePipe")
	procCreateProcessW                    = modKernel32.NewProc("CreateProcessW")
	procInitializeProcThreadAttributeList = modKernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttribute         = modKernel32.NewProc("UpdateProcThreadAttribute")
	procDeleteProcThreadAttributeList     = modKernel32.NewProc("DeleteProcThreadAttributeList")
	procCloseHandle                       = modKernel32.NewProc("CloseHandle")
	procWaitForSingleObject               = modKernel32.NewProc("WaitForSingleObject")
	procGetExitCodeProcess                = modKernel32.NewProc("GetExitCodeProcess")
	procTerminateProcess                  = modKernel32.NewProc("TerminateProcess")
	procCreateJobObjectW                  = modKernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObject           = modKernel32.NewProc("SetInformationJobObject")
	procAssignProcessToJobObject          = modKernel32.NewProc("AssignProcessToJobObject")
	procOpenProcess                       = modKernel32.NewProc("OpenProcess")
)

type coord struct {
	X int16
	Y int16
}

type startupInfo struct {
	Cb              uint32
	LpReserved      *uint16
	LpDesktop       *uint16
	LpTitle         *uint16
	DwX             uint32
	DwY             uint32
	DwXSize         uint32
	DwYSize         uint32
	DwXCountChars   uint32
	DwYCountChars   uint32
	DwFillAttribute uint32
	DwFlags         uint32
	WShowWindow     uint16
	CbReserved2     uint16
	LpReserved2     *byte
	HStdInput       uintptr
	HStdOutput      uintptr
	HStdError       uintptr
}

type startupInfoEx struct {
	StartupInfo     startupInfo
	LpAttributeList uintptr
}

type processInformation struct {
	Process   uintptr
	Thread    uintptr
	ProcessId uint32
	ThreadId  uint32
}

type windowsBackend struct {
	mu        sync.Mutex
	cmd       *exec.Cmd
	inPipe    io.WriteCloser
	rPipe     io.ReadCloser
	hProcess  uintptr
	pid       int
	hPC       uintptr
	hJob      uintptr
	isConPTY  bool
	mechanism string
}

func newPlatformBackend() Backend {
	return &windowsBackend{}
}

// BackendMechanism reports the platform's primary terminal backend.
func BackendMechanism() string { return "Windows standard pipes (ConPTY opt-in)" }

func (b *windowsBackend) Start(cmd *exec.Cmd, initialRows, initialCols int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	prepareInteractiveCommand(cmd)

	b.cmd = cmd
	if initialRows <= 0 {
		initialRows = 24
	}
	if initialCols <= 0 {
		initialCols = 80
	}

	// 1. Create anonymous pipes used as the ConPTY transport.
	var hInRead, hInWrite uintptr
	r, _, e := procCreatePipe.Call(
		uintptr(unsafe.Pointer(&hInRead)),
		uintptr(unsafe.Pointer(&hInWrite)),
		0, 0)
	if r == 0 {
		return fmt.Errorf("CreatePipe failed (input): %v", e)
	}
	var hOutRead, hOutWrite uintptr
	r, _, e = procCreatePipe.Call(
		uintptr(unsafe.Pointer(&hOutRead)),
		uintptr(unsafe.Pointer(&hOutWrite)),
		0, 0)
	if r == 0 {
		_ = closeHandle(hInRead)
		_ = closeHandle(hInWrite)
		return fmt.Errorf("CreatePipe failed (output): %v", e)
	}

	// 2. Create the pseudo console. Size is passed by value as a packed COORD.
	sizeVal := *(*uint32)(unsafe.Pointer(&coord{X: int16(initialCols), Y: int16(initialRows)}))
	var hPC uintptr
	r, _, e = procCreatePseudoConsole.Call(
		uintptr(sizeVal), uintptr(hInRead), uintptr(hOutWrite), 0, uintptr(unsafe.Pointer(&hPC)))
	if r != 0 || os.Getenv("AI_CLI_ENABLE_CONPTY") != "1" {
		// CreatePseudoConsole returns HRESULT S_OK (0) on success. A non-zero
		// HRESULT means ConPTY is unavailable. Even when ConPTY reports success,
		// keep standard pipes as the default until AI_CLI_ENABLE_CONPTY=1.
		if r == 0 {
			_ = closePseudoConsole(hPC)
			hPC = 0
		}
		childIn := os.NewFile(hInRead, "conpty-fallback-child-in")
		inFile := os.NewFile(hInWrite, "conpty-fallback-in")
		outFile := os.NewFile(hOutRead, "conpty-fallback-out")
		childOut := os.NewFile(hOutWrite, "conpty-fallback-child-out")
		b.inPipe = inFile
		b.rPipe = outFile
		b.pid = 0
		b.isConPTY = false
		b.mechanism = "standard pipes (ConPTY unavailable)"
		if r == 0 && os.Getenv("AI_CLI_ENABLE_CONPTY") != "1" {
			b.mechanism = "standard pipes (ConPTY opt-in)"
		}
		cmd.Stdin = childIn
		cmd.Stdout = childOut
		cmd.Stderr = childOut
		if err := cmd.Start(); err != nil {
			_ = childIn.Close()
			_ = childOut.Close()
			_ = inFile.Close()
			_ = outFile.Close()
			return fmt.Errorf("failed to start process (pipe fallback): %w", err)
		}
		// Parent copies of child-side handles must be closed so EOF is
		// observable when the child exits.
		_ = childIn.Close()
		_ = childOut.Close()
		b.pid = cmd.Process.Pid
		b.cmd = cmd
		return nil
	}

	// 3. Keep the server ends open until CreateProcessW. Windows requires the
	// handles passed to CreatePseudoConsole to remain valid while the child is
	// attached to the pseudo console.

	// 4. Build the PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE attribute list. The
	// child communicates through HPCON and must not inherit the transport pipe
	// handles. Adding PROC_THREAD_ATTRIBUTE_HANDLE_LIST while passing
	// bInheritHandles=FALSE makes CreateProcessW fail with ERROR_INVALID_PARAMETER.
	var attrSize uintptr
	procInitializeProcThreadAttributeList.Call(0, numAttributesForConPTY, 0, uintptr(unsafe.Pointer(&attrSize)))
	if attrSize == 0 {
		_ = closeHandle(hInRead)
		_ = closeHandle(hOutWrite)
		_ = closeHandle(hInWrite)
		_ = closeHandle(hOutRead)
		_ = closePseudoConsole(hPC)
		return errors.New("InitializeProcThreadAttributeList failed to size attribute list")
	}
	attrBuf := make([]byte, attrSize)
	r, _, e = procInitializeProcThreadAttributeList.Call(
		uintptr(unsafe.Pointer(&attrBuf[0])), numAttributesForConPTY, 0, uintptr(unsafe.Pointer(&attrSize)))
	if r == 0 {
		_ = closeHandle(hInRead)
		_ = closeHandle(hOutWrite)
		_ = closeHandle(hInWrite)
		_ = closeHandle(hOutRead)
		_ = closePseudoConsole(hPC)
		return fmt.Errorf("InitializeProcThreadAttributeList failed: %v", e)
	}
	defer procDeleteProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&attrBuf[0])))

	r, _, e = procUpdateProcThreadAttribute.Call(
		uintptr(unsafe.Pointer(&attrBuf[0])),
		0,
		procThreadAttributePseudoConsole,
		// PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE expects the HPCON handle
		// value itself, not a pointer to the local handle variable. This
		// matches the Win32 ConPTY contract and keeps the child attached
		// to the pseudo console that was just created.
		hPC,
		unsafe.Sizeof(hPC),
		0, 0)
	if r == 0 {
		_ = closeHandle(hInRead)
		_ = closeHandle(hOutWrite)
		_ = closeHandle(hInWrite)
		_ = closeHandle(hOutRead)
		_ = closePseudoConsole(hPC)
		return fmt.Errorf("UpdateProcThreadAttribute (pseudo console) failed: %v", e)
	}

	// 5. Launch the child attached to the pseudo console. Shell shims
	// (.cmd/.bat) must go through ComSpec; CreateProcessW gets lpApplicationName
	// null and a full command line from createProcessSpec.
	appPath, commandLineText := createProcessSpec(cmd)
	if appPath == "" || commandLineText == "" {
		_ = closeHandle(hInRead)
		_ = closeHandle(hOutWrite)
		_ = closeHandle(hInWrite)
		_ = closeHandle(hOutRead)
		_ = closePseudoConsole(hPC)
		return fmt.Errorf("invalid process spec for %q", cmd.Path)
	}
	commandLine, err := syscall.UTF16FromString(commandLineText)
	if err != nil {
		_ = closeHandle(hInRead)
		_ = closeHandle(hOutWrite)
		_ = closeHandle(hInWrite)
		_ = closeHandle(hOutRead)
		_ = closePseudoConsole(hPC)
		return fmt.Errorf("invalid command line: %w", err)
	}
	envBlock := utf16EnvBlock(cmd.Environ())
	var dirPtr *uint16
	if cmd.Dir != "" {
		dirPtr, err = syscall.UTF16PtrFromString(cmd.Dir)
		if err != nil {
			_ = closeHandle(hInRead)
			_ = closeHandle(hOutWrite)
			_ = closeHandle(hInWrite)
			_ = closeHandle(hOutRead)
			_ = closePseudoConsole(hPC)
			return fmt.Errorf("invalid working directory %q: %w", cmd.Dir, err)
		}
	}

	siEx := startupInfoEx{}
	siEx.StartupInfo.Cb = uint32(unsafe.Sizeof(siEx))
	siEx.LpAttributeList = uintptr(unsafe.Pointer(&attrBuf[0]))

	var pi processInformation
	r, _, e = procCreateProcessW.Call(
		0,
		uintptr(unsafe.Pointer(&commandLine[0])),
		0, 0, 0,
		extendedStartupInfoPresent|createUnicodeEnvironment,
		uintptr(unsafe.Pointer(&envBlock[0])),
		uintptr(unsafe.Pointer(dirPtr)),
		uintptr(unsafe.Pointer(&siEx)),
		uintptr(unsafe.Pointer(&pi)))
	if r == 0 {
		_ = closeHandle(hInRead)
		_ = closeHandle(hOutWrite)
		_ = closeHandle(hInWrite)
		_ = closeHandle(hOutRead)
		_ = closePseudoConsole(hPC)
		return fmt.Errorf("CreateProcessW failed for %q: %v", appPath, e)
	}
	if pi.Thread != 0 {
		_ = closeHandle(pi.Thread)
	}
	_ = closeHandle(hInRead)
	_ = closeHandle(hOutWrite)

	b.inPipe = os.NewFile(hInWrite, "conpty-in")
	b.rPipe = os.NewFile(hOutRead, "conpty-out")
	b.hProcess = pi.Process
	b.pid = int(pi.ProcessId)
	b.hPC = hPC
	b.isConPTY = true
	b.mechanism = "ConPTY (CreatePseudoConsole)"

	// exec.Cmd must not try to manage the child itself on Windows.
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Process = &os.Process{Pid: b.pid}
	return nil
}

func (b *windowsBackend) Read(p []byte) (int, error) {
	b.mu.Lock()
	rPipe := b.rPipe
	b.mu.Unlock()
	if rPipe != nil {
		return rPipe.Read(p)
	}
	return 0, io.EOF
}

func (b *windowsBackend) Write(p []byte) (int, error) {
	b.mu.Lock()
	inPipe := b.inPipe
	b.mu.Unlock()
	if inPipe != nil {
		return inPipe.Write(p)
	}
	return 0, io.ErrClosedPipe
}

func (b *windowsBackend) Resize(rows, cols int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.isConPTY || b.hPC == 0 || rows <= 0 || cols <= 0 {
		return nil
	}
	sizeVal := *(*uint32)(unsafe.Pointer(&coord{X: int16(cols), Y: int16(rows)}))
	_, _, _ = procResizePseudoConsole.Call(b.hPC, uintptr(sizeVal))
	return nil
}

func (b *windowsBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.inPipe != nil {
		_ = b.inPipe.Close()
		b.inPipe = nil
	}
	if b.rPipe != nil {
		_ = b.rPipe.Close()
		b.rPipe = nil
	}
	if b.hPC != 0 {
		_ = closePseudoConsole(b.hPC)
		b.hPC = 0
	}
	if b.hProcess != 0 {
		_ = closeHandle(b.hProcess)
		b.hProcess = 0
	}
	if b.hJob != 0 {
		_ = closeHandle(b.hJob)
		b.hJob = 0
	}
	return nil
}

func (b *windowsBackend) Supervise() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.hJob != 0 {
		return nil
	}
	process := b.hProcess
	if process == 0 && b.cmd != nil && b.cmd.Process != nil {
		const processAllAccess = 0x001F0FFF
		h, _, _ := procOpenProcess.Call(processAllAccess, 0, uintptr(b.cmd.Process.Pid))
		process = h
		if process != 0 {
			b.hProcess = process
		}
	}
	if process == 0 {
		return errors.New("provider process handle unavailable")
	}
	job, _, err := procCreateJobObjectW.Call(0, 0)
	if job == 0 {
		return fmt.Errorf("CreateJobObjectW failed: %v", err)
	}
	// JOBOBJECT_EXTENDED_LIMIT_INFORMATION has the BasicLimitInformation
	// block followed by IO counters and four memory limits. The kill-on-close
	// bit is the first DWORD in the BasicLimitInformation.Flags field.
	info := make([]byte, 144)
	*(*uint32)(unsafe.Pointer(&info[16])) = 0x00002000
	if result, _, callErr := procSetInformationJobObject.Call(job, 9, uintptr(unsafe.Pointer(&info[0])), uintptr(len(info))); result == 0 {
		_ = closeHandle(job)
		return fmt.Errorf("SetInformationJobObject failed: %v", callErr)
	}
	if result, _, callErr := procAssignProcessToJobObject.Call(job, process); result == 0 {
		_ = closeHandle(job)
		return fmt.Errorf("AssignProcessToJobObject failed: %v", callErr)
	}
	b.hJob = job
	return nil
}

func (b *windowsBackend) PID() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.pid
}

func (b *windowsBackend) Wait() error {
	b.mu.Lock()
	h := b.hProcess
	cmd := b.cmd
	isConPTY := b.isConPTY
	b.mu.Unlock()
	if !isConPTY {
		if cmd != nil {
			return cmd.Wait()
		}
		return nil
	}
	if h == 0 {
		return nil
	}
	_, _, _ = procWaitForSingleObject.Call(h, ^uintptr(0)) // INFINITE
	b.mu.Lock()
	if b.rPipe != nil {
		_ = b.rPipe.Close()
		b.rPipe = nil
	}
	b.mu.Unlock()
	var code uint32
	_, _, _ = procGetExitCodeProcess.Call(h, uintptr(unsafe.Pointer(&code)))
	if code == stillActive {
		return errors.New("process still active")
	}
	return nil
}

func (b *windowsBackend) WaitContext(ctx context.Context) error {
	done := make(chan error, 1)
	go func() { done <- b.Wait() }()
	select {
	case <-ctx.Done():
		_ = b.Kill()
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func (b *windowsBackend) Signal(sig os.Signal) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.isConPTY {
		// Interrupt (Ctrl+C) is delivered to the pseudo console via the input pipe.
		if sig == os.Interrupt && b.inPipe != nil {
			_, _ = b.inPipe.Write([]byte{0x03})
			return nil
		}
		return nil
	}
	if b.cmd != nil && b.cmd.Process != nil {
		return b.cmd.Process.Signal(sig)
	}
	return nil
}

func (b *windowsBackend) Kill() error {
	b.mu.Lock()
	h := b.hProcess
	cmd := b.cmd
	isConPTY := b.isConPTY
	b.mu.Unlock()
	if isConPTY {
		if h == 0 {
			return nil
		}
		r, _, e := procTerminateProcess.Call(h, 1)
		if r == 0 {
			return e
		}
		return nil
	}
	if cmd != nil && cmd.Process != nil {
		return cmd.Process.Kill()
	}
	return nil
}

func (b *windowsBackend) SupportsResize() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.isConPTY
}

func (b *windowsBackend) SupportsRawMode() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.isConPTY
}

func (b *windowsBackend) Mechanism() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.mechanism == "" {
		return "unknown"
	}
	return b.mechanism
}

func closePseudoConsole(h uintptr) error {
	// ClosePseudoConsole is a void Win32 API. Its return register is undefined;
	// interpreting it as BOOL can report false failures after a successful close.
	_, _, _ = procClosePseudoConsole.Call(h)
	return nil
}

func closeHandle(h uintptr) error {
	if h == 0 {
		return nil
	}
	r, _, e := procCloseHandle.Call(h)
	if r == 0 {
		return e
	}
	return nil
}

// buildCommandLine joins argv with Windows quoting rules.
func buildCommandLine(args []string) string {
	var sb strings.Builder
	for i, a := range args {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(escapeArg(a))
	}
	return sb.String()
}

// createProcessSpec returns an application path and command line suitable for
// CreateProcessW. Windows shell shims (.cmd/.bat) are not Win32 executables,
// so they must be launched through ComSpec instead of being passed as the
// application name directly.
func createProcessSpec(cmd *exec.Cmd) (string, string) {
	if cmd == nil {
		return "", ""
	}
	if len(cmd.Args) == 0 {
		return cmd.Path, escapeArg(cmd.Path)
	}
	if !isWindowsShellShim(cmd.Path) {
		args := append([]string{cmd.Path}, cmd.Args[1:]...)
		return cmd.Path, buildCommandLine(args)
	}

	comspec := os.Getenv("ComSpec")
	if comspec == "" {
		comspec = filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	}

	// The doubled quote after /c is intentional: it preserves a quoted
	// script path while using cmd.exe's /s /c parsing rules.
	payload := buildCommandLine(cmd.Args)
	return comspec, escapeArg(comspec) + " /d /s /c \"" + payload + "\""
}

func isWindowsShellShim(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".cmd" || ext == ".bat"
}

// escapeArg quotes an argument following Windows CreateProcess conventions.
func escapeArg(s string) string {
	if len(s) == 0 {
		return `""`
	}
	hasSpace := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"', '\\', ' ', '\t':
			hasSpace = true
		}
	}
	if !hasSpace {
		return s
	}
	var sb strings.Builder
	sb.WriteByte('"')
	i := 0
	for i < len(s) {
		backslashes := 0
		for i < len(s) && s[i] == '\\' {
			backslashes++
			i++
		}
		if i == len(s) {
			for j := 0; j < backslashes*2; j++ {
				sb.WriteByte('\\')
			}
			break
		}
		if s[i] == '"' {
			for j := 0; j < backslashes*2+1; j++ {
				sb.WriteByte('\\')
			}
			sb.WriteByte('"')
			i++
		} else {
			for j := 0; j < backslashes; j++ {
				sb.WriteByte('\\')
			}
			sb.WriteByte(s[i])
			i++
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

// utf16EnvBlock builds a double-null-terminated UTF-16 environment block.
func utf16EnvBlock(env []string) []uint16 {
	var block []uint16
	for _, e := range env {
		block = append(block, syscall.StringToUTF16(e)...)
	}
	block = append(block, 0)
	if len(block) == 0 {
		block = append(block, 0)
	}
	return block
}
