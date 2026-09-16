//go:build windows

package runtime

import (
	"errors"
	"os"
	"reflect"
	"sync"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsScopeFakeAttributes struct {
	updates map[uintptr][]windows.Handle
	failKey uintptr
	closed  int
}

func (a *windowsScopeFakeAttributes) update(key uintptr, p unsafe.Pointer, size uintptr) error {
	a.updates[key] = append([]windows.Handle(nil), unsafe.Slice((*windows.Handle)(p), int(size/unsafe.Sizeof(windows.Handle(0))))...)
	if key == a.failKey {
		return windows.ERROR_NOT_SUPPORTED
	}
	return nil
}
func (a *windowsScopeFakeAttributes) list() *windows.ProcThreadAttributeList { return nil }
func (a *windowsScopeFakeAttributes) close() error                           { a.closed++; return nil }

func windowsScopeTestSpec() *windowsProcessScopeLaunchSpec {
	return &windowsProcessScopeLaunchSpec{executable: `C:\fixture.exe`, commandLine: `fixture.exe mode`, directory: `C:\`, environment: []string{"Z=last", "A=first"}, stdio: [3]windows.Handle{20, 21, 22}}
}

type windowsScopeFake struct {
	attrs                                          windowsScopeFakeAttributes
	closed                                         map[windows.Handle]int
	created, configured, starts, kills, duplicates int
	fail                                           string
	killErr                                        error
	closeErr                                       error
	t                                              *testing.T
}

func (f *windowsScopeFake) native() windowsScopeNative {
	f.attrs.updates = make(map[uintptr][]windows.Handle)
	f.closed = make(map[windows.Handle]int)
	return windowsScopeNative{
		newJob: func() (windows.Handle, error) {
			f.created++
			if f.fail == "job" {
				return 0, windows.ERROR_ACCESS_DENIED
			}
			return 10, nil
		},
		configure: func(h windows.Handle) error {
			f.configured++
			if h != 10 {
				f.t.Fatal("wrong job")
			}
			if f.fail == "config" {
				return windows.ERROR_ACCESS_DENIED
			}
			return nil
		},
		duplicate: func(h windows.Handle) (windows.Handle, error) {
			f.duplicates++
			if f.fail == "duplicate" && f.duplicates == 2 {
				return 0, windows.ERROR_INVALID_HANDLE
			}
			return h + 100, nil
		},
		attributes: func() (windowsScopeAttributes, error) {
			if f.fail == "attributes" {
				return nil, windows.ERROR_NOT_ENOUGH_MEMORY
			}
			return &f.attrs, nil
		},
		create: func(app, cmd *uint16, inherit bool, flags uint32, env, dir *uint16, si *windows.StartupInfo, pi *windows.ProcessInformation) error {
			f.starts++
			if f.configured != 1 || !inherit || flags != windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_UNICODE_ENVIRONMENT {
				f.t.Fatal("creation contract")
			}
			if si.Cb != uint32(unsafe.Sizeof(windows.StartupInfoEx{})) || si.Flags != windows.STARTF_USESTDHANDLES || si.StdInput != 120 || si.StdOutput != 121 || si.StdErr != 122 {
				f.t.Fatal("stdio/startup contract")
			}
			if !reflect.DeepEqual(f.attrs.updates[windowsScopeJobList], []windows.Handle{10}) || !reflect.DeepEqual(f.attrs.updates[windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST], []windows.Handle{120, 121, 122}) {
				f.t.Fatal("attribute authority")
			}
			if f.fail == "create" {
				return windows.ERROR_ACCESS_DENIED
			}
			pi.Process, pi.Thread = 30, 31
			return nil
		},
		terminate: func(h windows.Handle, code uint32) error {
			f.kills++
			if h != 10 || code != 1 {
				f.t.Fatal("wrong termination authority")
			}
			return f.killErr
		},
		close: func(h windows.Handle) error {
			f.closed[h]++
			if (h == 31 && f.fail == "thread-close") || (h == 10 && f.closeErr != nil) {
				if f.closeErr != nil {
					return f.closeErr
				}
				return windows.ERROR_INVALID_HANDLE
			}
			return nil
		},
	}
}

func TestWindowsScopeLaunchValidation(t *testing.T) {
	for _, mutate := range []func(*windowsProcessScopeLaunchSpec){
		func(s *windowsProcessScopeLaunchSpec) { s.executable = "relative.exe" },
		func(s *windowsProcessScopeLaunchSpec) { s.directory = "relative" },
		func(s *windowsProcessScopeLaunchSpec) { s.commandLine = "" },
		func(s *windowsProcessScopeLaunchSpec) { s.commandLine = "bad\x00line" },
		func(s *windowsProcessScopeLaunchSpec) { s.environment = nil },
		func(s *windowsProcessScopeLaunchSpec) { s.environment = []string{"A=1", "a=2"} },
		func(s *windowsProcessScopeLaunchSpec) { s.environment = []string{"invalid"} },
		func(s *windowsProcessScopeLaunchSpec) { s.stdio[1] = 0 },
		func(s *windowsProcessScopeLaunchSpec) { s.stdio[2] = windows.InvalidHandle },
	} {
		s := windowsScopeTestSpec()
		mutate(s)
		if _, err := startWindowsProcessInScopeWith(s, windowsScopeNative{}); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	if _, err := startWindowsProcessInScope(nil); err == nil {
		t.Fatal("nil accepted")
	}
	_, _, _, env, err := windowsScopeStrings(windowsScopeTestSpec())
	if err != nil || !reflect.DeepEqual(env, []uint16{'A', '=', 'f', 'i', 'r', 's', 't', 0, 'Z', '=', 'l', 'a', 's', 't', 0, 0}) {
		t.Fatal("explicit sorted environment", err)
	}
	s := windowsScopeTestSpec()
	s.environment = []string{}
	_, _, _, env, err = windowsScopeStrings(s)
	if err != nil || !reflect.DeepEqual(env, []uint16{0, 0}) {
		t.Fatal("empty environment")
	}
}

func TestWindowsScopeLaunchOwnership(t *testing.T) {
	f := &windowsScopeFake{t: t}
	n := f.native()
	l, err := startWindowsProcessInScopeWith(windowsScopeTestSpec(), n)
	if err != nil {
		t.Fatal(err)
	}
	if l.process != 30 || l.scope.job != 10 || f.attrs.closed != 1 || f.kills != 0 || !reflect.DeepEqual(f.closed, map[windows.Handle]int{120: 1, 121: 1, 122: 1, 31: 1}) {
		t.Fatalf("ownership: %+v", f)
	}
	if err = l.scope.terminate(); err != nil {
		t.Fatal(err)
	}
	if f.kills != 1 {
		t.Fatal("terminate count")
	}
	if err = l.scope.finalize(); err != nil {
		t.Fatal(err)
	}
	if err = l.scope.finalize(); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(l.scope.terminate(), os.ErrProcessDone) || f.kills != 1 || f.closed[10] != 1 || f.closed[30] != 0 {
		t.Fatal("retired scope touched child/control")
	}
	if l.scope.job != 0 || l.scope.killJob != nil || l.scope.closeJob != nil {
		t.Fatal("references retained")
	}
}

func TestWindowsScopePartialStarts(t *testing.T) {
	for _, stage := range []string{"job", "config", "duplicate", "attributes", "job-attribute", "handle-attribute", "create", "thread-close"} {
		t.Run(stage, func(t *testing.T) {
			f := &windowsScopeFake{t: t, fail: stage}
			n := f.native()
			if stage == "job-attribute" {
				f.attrs.failKey = windowsScopeJobList
			}
			if stage == "handle-attribute" {
				f.attrs.failKey = windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST
			}
			l, err := startWindowsProcessInScopeWith(windowsScopeTestSpec(), n)
			if err == nil || l != nil {
				t.Fatal("failed start returned ownership")
			}
			wantJob := 1
			if stage == "job" {
				wantJob = 0
			}
			if f.closed[10] != wantJob {
				t.Fatalf("job close count %d", f.closed[10])
			}
			for h, c := range f.closed {
				if c != 1 {
					t.Fatalf("handle %d close count %d", h, c)
				}
			}
			wantAttrs := 0
			if stage == "job-attribute" || stage == "handle-attribute" || stage == "create" || stage == "thread-close" {
				wantAttrs = 1
			}
			if f.attrs.closed != wantAttrs {
				t.Fatal("attribute cleanup count")
			}
			if stage == "thread-close" {
				if f.kills != 1 || f.closed[30] != 1 || f.closed[31] != 1 {
					t.Fatal("post-create cleanup")
				}
			} else if f.kills != 0 {
				t.Fatal("unnecessary termination")
			}
			for _, h := range []windows.Handle{20, 21, 22} {
				if f.closed[h] != 0 {
					t.Fatal("borrowed stdio closed")
				}
			}
		})
	}
}

func TestWindowsScopeFailureAndConcurrentFinalization(t *testing.T) {
	f := &windowsScopeFake{t: t, killErr: windows.ERROR_ACCESS_DENIED, closeErr: windows.ERROR_INVALID_HANDLE}
	n := f.native()
	s := &windowsProcessScopePlatform{job: 10, killJob: n.terminate, closeJob: n.close}
	if !errors.Is(s.terminate(), windows.ERROR_ACCESS_DENIED) {
		t.Fatal("control failure lost")
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !errors.Is(s.finalize(), windows.ERROR_INVALID_HANDLE) {
				t.Error("close failure lost")
			}
		}()
	}
	wg.Wait()
	if f.closed[10] != 1 || f.kills != 1 {
		t.Fatal("call counts")
	}
	if !errors.Is(s.terminate(), os.ErrProcessDone) {
		t.Fatal("retired control")
	}
}
