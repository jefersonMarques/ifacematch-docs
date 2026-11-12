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
		"/",
		"/api",
		"/upload-image",
		"/sdk",
		"/installation",
		"/releases",
		"/upgrade",
	}

	for _, pagePath := range paths {
		t.Run(pagePath, func(t *testing.T) {
			response := request(t, handler, http.MethodGet, pagePath, nil)
			if response.Code != http.StatusOK {
				t.Fatalf("GET %s: got %d", pagePath, response.Code)
			}

			body := response.Body.String()
			for _, expected := range []string{
				"<html lang=\"pt-BR\">",
				"<main class=\"document\" id=\"content\">",
				"<nav class=\"sidebar\"",
				"https://docs.ifacematch.com" + pagePath,
				"/assets/htmx.min.js",
			} {
				if !strings.Contains(body, expected) {
					t.Errorf("GET %s missing %q", pagePath, expected)
				}
			}

			if !strings.Contains(body, `rel="canonical"`) {
				t.Errorf("GET %s missing canonical tag", pagePath)
			}
		})
	}
}

func TestLegacyPageRedirects(t *testing.T) {
	handler := NewHandler()
	redirects := map[string]string{
		"/docs.html":          "/",
		"/docs-api.html":      "/api",
		"/docs-sdk.html":      "/sdk",
		"/docs-install.html":  "/installation",
		"/releases.html":      "/releases",
		"/upgrade-guide.html": "/upgrade",
	}

	for oldPath, canonicalPath := range redirects {
		t.Run(oldPath, func(t *testing.T) {
			for _, method := range []string{http.MethodGet, http.MethodHead} {
				response := request(t, handler, method, oldPath, nil)
				if response.Code != http.StatusMovedPermanently {
					t.Errorf("%s %s returned %d instead of 301", method, oldPath, response.Code)
				}
				if location := response.Header().Get("Location"); location != canonicalPath {
					t.Errorf("%s %s redirected to %q instead of %q", method, oldPath, location, canonicalPath)
				}
			}

			response := request(t, handler, http.MethodGet, oldPath+"?source=legacy", nil)
			if location := response.Header().Get("Location"); location != canonicalPath+"?source=legacy" {
				t.Errorf("query not preserved: %s -> %s", oldPath, location)
			}
		})
	}
}

func TestCanonicalTrailingSlashRedirects(t *testing.T) {
	handler := NewHandler()
	for _, pagePath := range []string{"/api", "/upload-image", "/sdk", "/installation", "/releases", "/upgrade"} {
		response := request(t, handler, http.MethodGet, pagePath+"/", nil)
		if response.Code != http.StatusMovedPermanently {
			t.Errorf("%s/ returned %d instead of 301", pagePath, response.Code)
		}
		if location := response.Header().Get("Location"); location != pagePath {
			t.Errorf("%s/ redirected to %q", pagePath, location)
		}
	}
}

func TestDocumentationLinksUseCanonicalPaths(t *testing.T) {
	handler := NewHandler()
	legacyPaths := []string{
		"/docs.html", "/docs-api.html", "/docs-sdk.html",
		"/docs-install.html", "/releases.html", "/upgrade-guide.html",
	}

	for _, pagePath := range []string{"/", "/api", "/upload-image", "/sdk", "/installation", "/releases", "/upgrade"} {
		response := request(t, handler, http.MethodGet, pagePath, nil)
		body := response.Body.String()
		for _, legacyPath := range legacyPaths {
			if strings.Contains(body, `href="`+legacyPath) {
				t.Errorf("%s links to legacy URL %s", pagePath, legacyPath)
			}
		}
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
	if !strings.Contains(response.Body.String(), `href="/sdk"`) {
		t.Fatal("search result must point to the canonical SDK page")
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
	for _, assetPath := range []string{
		"/openapi.json",
		"/openapi-preview.json",
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
		response := request(t, handler, http.MethodGet, assetPath, nil)
		if response.Code != http.StatusOK || response.Body.Len() == 0 {
			t.Errorf("%s returned status=%d length=%d", assetPath, response.Code, response.Body.Len())
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

func TestFaceUploadPreviewNotAdvertisedAsStable(t *testing.T) {
    handler := NewHandler()

    previewPage := request(t, handler, http.MethodGet, "/upload-image", nil)
    if previewPage.Code != http.StatusOK {
        t.Fatalf("preview page status: %d", previewPage.Code)
    }
    for _, expected := range []string{
        "/v1/enrollments/images",
        "multipart/form-data",
        "0.1.11",
        "não",
        "/openapi-preview.json",
    } {
        if !strings.Contains(previewPage.Body.String(), expected) {
            t.Errorf("upload preview page missing %q", expected)
        }
    }

    stable := request(t, handler, http.MethodGet, "/openapi.json", nil)
    var stableContract struct {
        Paths map[string]json.RawMessage `json:"paths"`
    }
    if err := json.Unmarshal(stable.Body.Bytes(), &stableContract); err != nil {
        t.Fatal(err)
    }
    if _, exists := stableContract.Paths["/v1/enrollments/images"]; exists {
        t.Fatal("unpublished upload endpoint must not appear in stable OpenAPI")
    }

    preview := request(t, handler, http.MethodGet, "/openapi-preview.json", nil)
    var draftContract struct {
        OpenAPI string `json:"openapi"`
        Paths map[string]json.RawMessage `json:"paths"`
    }
    if err := json.Unmarshal(preview.Body.Bytes(), &draftContract); err != nil {
        t.Fatal(err)
    }
    if draftContract.OpenAPI != "3.1.0" || len(draftContract.Paths) != 3 {
        t.Fatalf("invalid preview OpenAPI: version=%s paths=%d", draftContract.OpenAPI, len(draftContract.Paths))
    }
    for _, endpoint := range []string{
        "/v1/enrollments/images",
        "/v1/subjects/{subjectId}/enrollment",
        "/v1/subjects/{subjectId}/enrollment/image",
    } {
        if _, exists := draftContract.Paths[endpoint]; !exists {
            t.Errorf("preview missing endpoint: %s", endpoint)
        }
    }
}

func TestSitemapUsesCanonicalPaths(t *testing.T) {
	response := request(t, NewHandler(), http.MethodGet, "/sitemap.xml", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("sitemap status: %d", response.Code)
	}

	sitemap := response.Body.String()
	for _, pagePath := range []string{"/", "/api", "/upload-image", "/sdk", "/installation", "/releases", "/upgrade"} {
		expected := "<loc>https://docs.ifacematch.com" + pagePath + "</loc>"
		if !strings.Contains(sitemap, expected) {
			t.Errorf("sitemap missing %q", expected)
		}
	}
	if strings.Contains(sitemap, ".html") {
		t.Fatal("sitemap contains old page URLs")
	}
}

func TestPrivateRoutesAreUnavailable(t *testing.T) {
	handler := NewHandler()
	for _, pagePath := range []string{
		"/api/admin/v1/login",
		"/api/portal/v1/login",
		"/.env",
		"/admin",
		"/internal/content/catalog.go",
	} {
		response := request(t, handler, http.MethodGet, pagePath, nil)
		if response.Code != http.StatusNotFound {
			t.Errorf("%s: expected 404, got %d", pagePath, response.Code)
		}
	}

	response := request(t, handler, http.MethodPost, "/api", nil)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected POST status: %d", response.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	response := request(t, NewHandler(), http.MethodGet, "/", nil)
	for _, header := range []string{
		"X-Content-Type-Options",
		"Content-Security-Policy",
		"X-Frame-Options",
		"Permissions-Policy",
	} {
		if response.Header().Get(header) == "" {
			t.Errorf("missing security header %s", header)
		}
	}
}

func TestSearchQueryLimit(t *testing.T) {
	response := request(t, NewHandler(), http.MethodGet, "/search?q="+strings.Repeat("a", 121), nil)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}

func TestExampleHTMLIsSourceOnly(t *testing.T) {
	response := request(t, NewHandler(), http.MethodGet, "/example.html", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	if got := response.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", got)
	}
	_, _ = io.Copy(io.Discard, response.Body)
}
