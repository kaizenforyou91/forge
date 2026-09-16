//go:build windows

package runtime

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows SDK ProcThreadAttributeValue(13, false, true, false). This attribute
// requires Windows 10 / Server 2016 or newer. Unsupported admission fails closed.
const windowsScopeJobList = 0x0002000d

// This is a private, trusted-caller launch declaration, not a public executor.
// All paths/environment/stdio are explicit; no shell, inherited environment,
// arbitrary creation flags, external Job, token, or parent-process override.
// The caller must keep the borrowed stdio handles live until start returns.
type windowsProcessScopeLaunchSpec struct {
	executable  string
	commandLine string
	directory   string
	environment []string
	stdio       [3]windows.Handle
}

// The launch result separates Job control from direct-child handle ownership.
// Its caller exclusively owns process (including wait/exit-code/CloseHandle).
// No ProcessRunner path consumes this object in B3.
type windowsProcessScopeLaunch struct {
	scope   *windowsProcessScopePlatform
	process windows.Handle
}

type windowsProcessScopePlatform struct {
	mu       sync.Mutex
	job      windows.Handle
	killJob  func(windows.Handle, uint32) error
	closeJob func(windows.Handle) error
	closeErr error
}

var _ processScopePlatform = (*windowsProcessScopePlatform)(nil)

func (s *windowsProcessScopePlatform) terminate() error {
	if s == nil {
		return os.ErrProcessDone
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.job == 0 || s.killJob == nil {
		return os.ErrProcessDone
	}
	// Acknowledges only the control request. Completion must be observed
	// separately; neither process exit nor I/O quiescence is implied.
	return s.killJob(s.job, 1)
}

func (s *windowsProcessScopePlatform) finalize() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.job == 0 {
		return s.closeErr
	}
	h, closeJob := s.job, s.closeJob
	s.job, s.killJob, s.closeJob = 0, nil, nil
	s.closeErr = errProcessScopeFinalizeInterrupted
	s.closeErr = closeJob(h)
	return s.closeErr
}

// Narrow native seams are per-call, never mutable globals. They provide no
// caller-selected PID/Job authority and are used by deterministic failure tests.
type windowsScopeAttributes interface {
	update(uintptr, unsafe.Pointer, uintptr) error
	list() *windows.ProcThreadAttributeList
	close() error
}

type windowsScopeNative struct {
	newJob     func() (windows.Handle, error)
	configure  func(windows.Handle) error
	duplicate  func(windows.Handle) (windows.Handle, error)
	attributes func() (windowsScopeAttributes, error)
	create     func(*uint16, *uint16, bool, uint32, *uint16, *uint16, *windows.StartupInfo, *windows.ProcessInformation) error
	terminate  func(windows.Handle, uint32) error
	close      func(windows.Handle) error
}

func windowsScopeSystem() windowsScopeNative {
	return windowsScopeNative{
		newJob: func() (windows.Handle, error) { return windows.CreateJobObject(nil, nil) },
		configure: func(h windows.Handle) error {
			info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
			info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
			_, err := windows.SetInformationJobObject(h, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
			return err
		},
		duplicate: func(h windows.Handle) (windows.Handle, error) {
			// stdio can only be a file/pipe/console handle, never a Job capability.
			kind, err := windows.GetFileType(h)
			if err != nil {
				return 0, err
			}
			if kind != windows.FILE_TYPE_DISK && kind != windows.FILE_TYPE_PIPE && kind != windows.FILE_TYPE_CHAR {
				return 0, errors.New("invalid scope stdio type")
			}
			var dup windows.Handle
			err = windows.DuplicateHandle(windows.CurrentProcess(), h, windows.CurrentProcess(), &dup, 0, true, windows.DUPLICATE_SAME_ACCESS)
			return dup, err
		},
		attributes: newWindowsScopeAttributes,
		create: func(app, cmd *uint16, inherit bool, flags uint32, env, dir *uint16, si *windows.StartupInfo, pi *windows.ProcessInformation) error {
			return windows.CreateProcess(app, cmd, nil, nil, inherit, flags, env, dir, si, pi)
		},
		terminate: windows.TerminateJobObject,
		close:     windows.CloseHandle,
	}
}

func startWindowsProcessInScope(spec *windowsProcessScopeLaunchSpec) (*windowsProcessScopeLaunch, error) {
	return startWindowsProcessInScopeWith(spec, windowsScopeSystem())
}

func windowsScopeStrings(spec *windowsProcessScopeLaunchSpec) (app, cmd, dir, env []uint16, err error) {
	invalid := errors.New("invalid Windows scope launch declaration")
	if spec == nil || !filepath.IsAbs(spec.executable) || !filepath.IsAbs(spec.directory) || spec.commandLine == "" || spec.environment == nil {
		err = invalid
		return
	}
	for _, h := range spec.stdio {
		if h == 0 || h == windows.InvalidHandle {
			err = invalid
			return
		}
	}
	if app, err = windows.UTF16FromString(spec.executable); err != nil {
		return
	}
	if cmd, err = windows.UTF16FromString(spec.commandLine); err != nil {
		return
	}
	if dir, err = windows.UTF16FromString(spec.directory); err != nil {
		return
	}
	if len(cmd) > 32767 {
		err = invalid
		return
	}
	entries := append([]string(nil), spec.environment...)
	seen := make(map[string]bool, len(entries))
	for _, e := range entries {
		i := strings.IndexByte(e, '=')
		if i <= 0 || strings.ContainsRune(e, 0) {
			err = invalid
			return
		}
		key := strings.ToUpper(e[:i])
		if seen[key] {
			err = invalid
			return
		}
		seen[key] = true
	}
	sort.Slice(entries, func(i, j int) bool { return strings.ToUpper(entries[i]) < strings.ToUpper(entries[j]) })
	for _, e := range entries {
		env = append(env, utf16.Encode([]rune(e))...)
		env = append(env, 0)
	}
	env = append(env, 0)
	if len(entries) == 0 {
		env = append(env, 0)
	}
	return
}

func startWindowsProcessInScopeWith(spec *windowsProcessScopeLaunchSpec, n windowsScopeNative) (launch *windowsProcessScopeLaunch, err error) {
	app, cmd, dir, env, err := windowsScopeStrings(spec)
	if err != nil {
		return nil, err
	}
	job, err := n.newJob()
	if err != nil {
		return nil, err
	}
	scope := &windowsProcessScopePlatform{job: job, killJob: n.terminate, closeJob: n.close}
	var pi windows.ProcessInformation
	var handles []windows.Handle
	var attrs windowsScopeAttributes
	// Cleanup runs synchronously, including failure after successful creation.
	// No retry, detached worker, fallback profile, or hidden cleanup error.
	defer func() {
		if attrs != nil {
			err = errors.Join(err, attrs.close())
		}
		for _, h := range handles {
			err = errors.Join(err, n.close(h))
		}
		if pi.Thread != 0 {
			err = errors.Join(err, n.close(pi.Thread))
		}
		if launch == nil || err != nil {
			if pi.Process != 0 {
				err = errors.Join(err, scope.terminate())
			}
			err = errors.Join(err, scope.finalize())
			if pi.Process != 0 {
				err = errors.Join(err, n.close(pi.Process))
			}
			launch = nil
		}
	}()
	if err = n.configure(job); err != nil {
		return nil, err
	}
	for _, h := range spec.stdio {
		var dup windows.Handle
		dup, err = n.duplicate(h)
		if err != nil {
			return nil, err
		}
		handles = append(handles, dup)
	}
	attrs, err = n.attributes()
	if err != nil {
		return nil, err
	}
	jobs := [1]windows.Handle{job}
	// Attribute values must live through list destruction, not just Update.
	defer func() { runtime.KeepAlive(jobs); runtime.KeepAlive(handles) }()
	if err = attrs.update(windowsScopeJobList, unsafe.Pointer(&jobs[0]), unsafe.Sizeof(jobs)); err != nil {
		return nil, err
	}
	if err = attrs.update(windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST, unsafe.Pointer(&handles[0]), uintptr(len(handles))*unsafe.Sizeof(handles[0])); err != nil {
		return nil, err
	}
	si := windows.StartupInfoEx{}
	si.Cb = uint32(unsafe.Sizeof(si))
	si.Flags = windows.STARTF_USESTDHANDLES
	si.StdInput, si.StdOutput, si.StdErr = handles[0], handles[1], handles[2]
	si.ProcThreadAttributeList = attrs.list()
	const flags = windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_UNICODE_ENVIRONMENT
	err = n.create(&app[0], &cmd[0], true, flags, &env[0], &dir[0], &si.StartupInfo, &pi)
	if err != nil {
		return nil, err
	}
	return &windowsProcessScopeLaunch{scope: scope, process: pi.Process}, nil
}

// x/sys v0.13.0's container does not free its allocation when its second
// initialization fails. Keep this narrow wrapper so partial initialization also
// has exact ownership. The aligned buffer and attribute values are pinned until
// Delete, and are never retained after CreateProcess returns.
type windowsScopeAttributeList struct {
	buffer  []uintptr
	pins    runtime.Pinner
	pointer *windows.ProcThreadAttributeList
	values  []unsafe.Pointer
}

var (
	windowsScopeKernel           = windows.NewLazySystemDLL("kernel32.dll")
	windowsScopeInitAttributes   = windowsScopeKernel.NewProc("InitializeProcThreadAttributeList")
	windowsScopeUpdateAttributes = windowsScopeKernel.NewProc("UpdateProcThreadAttribute")
	windowsScopeDeleteAttributes = windowsScopeKernel.NewProc("DeleteProcThreadAttributeList")
)

func newWindowsScopeAttributes() (windowsScopeAttributes, error) {
	for _, proc := range []*windows.LazyProc{windowsScopeInitAttributes, windowsScopeUpdateAttributes, windowsScopeDeleteAttributes} {
		if err := proc.Find(); err != nil {
			return nil, err
		}
	}
	var size uintptr
	r, _, e := windowsScopeInitAttributes.Call(0, 2, 0, uintptr(unsafe.Pointer(&size)))
	if r != 0 || e != windows.ERROR_INSUFFICIENT_BUFFER || size == 0 || size > 1<<20 {
		return nil, errors.New("invalid scope attribute allocation size")
	}
	word := unsafe.Sizeof(uintptr(0))
	a := &windowsScopeAttributeList{buffer: make([]uintptr, (size+word-1)/word)}
	a.pointer = (*windows.ProcThreadAttributeList)(unsafe.Pointer(&a.buffer[0]))
	a.pins.Pin(&a.buffer[0])
	r, _, e = windowsScopeInitAttributes.Call(uintptr(unsafe.Pointer(a.pointer)), 2, 0, uintptr(unsafe.Pointer(&size)))
	if r == 0 {
		a.pins.Unpin()
		return nil, e
	}
	return a, nil
}

func (a *windowsScopeAttributeList) update(key uintptr, p unsafe.Pointer, size uintptr) error {
	a.values = append(a.values, p)
	a.pins.Pin(p)
	r, _, e := windowsScopeUpdateAttributes.Call(uintptr(unsafe.Pointer(a.pointer)), 0, key, uintptr(p), size, 0, 0)
	if r == 0 {
		return e
	}
	return nil
}

func (a *windowsScopeAttributeList) list() *windows.ProcThreadAttributeList { return a.pointer }

func (a *windowsScopeAttributeList) close() error {
	windowsScopeDeleteAttributes.Call(uintptr(unsafe.Pointer(a.pointer)))
	runtime.KeepAlive(a.values)
	a.pins.Unpin()
	a.buffer, a.pointer, a.values = nil, nil, nil
	return nil
}
