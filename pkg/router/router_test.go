package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouter(t *testing.T) {
	r := New()

	if r == nil {
		t.Fatal("expected router")
	}

	if len(r.Routes()) != 0 {
		t.Fatal("expected empty routes")
	}
}

func TestRegisterRoute(t *testing.T) {
	r := New()

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	r.Handle(http.MethodPost, "/users", handler)

	routes := r.Routes()

	if len(routes) != 1 {
		t.Fatalf("expected one route, got %d", len(routes))
	}

	if routes[0].Method != http.MethodPost {
		t.Fatalf("unexpected method: %s", routes[0].Method)
	}

	if routes[0].Path != "/users" {
		t.Fatalf("unexpected path: %s", routes[0].Path)
	}
}

func TestExactPathMatch(t *testing.T) {
	r := New()

	r.GET("/users", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestMethodMismatch(t *testing.T) {
	r := New()

	r.GET("/users", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/users", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestPathMismatch(t *testing.T) {
	r := New()

	r.GET("/users", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestHTTPMethodHelpers(t *testing.T) {
	r := New()

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {})

	r.GET("/", handler)
	r.POST("/users", handler)
	r.PUT("/users/1", handler)
	r.DELETE("/users/1", handler)

	if len(r.Routes()) != 4 {
		t.Fatalf("expected four routes, got %d", len(r.Routes()))
	}

	expected := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/"},
		{http.MethodPost, "/users"},
		{http.MethodPut, "/users/1"},
		{http.MethodDelete, "/users/1"},
	}

	for i, want := range expected {
		got := r.Routes()[i]

		if got.Method != want.method {
			t.Fatalf("route %d: expected method %s, got %s", i, want.method, got.Method)
		}

		if got.Path != want.path {
			t.Fatalf("route %d: expected path %s, got %s", i, want.path, got.Path)
		}
	}
}

func TestHandler(t *testing.T) {
	r := New()

	handler := r.Handler()

	if handler == nil {
		t.Fatal("expected http handler")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestRoutesReturnsCopy(t *testing.T) {
	r := New()

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {})

	r.GET("/", handler)

	routes := r.Routes()
	routes[0].Path = "/modified"

	if r.Routes()[0].Path != "/" {
		t.Fatal("Routes should return a copy")
	}
}
