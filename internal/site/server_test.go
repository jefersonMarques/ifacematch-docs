package site

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func request(t *testing.T, handler http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func TestDocumentationPages(t *testing.T) {
	handler := NewHandler()
	paths := []string{
		"/docs.html",
		"/docs-api.html",
		"/docs-sdk.html",
		"/docs-install.html",
		"/releases.html",
		"/upgrade-guide.html",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			res := request(t, handler, http.MethodGet, path, nil)
			if res.Code != http.StatusOK {
				t.Fatalf("GET %s: got %d", path, res.Code)
			}

			body := res.Body.String()
			for _, expected := range []string{
				"<html lang=\"pt-BR\">",
				"<main class=\"document\" id=\"content\">",
				"<nav class=\"sidebar\"",
				"https://docs.ifacematch.com" + path,
				"/assets/htmx.min.js",
			} {
				if !strings.Contains(body, expected) {
					t.Errorf("GET %s missing %q", path, expected)
				}
			}
		})
	}
}

func TestHomeRedirect(t *testing.T) {
	res := request(t, NewHandler(), http.MethodGet, "/", nil)
	if res.Code != http.StatusPermanentRedirect {
		t.Fatalf("unexpected status: %d", res.Code)
	}
	if res.Header().Get("Location") != "/docs.html" {
		t.Fatalf("unexpected location: %q", res.Header().Get("Location"))
	}
}

func TestSearchHTMXAndEscaping(t *testing.T) {
	handler := NewHandler()
	response := request(t, handler, http.MethodGet, "/search?q=SDK", map[string]string{
		"HX-Request": "true",
	})
	if response.Code != http.StatusOK {
		t.Fatalf("search status: %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "SDK Web") {
		t.Fatal("expected SDK Web search result")
	}
	if strings.Contains(response.Body.String(), "<html") {
		t.Fatal("HTMX must return an HTML fragment")
	}

	payload := "/search?q=%3Cscript%3Ealert(1)%3C%2Fscript%3E"
	searchPage := request(t, handler, http.MethodGet, payload, nil)
	if searchPage.Code != http.StatusOK {
		t.Fatalf("search status: %d", searchPage.Code)
	}
	if strings.Contains(searchPage.Body.String(), "<script>alert(1)</script>") {
		t.Fatal("unescaped search input")
	}
	if !strings.Contains(searchPage.Body.String(), "noindex,follow") {
		t.Fatal("search pages must not be indexed")
	}
}

func TestStaticOpenAPIAndAIAssets(t *testing.T) {
	handler := NewHandler()
	for _, path := range []string{
		"/openapi.json",
		"/releases.json",
		"/llms.txt",
		"/llms-full.txt",
		"/docs.md",
		"/robots.txt",
		"/sitemap.xml",
		"/assets/docs.css",
		"/assets/htmx.min.js",
		"/assets/app.js",
	} {
		res := request(t, handler, http.MethodGet, path, nil)
		if res.Code != http.StatusOK || res.Body.Len() == 0 {
			t.Errorf("%s returned status=%d length=%d", path, res.Code, res.Body.Len())
		}
	}

	response := request(t, handler, http.MethodGet, "/openapi.json", nil)
	var contract struct {
		OpenAPI string                     `json:"openapi"`
		Paths   map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &contract); err != nil {
		t.Fatal(err)
	}
	if contract.OpenAPI != "3.1.0" || len(contract.Paths) != 6 {
		t.Fatalf("invalid public contract: version=%s paths=%d", contract.OpenAPI, len(contract.Paths))
	}
}

func TestPrivateRoutesAreUnavailable(t *testing.T) {
	handler := NewHandler()
	for _, path := range []string{
		"/api/admin/v1/login",
		"/api/portal/v1/login",
		"/.env",
		"/admin",
		"/internal/content/catalog.go",
	} {
		res := request(t, handler, http.MethodGet, path, nil)
		if res.Code != http.StatusNotFound {
			t.Errorf("%s: expected 404, got %d", path, res.Code)
		}
	}

	res := request(t, handler, http.MethodPost, "/docs.html", nil)
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected POST status: %d", res.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	res := request(t, NewHandler(), http.MethodGet, "/docs.html", nil)
	for _, key := range []string{
		"X-Content-Type-Options",
		"Content-Security-Policy",
		"X-Frame-Options",
		"Permissions-Policy",
	} {
		if res.Header().Get(key) == "" {
			t.Errorf("missing security header %s", key)
		}
	}
}

func TestSearchQueryLimit(t *testing.T) {
	res := request(t, NewHandler(), http.MethodGet, "/search?q="+strings.Repeat("a", 121), nil)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", res.Code)
	}
}

func TestExampleHTMLIsSourceOnly(t *testing.T) {
	res := request(t, NewHandler(), http.MethodGet, "/example.html", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", res.Code)
	}
	if got := res.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", got)
	}
	_, _ = io.Copy(io.Discard, res.Body)
}
