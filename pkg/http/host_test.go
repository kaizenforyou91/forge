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
