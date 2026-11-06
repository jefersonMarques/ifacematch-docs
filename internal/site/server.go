package site

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/a-h/templ"
	"github.com/jefersonMarques/ifacematch-docs/internal/content"
	"github.com/jefersonMarques/ifacematch-docs/internal/views"
)

var navigation = []views.Group{
	{
		Title: "Começar",
		Links: []views.Link{
			{Label: "Visão geral", URL: "/"},
			{Label: "Início rápido", URL: "/#quickstart"},
			{Label: "Instalação", URL: "/installation"},
			{Label: "Autenticação", URL: "/#credentials"},
		},
	},
	{
		Title: "Integração",
		Links: []views.Link{
			{Label: "API REST", URL: "/api"},
			{Label: "Prova de vida", URL: "/api#liveness"},
			{Label: "Cadastro facial", URL: "/api#enrollment"},
			{Label: "Verificação 1:1", URL: "/api#verification"},
			{Label: "SDK Web", URL: "/sdk"},
		},
	},
	{
		Title: "Operação",
		Links: []views.Link{
			{Label: "Releases", URL: "/releases"},
			{Label: "Upgrade e rollback", URL: "/upgrade"},
			{Label: "Especificação OpenAPI", URL: "/openapi.json"},
			{Label: "Documentação para IA", URL: "/llms.txt"},
		},
	},
}

var legacyPagePaths = map[string]string{
	"/docs.html": "/",
	"/docs-api.html": "/api",
	"/docs-sdk.html": "/sdk",
	"/docs-install.html": "/installation",
	"/releases.html": "/releases",
	"/upgrade-guide.html": "/upgrade",
}

var publicAssets = []string{
	"/openapi.json",
	"/releases.json",
	"/docs.md",
	"/llms.txt",
	"/llms-full.txt",
	"/robots.txt",
	"/sitemap.xml",
	"/example.html",
	"/example.js",
	"/example-backend.go",
}

func NewHandler() http.Handler {
	staticFiles, err := content.StaticFiles()
	if err != nil {
		panic(fmt.Errorf("initialize embedded assets: %w", err))
	}

	staticServer := http.FileServer(http.FS(staticFiles))
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer.Header().Set("Cache-Control", "no-store")
		_, _ = writer.Write([]byte("ok\n"))
	})

	for _, metadata := range content.Pages() {
		pagePath := metadata.Path
		routePattern := "GET " + pagePath
		if pagePath == "/" {
			routePattern = "GET /{$}"
		}

		mux.HandleFunc(routePattern, func(writer http.ResponseWriter, request *http.Request) {
			page, found, err := content.Lookup(pagePath)
			if err != nil {
				log.Printf("documentation page unavailable: %v", err)
				http.Error(writer, "documentation unavailable", http.StatusInternalServerError)
				return
			}
			if !found {
				http.NotFound(writer, request)
				return
			}

			render(writer, request, views.DocumentPage(toViewPage(page), navigation))
		})
	}

	for oldPath, canonicalPath := range legacyPagePaths {
		mux.HandleFunc("GET "+oldPath, redirectToCanonical(canonicalPath))
	}

	for _, metadata := range content.Pages() {
		if metadata.Path == "/" {
			continue
		}

		mux.HandleFunc("GET "+metadata.Path+"/{$}", redirectToCanonical(metadata.Path))
	}

	mux.HandleFunc("GET /search", handleSearch)
	mux.Handle("GET /assets/", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/assets/" || strings.HasSuffix(request.URL.Path, "/") {
			http.NotFound(writer, request)
			return
		}

		writer.Header().Set("Cache-Control", "public, max-age=600")
		staticServer.ServeHTTP(writer, request)
	}))

	for _, assetPath := range publicAssets {
		asset := assetPath
		mux.Handle("GET "+asset, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			// Arquivos de demonstração são apresentados como fonte, nunca executados no domínio de documentação.
			if asset == "/example.html" {
				writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
				servePlainText(writer, request, staticFiles, strings.TrimPrefix(asset, "/"))
				return
			}
			writer.Header().Set("Cache-Control", "public, max-age=300")
			staticServer.ServeHTTP(writer, request)
		}))
	}

	return securityHeaders(mux)
}

func redirectToCanonical(target string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		destination := target
		if request.URL.RawQuery != "" {
			destination += "?" + request.URL.RawQuery
		}
		http.Redirect(writer, request, destination, http.StatusMovedPermanently)
	}
}

func servePlainText(writer http.ResponseWriter, request *http.Request, files fs.FS, name string) {
	raw, err := fs.ReadFile(files, name)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	_, _ = writer.Write(raw)
}

func handleSearch(writer http.ResponseWriter, request *http.Request) {
	query := strings.TrimSpace(request.URL.Query().Get("q"))
	if utf8.RuneCountInString(query) > 120 {
		http.Error(writer, "query is too long", http.StatusBadRequest)
		return
	}

	found := content.Search(query)
	results := make([]views.SearchResult, 0, len(found))
	for _, item := range found {
		results = append(results, views.SearchResult{
			Path:        item.Path,
			Title:       item.Title,
			Description: item.Description,
		})
	}

	writer.Header().Set("Cache-Control", "no-store")
	if request.Header.Get("HX-Request") == "true" {
		render(writer, request, views.SearchResults(query, results))
		return
	}

	page := views.Page{
		Path:        "/search",
		Title:       "Buscar documentação",
		Description: "Pesquise tópicos da documentação do iFaceMatch.",
		SearchQuery: query,
	}
	render(writer, request, views.SearchPage(page, navigation, results))
}

func render(writer http.ResponseWriter, request *http.Request, component templ.Component) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	if err := component.Render(request.Context(), writer); err != nil {
		log.Printf("documentation render failed: %v", err)
	}
}

func toViewPage(page content.Page) views.Page {
	return views.Page{
		Path:        page.Path,
		Title:       page.Title,
		Description: page.Description,
		Body:        page.Body,
		Contents:    page.Contents,
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		writer.Header().Set("Content-Security-Policy", strings.Join([]string{
			"default-src 'none'",
			"script-src 'self'",
			"style-src 'self'",
			"img-src 'self' data:",
			"connect-src 'self'",
			"font-src 'self'",
			"form-action 'self'",
			"base-uri 'none'",
			"frame-ancestors 'none'",
		}, "; "))
		writer.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
		next.ServeHTTP(writer, request)
	})
}

