package http

import (
	"context"
	"net/http"
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
	host.server.Addr = "127.0.0.1:0"

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
