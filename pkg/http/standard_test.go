package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kaizenforyou91/forge/pkg/middleware"
	"github.com/kaizenforyou91/forge/pkg/router"
)

func TestNewModuleWithStandardMiddlewarePipeline(t *testing.T) {
	var (
		gotRequestID string
		gotStatus    int
		gotPath      string
		gotMethod    string
	)

	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
		gotMethod = method
		gotPath = path
		gotRequestID = requestID
		gotStatus = status
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if middleware.RequestIDFromContext(r.Context()) == "" {
			t.Fatal("expected request ID in handler context")
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("forge"))
	})

	module := NewModule(
		"127.0.0.1:0",
		handler,
		middleware.Standard(logger),
	)

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	rec := httptest.NewRecorder()

	module.Host().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			rec.Code,
		)
	}

	if got := rec.Body.String(); got != "forge" {
		t.Fatalf(
			"expected response body %q, got %q",
			"forge",
			got,
		)
	}

	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID response header")
	}

	if gotMethod != http.MethodPost {
		t.Fatalf(
			"expected logged method %q, got %q",
			http.MethodPost,
			gotMethod,
		)
	}

	if gotPath != "/api/test" {
		t.Fatalf(
			"expected logged path %q, got %q",
			"/api/test",
			gotPath,
		)
	}

	if gotRequestID == "" {
		t.Fatal("expected access log to capture request ID")
	}

	if gotStatus != http.StatusCreated {
		t.Fatalf(
			"expected logged status %d, got %d",
			http.StatusCreated,
			gotStatus,
		)
	}
}

func TestNewModuleWithStandardMiddlewarePreservesExistingRequestID(t *testing.T) {
	const requestID = "forge-http-standard-test-id"

	var loggedRequestID string

	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
		loggedRequestID = requestID
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := middleware.RequestIDFromContext(r.Context())

		if got != requestID {
			t.Fatalf(
				"expected request ID %q in context, got %q",
				requestID,
				got,
			)
		}

		w.WriteHeader(http.StatusOK)
	})

	module := NewModule(
		"127.0.0.1:0",
		handler,
		middleware.Standard(logger),
	)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", requestID)

	rec := httptest.NewRecorder()

	module.Host().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	if got := rec.Header().Get("X-Request-ID"); got != requestID {
		t.Fatalf(
			"expected response request ID %q, got %q",
			requestID,
			got,
		)
	}

	if loggedRequestID != requestID {
		t.Fatalf(
			"expected access log request ID %q, got %q",
			requestID,
			loggedRequestID,
		)
	}
}

func TestNewModuleWithStandardMiddlewareRecoversPanic(t *testing.T) {
	var (
		loggedStatus    int
		loggedRequestID string
	)

	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
		loggedStatus = status
		loggedRequestID = requestID
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("forge integration test panic")
	})

	module := NewModule(
		"127.0.0.1:0",
		handler,
		middleware.Standard(logger),
	)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	module.Host().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}

	if loggedStatus != http.StatusInternalServerError {
		t.Fatalf(
			"expected access log status %d, got %d",
			http.StatusInternalServerError,
			loggedStatus,
		)
	}

	if loggedRequestID == "" {
		t.Fatal("expected recovered request to retain request ID")
	}

	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID response header after recovery")
	}
}

func TestNewModuleWithoutMiddlewarePreservesHandlerBehavior(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("plain handler"))
	})

	module := NewModule(
		"127.0.0.1:0",
		handler,
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	module.Host().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusAccepted,
			rec.Code,
		)
	}

	if got := rec.Body.String(); got != "plain handler" {
		t.Fatalf(
			"expected body %q, got %q",
			"plain handler",
			got,
		)
	}
}

func TestNewModuleAppliesMiddlewareInDeclaredOrder(t *testing.T) {
	var order []string

	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m1-before")

			next.ServeHTTP(w, r)

			order = append(order, "m1-after")
		})
	}

	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m2-before")

			next.ServeHTTP(w, r)

			order = append(order, "m2-after")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	module := NewModule(
		"127.0.0.1:0",
		handler,
		m1,
		m2,
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	module.Host().Handler().ServeHTTP(rec, req)

	expected := []string{
		"m1-before",
		"m2-before",
		"handler",
		"m2-after",
		"m1-after",
	}

	if len(order) != len(expected) {
		t.Fatalf(
			"unexpected middleware execution count: got %d want %d; order=%#v",
			len(order),
			len(expected),
			order,
		)
	}

	for i := range expected {
		if order[i] != expected[i] {
			t.Fatalf(
				"unexpected middleware order at index %d: got %q want %q; full order=%#v",
				i,
				order[i],
				expected[i],
				order,
			)
		}
	}
}

func TestNewStandardModuleUsesStandardPipeline(t *testing.T) {
	var (
		loggedStatus    int
		loggedRequestID string
	)

	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
		loggedStatus = status
		loggedRequestID = requestID
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if middleware.RequestIDFromContext(r.Context()) == "" {
			t.Fatal("expected request ID in handler context")
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("standard"))
	})

	module := NewStandardModule(
		"127.0.0.1:0",
		handler,
		logger,
	)

	if module == nil {
		t.Fatal("expected standard HTTP module")
	}

	req := httptest.NewRequest(http.MethodGet, "/standard", nil)
	rec := httptest.NewRecorder()

	module.Host().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			rec.Code,
		)
	}

	if got := rec.Body.String(); got != "standard" {
		t.Fatalf(
			"expected response body %q, got %q",
			"standard",
			got,
		)
	}

	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID response header")
	}

	if loggedStatus != http.StatusCreated {
		t.Fatalf(
			"expected access log status %d, got %d",
			http.StatusCreated,
			loggedStatus,
		)
	}

	if loggedRequestID == "" {
		t.Fatal("expected access log request ID")
	}
}

func TestNewStandardModuleRecoversPanicAndLogs500(t *testing.T) {
	var (
		loggedStatus    int
		loggedRequestID string
	)

	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
		loggedStatus = status
		loggedRequestID = requestID
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("forge standard module panic")
	})

	module := NewStandardModule(
		"127.0.0.1:0",
		handler,
		logger,
	)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	module.Host().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}

	if loggedStatus != http.StatusInternalServerError {
		t.Fatalf(
			"expected access log status %d, got %d",
			http.StatusInternalServerError,
			loggedStatus,
		)
	}

	if loggedRequestID == "" {
		t.Fatal("expected access log request ID")
	}

	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID response header")
	}
}

func TestNewStandardRouterModule(t *testing.T) {
	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
	}

	r := router.New()

	r.GET(
		"/health",
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}),
	)

	module := NewStandardRouterModule(
		"127.0.0.1:0",
		r,
		logger,
	)

	if module == nil {
		t.Fatal("expected standard router module")
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	module.Host().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	if got := rec.Body.String(); got != "ok" {
		t.Fatalf(
			"expected response body %q, got %q",
			"ok",
			got,
		)
	}

	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID response header")
	}
}

func TestNewStandardModuleAppliesAdditionalMiddleware(t *testing.T) {
	var order []string

	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
		order = append(order, "access-log")
	}

	additional := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "additional-before")

			next.ServeHTTP(w, r)

			order = append(order, "additional-after")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	module := NewStandardModule(
		"127.0.0.1:0",
		handler,
		logger,
		additional,
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	module.Host().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	expected := []string{
		"additional-before",
		"handler",
		"additional-after",
		"access-log",
	}

	if len(order) != len(expected) {
		t.Fatalf(
			"unexpected execution count: got %d want %d; order=%#v",
			len(order),
			len(expected),
			order,
		)
	}

	for i := range expected {
		if order[i] != expected[i] {
			t.Fatalf(
				"unexpected execution order at index %d: got %q want %q; full order=%#v",
				i,
				order[i],
				expected[i],
				order,
			)
		}
	}
}
