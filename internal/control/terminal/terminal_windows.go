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
	"time"
	"unicode/utf16"
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
// Standard pipes remain the default for generic runtimes. Codex and AGY use
// ConPTY automatically because their interactive CLIs require a console;
// AI_CLI_ENABLE_CONPTY=1 opts other providers in and AI_CLI_DISABLE_CONPTY=1
// preserves the compatibility fallback.

const (
	procThreadAttributePseudoConsole = 0x00020016
	extendedStartupInfoPresent       = 0x00080000
	createUnicodeEnvironment         = 0x00000400
	startfUseStdHandles              = 0x00000100
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
	procLocalAlloc                        = modKernel32.NewProc("LocalAlloc")
	procLocalFree                         = modKernel32.NewProc("LocalFree")
	procCloseHandle                       = modKernel32.NewProc("CloseHandle")
	procWaitForSingleObject               = modKernel32.NewProc("WaitForSingleObject")
	procGetExitCodeProcess                = modKernel32.NewProc("GetExitCodeProcess")
	procTerminateProcess                  = modKernel32.NewProc("TerminateProcess")
	procTerminateJobObject                = modKernel32.NewProc("TerminateJobObject")
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
	mu         sync.Mutex
	providerID string
	cmd        *exec.Cmd
	inPipe     io.WriteCloser
	rPipe      io.ReadCloser
	hProcess   uintptr
	pid        int
	hPC        uintptr
	hJob       uintptr
	isConPTY   bool
	mechanism  string
}

func newPlatformBackend() Backend {
	return &windowsBackend{}
}

func newPlatformBackendForProvider(providerID string) Backend {
	return &windowsBackend{providerID: strings.TrimSpace(providerID)}
}

// BackendMechanism reports the platform's primary terminal backend.
func BackendMechanism() string { return "Windows standard pipes (ConPTY opt-in)" }

func platformBackendMechanismForProvider(providerID string) string {
	if !shouldUseConPTY(providerID) {
		return BackendMechanism()
	}
	if os.Getenv("AI_CLI_ENABLE_CONPTY") != "1" {
		switch {
		case strings.EqualFold(strings.TrimSpace(providerID), "codex"):
			return "Windows ConPTY (automatic for supervised Codex)"
		case strings.EqualFold(strings.TrimSpace(providerID), "agy"):
			return "Windows ConPTY (automatic for supervised AGY)"
		}
	}
	return "Windows ConPTY (explicit opt-in)"
}

func shouldUseConPTY(providerID string) bool {
	if os.Getenv("AI_CLI_DISABLE_CONPTY") == "1" {
		return false
	}
	if os.Getenv("AI_CLI_ENABLE_CONPTY") == "1" {
		return true
	}
	providerID = strings.TrimSpace(providerID)
	return strings.EqualFold(providerID, "codex") || strings.EqualFold(providerID, "agy")
}

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

	// 2. Honor provider-specific defaults while preserving the global opt-in
	// policy for other runtimes and the explicit compatibility override.
	conPTYEnabled := shouldUseConPTY(b.providerID)
	sizeVal := *(*uint32)(unsafe.Pointer(&coord{X: int16(initialCols), Y: int16(initialRows)}))
	var hPC uintptr
	if conPTYEnabled {
		r, _, _ = procCreatePseudoConsole.Call(
			uintptr(sizeVal), hInRead, hOutWrite, 0, uintptr(unsafe.Pointer(&hPC)))
	}
	if !conPTYEnabled || r != 0 {
		// CreatePseudoConsole returns HRESULT S_OK (0) on success. If ConPTY was
		// requested but is unavailable, fall back to standard pipes truthfully.
		if conPTYEnabled && r == 0 && hPC != 0 {
			closePseudoConsole(hPC)
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
		b.mechanism = "standard pipes (ConPTY opt-in disabled)"
		if conPTYEnabled && r != 0 {
			b.mechanism = "standard pipes (ConPTY unavailable)"
		} else if os.Getenv("AI_CLI_DISABLE_CONPTY") == "1" {
			b.mechanism = "standard pipes (ConPTY explicitly disabled)"
		}
		cmd.Stdin = childIn
		cmd.Stdout = childOut
		cmd.Stderr = childOut
		if cmd.SysProcAttr == nil {
			cmd.SysProcAttr = &syscall.SysProcAttr{}
		}
		cmd.SysProcAttr.HideWindow = true
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
		closePseudoConsole(hPC)
		return errors.New("InitializeProcThreadAttributeList failed to size attribute list")
	}
	attrMem, _, e := procLocalAlloc.Call(0, attrSize)
	if attrMem == 0 {
		_ = closeHandle(hInWrite)
		_ = closeHandle(hOutRead)
		closePseudoConsole(hPC)
		return fmt.Errorf("LocalAlloc failed for process attribute list: %v", e)
	}
	defer procLocalFree.Call(attrMem)
	r, _, e = procInitializeProcThreadAttributeList.Call(
		attrMem, numAttributesForConPTY, 0, uintptr(unsafe.Pointer(&attrSize)))
	if r == 0 {
		_ = closeHandle(hInRead)
		_ = closeHandle(hOutWrite)
		_ = closeHandle(hInWrite)
		_ = closeHandle(hOutRead)
		closePseudoConsole(hPC)
		return fmt.Errorf("InitializeProcThreadAttributeList failed: %v", e)
	}
	defer procDeleteProcThreadAttributeList.Call(attrMem)

	r, _, e = procUpdateProcThreadAttribute.Call(
		attrMem,
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
		closePseudoConsole(hPC)
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
		closePseudoConsole(hPC)
		return fmt.Errorf("invalid process spec for %q", cmd.Path)
	}
	commandLine, err := syscall.UTF16FromString(commandLineText)
	if err != nil {
		_ = closeHandle(hInRead)
		_ = closeHandle(hOutWrite)
		_ = closeHandle(hInWrite)
		_ = closeHandle(hOutRead)
		closePseudoConsole(hPC)
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
			closePseudoConsole(hPC)
			return fmt.Errorf("invalid working directory %q: %w", cmd.Dir, err)
		}
	}

	siEx := startupInfoEx{}
	siEx.StartupInfo.Cb = uint32(unsafe.Sizeof(siEx))
	// ConPTY owns the terminal transport. Clearing inherited standard handles
	// prevents the child from falling back to a host console behind Nexus.
	siEx.StartupInfo.DwFlags = startfUseStdHandles
	siEx.LpAttributeList = attrMem

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
		closePseudoConsole(hPC)
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
	if b.isConPTY && b.hProcess != 0 {
		_, _, _ = procTerminateProcess.Call(b.hProcess, 1)
	}
	if b.rPipe != nil {
		// ConPTY can emit a final frame while closing. Drain concurrently so
		// ClosePseudoConsole cannot deadlock behind a full synchronous pipe.
		rPipe := b.rPipe
		go func() {
			buf := make([]byte, 4096)
			for {
				if _, err := rPipe.Read(buf); err != nil {
					return
				}
			}
		}()
	}
	if b.hPC != 0 {
		hPC := b.hPC
		done := make(chan struct{})
		go func() {
			closePseudoConsole(hPC)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
		b.hPC = 0
	}
	if b.rPipe != nil {
		_ = b.rPipe.Close()
		b.rPipe = nil
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
		// AssignProcessToJobObject requires PROCESS_SET_QUOTA and
		// PROCESS_TERMINATE. Avoid PROCESS_ALL_ACCESS: it is unnecessary and can
		// be rejected by locked-down Windows environments.
		const (
			processTerminate = 0x0001
			processSetQuota  = 0x0100
		)
		h, _, _ := procOpenProcess.Call(processTerminate|processSetQuota, 0, uintptr(b.cmd.Process.Pid))
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
	var code uint32
	result, _, getErr := procGetExitCodeProcess.Call(h, uintptr(unsafe.Pointer(&code)))
	if result == 0 {
		return getErr
	}
	if code == stillActive {
		_, _, _ = procWaitForSingleObject.Call(h, ^uintptr(0)) // INFINITE
	}
	// Closing the read side unblocks pending readers, but a synchronous Close
	// waits for any in-flight ReadFile, and conhost can delay the final EOF
	// after the client exits (observed as multi-minute teardown stalls in CI).
	// Bound the wait so a stuck conhost cannot pin Wait; the helper goroutine
	// still completes the close when the EOF arrives.
	b.mu.Lock()
	rPipe := b.rPipe
	b.rPipe = nil
	b.mu.Unlock()
	if rPipe != nil {
		closed := make(chan struct{})
		go func() {
			_ = rPipe.Close()
			close(closed)
		}()
		select {
		case <-closed:
		case <-time.After(2 * time.Second):
		}
	}
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

	// Windows does not provide Unix-style SIGINT delivery through os.Process.Signal
	// for ordinary child processes. Both the ConPTY and the safe pipe backend keep
	// a writable provider stdin, so deliver Ctrl+C as the ETX byte the interactive
	// provider expects. This is the same byte path used by attached user input and
	// does not involve Nexus prompt interpretation.
	if sig == os.Interrupt && b.inPipe != nil {
		_, err := b.inPipe.Write([]byte{0x03})
		return err
	}
	if b.isConPTY {
		// Other graceful signals have no direct ConPTY equivalent. SessionHost
		// escalates to Kill after its bounded graceful-stop timeout.
		return nil
	}
	if b.cmd != nil && b.cmd.Process != nil {
		return b.cmd.Process.Signal(sig)
	}
	return nil
}

func (b *windowsBackend) Kill() error {
	b.mu.Lock()
	hJob := b.hJob
	h := b.hProcess
	cmd := b.cmd
	isConPTY := b.isConPTY
	b.mu.Unlock()

	// Once the backend is supervised, the Job Object is the process-tree
	// authority. Killing only the root process can leave a child holding the
	// PTY/pipe open; SessionHost waits for terminal drain before Close(), so
	// that ordering can otherwise deadlock termination. Terminate the whole
	// job first and keep the handle for the normal Close() cleanup.
	if hJob != 0 {
		r, _, e := procTerminateJobObject.Call(hJob, 1)
		if r != 0 {
			return nil
		}
		jobErr := fmt.Errorf("TerminateJobObject failed: %v", e)
		if isConPTY {
			if h != 0 {
				if result, _, processErr := procTerminateProcess.Call(h, 1); result == 0 {
					return fmt.Errorf("%v; TerminateProcess failed: %v", jobErr, processErr)
				}
			}
			return jobErr
		}
		if cmd != nil && cmd.Process != nil {
			if processErr := cmd.Process.Kill(); processErr != nil {
				return fmt.Errorf("%v; process kill failed: %v", jobErr, processErr)
			}
		}
		return jobErr
	}

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

func closePseudoConsole(h uintptr) {
	// ClosePseudoConsole is a void Win32 API. Its return register is undefined;
	// interpreting it as BOOL can report false failures after a successful close.
	_, _, _ = procClosePseudoConsole.Call(h)
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
		block = append(block, utf16.Encode([]rune(e))...)
		block = append(block, 0)
	}
	if len(block) == 0 {
		return []uint16{0, 0}
	}
	block = append(block, 0)
	return block
}
