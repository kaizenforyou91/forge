package http

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	handler := http.NewServeMux()

	host := New(":8080", handler)

	if host == nil {
		t.Fatal("expected host")
	}

	if host.Addr() != ":8080" {
		t.Fatalf("unexpected address: %s", host.Addr())
	}

	if host.Handler() != handler {
		t.Fatal("unexpected handler")
	}
}

func TestNewNilHandler(t *testing.T) {
	host := New(":8080", nil)

	if host == nil {
		t.Fatal("expected host")
	}

	if host.Handler() != http.DefaultServeMux {
		t.Fatal("expected default mux")
	}
}

func TestListenerAddr(t *testing.T) {
	host := New("127.0.0.1:0", http.NewServeMux())

	if host.ListenerAddr() != nil {
		t.Fatal("listener should not exist before start")
	}

	done := make(chan error, 1)

	go func() {
		done <- host.Start()
	}()

	deadline := time.After(time.Second)

	for {
		if host.ListenerAddr() != nil {
			break
		}

		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}

		case <-deadline:
			t.Fatal("listener was not created")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	if err := host.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestStartStop(t *testing.T) {
	host := New("127.0.0.1:0", http.NewServeMux())

	done := make(chan error, 1)

	go func() {
		done <- host.Start()
	}()

	deadline := time.After(time.Second)

	for host.ListenerAddr() == nil {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}

		case <-deadline:
			t.Fatal("listener was not created")

		default:
			time.Sleep(time.Millisecond)
		}
	}

	if host.Listener() == nil {
		t.Fatal("expected active listener")
	}

	if err := host.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}

	case <-time.After(time.Second):
		t.Fatal("server did not stop")
	}
}

func TestHostStartTwiceReturnsError(t *testing.T) {
	host := New("127.0.0.1:0", http.NewServeMux())

	done := make(chan error, 1)

	go func() {
		done <- host.Start()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}

	case <-host.Ready():
	}

	if host.ListenerAddr() == nil {
		t.Fatal("expected active listener")
	}

	secondDone := make(chan error, 1)

	go func() {
		secondDone <- host.Start()
	}()

	select {
	case err := <-secondDone:
		if err == nil {
			t.Fatal("expected second Start to return an error")
		}

	case <-time.After(time.Second):
		t.Fatal("second Start did not return")
	}

	if err := host.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected first Start error: %v", err)
		}

	case <-time.After(time.Second):
		t.Fatal("server did not stop")
	}
}

func TestHostStopBeforeStartIsSafe(t *testing.T) {
	host := New("127.0.0.1:0", http.NewServeMux())

	if err := host.Stop(context.Background()); err != nil {
		t.Fatalf("expected Stop before Start to be safe, got %v", err)
	}

	if host.Listener() != nil {
		t.Fatal("expected no active listener")
	}

	if host.ListenerAddr() != nil {
		t.Fatal("expected no listener address")
	}
}

func TestHostReadyIsNotClosedBeforeStart(t *testing.T) {
	host := New("127.0.0.1:0", http.NewServeMux())

	select {
	case <-host.Ready():
		t.Fatal("host should not be ready before Start")

	default:
	}
}

func TestHostStartListenFailureResetsState(t *testing.T) {
	blocker := New("127.0.0.1:0", http.NewServeMux())

	blockerDone := make(chan error, 1)

	go func() {
		blockerDone <- blocker.Start()
	}()

	select {
	case <-blocker.Ready():
	case err := <-blockerDone:
		if err != nil {
			t.Fatal(err)
		}

		t.Fatal("blocker stopped before becoming ready")

	case <-time.After(time.Second):
		t.Fatal("blocker did not become ready")
	}

	addr := blocker.ListenerAddr()
	if addr == nil {
		t.Fatal("expected blocker listener address")
	}

	host := New(addr.String(), http.NewServeMux())

	if err := host.Start(); err == nil {
		t.Fatal("expected Start to fail while address is occupied")
	}

	// The failed Start must reset the internal started flag.
	// Change the server address to a free ephemeral port and try again.
	host.addr = "127.0.0.1:0"

	done := make(chan error, 1)

	go func() {
		done <- host.Start()
	}()

	select {
	case <-host.Ready():
	case err := <-done:
		if err != nil {
			t.Fatalf("host could not restart after listen failure: %v", err)
		}

		t.Fatal("host stopped before becoming ready")

	case <-time.After(time.Second):
		t.Fatal("host did not restart after listen failure")
	}

	if host.ListenerAddr() == nil {
		t.Fatal("expected listener after successful restart")
	}

	if err := host.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected restarted host error: %v", err)
		}

	case <-time.After(time.Second):
		t.Fatal("restarted host did not stop")
	}

	if err := blocker.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-blockerDone:
		if err != nil {
			t.Fatalf("unexpected blocker error: %v", err)
		}

	case <-time.After(time.Second):
		t.Fatal("blocker did not stop")
	}
}

func TestHostRestartAfterStop(t *testing.T) {
	host := New("127.0.0.1:0", http.NewServeMux())

	firstDone := make(chan error, 1)
	go func() {
		firstDone <- host.Start()
	}()

	select {
	case <-host.Ready():
	case err := <-firstDone:
		if err != nil {
			t.Fatal(err)
		}

		t.Fatal("first Start returned early")
	case <-time.After(time.Second):
		t.Fatal("first Start did not become ready")
	}

	if host.Listener() == nil || host.ListenerAddr() == nil {
		t.Fatal("expected active listener after first Start")
	}

	if err := host.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("first Start did not stop")
	}

	if host.Listener() != nil || host.ListenerAddr() != nil {
		t.Fatal("expected listener state cleared after Stop")
	}

	secondDone := make(chan error, 1)
	go func() {
		secondDone <- host.Start()
	}()

	select {
	case <-host.Ready():
	case err := <-secondDone:
		if err != nil {
			t.Fatal(err)
		}

		t.Fatal("second Start returned early")
	case <-time.After(time.Second):
		t.Fatal("second Start did not become ready")
	}

	if host.Listener() == nil || host.ListenerAddr() == nil {
		t.Fatal("expected active listener after second Start")
	}

	if err := host.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("second Start did not stop")
	}
}

func TestHostRepeatedStopIsSafe(t *testing.T) {
	host := New("127.0.0.1:0", http.NewServeMux())

	if err := host.Stop(context.Background()); err != nil {
		t.Fatalf("first Stop before start should be safe: %v", err)
	}

	if err := host.Stop(context.Background()); err != nil {
		t.Fatalf("second Stop before start should be safe: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- host.Start()
	}()

	select {
	case <-host.Ready():
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
		t.Fatal("Start returned early")
	case <-time.After(time.Second):
		t.Fatal("Start did not become ready")
	}

	if err := host.Stop(context.Background()); err != nil {
		t.Fatalf("Stop after start should be safe: %v", err)
	}

	if err := host.Stop(context.Background()); err != nil {
		t.Fatalf("second Stop after stop should be safe: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Start did not stop")
	}
}

func TestHostConcurrentStart(t *testing.T) {
	host := New("127.0.0.1:0", http.NewServeMux())

	errCh := make(chan error, 5)
	for i := 0; i < 5; i++ {
		go func() {
			errCh <- host.Start()
		}()
	}

	select {
	case <-host.Ready():
	case <-time.After(time.Second):
		t.Fatal("host did not become ready")
	}

	var failures int
	for i := 0; i < 4; i++ {
		select {
		case err := <-errCh:
			if err == nil {
				t.Fatal("expected failed Start for concurrent attempt")
			}
			failures++
		case <-time.After(time.Second):
			t.Fatal("concurrent Start did not return")
		}
	}

	if failures != 4 {
		t.Fatalf("expected 4 failed Starts, got %d", failures)
	}

	if err := host.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected active Start to stop cleanly, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("active Start did not return")
	}
}

func TestHostConcurrentStop(t *testing.T) {
	host := New("127.0.0.1:0", http.NewServeMux())

	go func() {
		_ = host.Start()
	}()

	select {
	case <-host.Ready():
	case <-time.After(time.Second):
		t.Fatal("host did not become ready")
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 10)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- host.Stop(context.Background())
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("expected Stop calls to be safe, got %v", err)
		}
	}
}

func TestHostGracefulShutdownWithActiveRequest(t *testing.T) {
	requestStarted := make(chan struct{})
	allowDone := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		close(requestStarted)
		<-allowDone
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "done")
	})

	host := New("127.0.0.1:0", handler)

	done := make(chan error, 1)
	go func() {
		done <- host.Start()
	}()

	select {
	case <-host.Ready():
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
		t.Fatal("host stopped before ready")
	case <-time.After(time.Second):
		t.Fatal("host did not become ready")
	}

	url := "http://" + host.ListenerAddr().String() + "/"

	clientDone := make(chan error, 1)
	go func() {
		resp, err := http.Get(url)
		if err != nil {
			clientDone <- err
			return
		}
		defer resp.Body.Close()
		_, err = io.ReadAll(resp.Body)
		clientDone <- err
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("request handler did not start")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		if err := host.Stop(shutdownCtx); err != nil {
			clientDone <- err
		}
	}()

	close(allowDone)

	select {
	case err := <-clientDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("client request did not complete")
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop")
	}
}
