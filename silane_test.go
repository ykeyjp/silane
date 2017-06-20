package silane_test

import (
	"fmt"
	"github.com/ykeyjp/silane"
	"github.com/ykeyjp/silane/middleware"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeMux(t *testing.T) {
	s := silane.New()
	s.Get("/u1/u2/:id", func(c *silane.Context) {
		c.Response.Json(struct {
			Id string `json:"id"`
		}{
			Id: c.Params["id"],
		})
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/u1/u2/10", nil)
	s.ServeHTTP(response, request)
	if response.Body.String() != "{\"id\":\"10\"}" {
		t.Fatalf("response body is not matched. %s", response.Body.String())
	}
}

func TestMiddleware(t *testing.T) {
	s := silane.New()
	s.Use(func(c *silane.Context, next silane.NextFunc) {
		c.Response.Header.Set("X-Test", "test")
		next(c)
	})
	s.Get("/u1/u2/:id", func(c *silane.Context) {
		c.Response.Json(struct {
			Id string `json:"id"`
		}{
			Id: c.Params["id"],
		})
	}).With(func(c *silane.Context, next silane.NextFunc) {
		next(c)
		c.Response.Header.Set("X-Test2", "test2")
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/u1/u2/10", nil)
	s.ServeHTTP(response, request)
	if response.Body.String() != "{\"id\":\"10\"}" {
		t.Fatalf("response body is not matched. %s", response.Body.String())
	}
	if response.Header().Get("X-Test") != "test" {
		t.Fatalf("response header is not matched. %s", response.Header().Get("X-Test"))
	}
	if response.Header().Get("X-Test2") != "test2" {
		t.Fatalf("response header is not matched. %s", response.Header().Get("X-Test2"))
	}
}

func TestGroupRouting(t *testing.T) {
	s := silane.New()
	s.Group("/u1", func(m *silane.Map) {
		m.Get("u2/:id", func(c *silane.Context) {
			c.Response.Json(struct {
				Id string `json:"id"`
			}{
				Id: c.Params["id"],
			})
		})
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/u1/u2/10", nil)
	s.ServeHTTP(response, request)
	if response.Body.String() != "{\"id\":\"10\"}" {
		t.Fatalf("response body is not matched. %s", response.Body.String())
	}
}

func TestBuiltinMiddleware(t *testing.T) {
	s := silane.New()
	s.Use(middleware.JsonStrategy)
	s.Get("/u1/u2/:id", func(c *silane.Context) {
		c.Response.Json(struct {
			Id string `json:"id"`
		}{
			Id: c.Params["id"],
		})
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/u1/u2/10", nil)
	s.ServeHTTP(response, request)
	if response.Body.String() != "{\"id\":\"10\"}" {
		t.Fatalf("response body is not matched. %s", response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("response header is not matched. %s", response.Header().Get("Content-Type"))
	}
}

func BenchmarkAddRoute(b *testing.B) {
	s := silane.New()
	h := func(c *silane.Context) {}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uri := fmt.Sprintf("/u1/u%d/u%d", i, i)
		s.Get(uri, h)
	}
}

func BenchmarkAddRouteBuiltin(b *testing.B) {
	s := http.NewServeMux()
	h := func(w http.ResponseWriter, r *http.Request) {}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uri := fmt.Sprintf("/u1/u%d/u%d", i, i)
		s.HandleFunc(uri, h)
	}
}

func BenchmarkRouting(b *testing.B) {
	s := silane.New()
	h := func(c *silane.Context) {}
	for i := 0; i < 1000; i++ {
		uri := fmt.Sprintf("/u1/u%d/u%d", i, i)
		s.Get(uri, h)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uri := fmt.Sprintf("/u1/u%d/u%d", i, i)
		s.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", uri, nil))
	}
}

func BenchmarkRoutingBuiltin(b *testing.B) {
	s := http.NewServeMux()
	h := func(w http.ResponseWriter, r *http.Request) {}
	for i := 0; i < 1000; i++ {
		uri := fmt.Sprintf("/u1/u%d/u%d", i, i)
		s.HandleFunc(uri, h)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uri := fmt.Sprintf("/u1/u%d/u%d", i, i)
		s.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", uri, nil))
	}
}
