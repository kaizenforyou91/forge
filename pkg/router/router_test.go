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

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}

	if got := rec.Header().Get("Allow"); got != "GET" {
		t.Fatalf("expected Allow header %q, got %q", "GET", got)
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
func TestPathParameter(t *testing.T) {
	r := New()

	r.GET("/users/:id", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if got := Param(req, "id"); got != "123" {
			t.Fatalf("expected id 123, got %q", got)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestMultiplePathParameters(t *testing.T) {
	r := New()

	r.GET("/users/:userID/posts/:postID", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if got := Param(req, "userID"); got != "10" {
			t.Fatalf("expected userID 10, got %q", got)
		}

		if got := Param(req, "postID"); got != "20" {
			t.Fatalf("expected postID 20, got %q", got)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/users/10/posts/20", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestStaticRouteHasPriorityOverParameterRoute(t *testing.T) {
	r := New()

	r.GET("/users/:id", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		t.Fatal("parameter route should not be selected")
	}))

	r.GET("/users/me", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestMissingPathParameter(t *testing.T) {
	r := New()

	r.GET("/users/:id", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if got := Param(req, "unknown"); got != "" {
			t.Fatalf("expected empty parameter, got %q", got)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestParameterRoutePathMismatch(t *testing.T) {
	r := New()

	r.GET("/users/:id", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		t.Fatal("handler should not execute")
	}))

	req := httptest.NewRequest(http.MethodGet, "/users/123/profile", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestRouterMethodRouting(t *testing.T) {
	r := New()

	r.GET("/users", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r.POST("/users", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	tests := []struct {
		method string
		want   int
	}{
		{http.MethodGet, http.StatusOK},
		{http.MethodPost, http.StatusCreated},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, "/users", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != tt.want {
			t.Fatalf("%s /users: expected %d, got %d", tt.method, tt.want, rec.Code)
		}
	}
}

func TestRouterMethodNotAllowed(t *testing.T) {
	r := New()

	r.GET("/users", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r.POST("/users", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodDelete, "/users", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}

	if got := rec.Header().Get("Allow"); got != "GET, POST" {
		t.Fatalf("expected Allow header %q, got %q", "GET, POST", got)
	}
}

func TestRouterNotFound(t *testing.T) {
	r := New()

	r.GET("/users", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	if got := rec.Header().Get("Allow"); got != "" {
		t.Fatalf("expected no Allow header, got %q", got)
	}
}

func TestRouterMethodNotAllowedWithParameterRoute(t *testing.T) {
	r := New()

	r.GET("/users/:id", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/users/123", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}

	if got := rec.Header().Get("Allow"); got != "GET" {
		t.Fatalf("expected Allow header %q, got %q", "GET", got)
	}
}
