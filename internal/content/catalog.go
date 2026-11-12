package content

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed pages toc public
var embeddedFiles embed.FS

type Page struct {
	Path        string
	SourceFile  string
	Title       string
	Description string
	Body        string
	Contents    string
}

type SearchResult struct {
	Path        string
	Title       string
	Description string
}

var catalog = []Page{
	{
		Path:        "/",
		SourceFile:  "docs.html",
		Title:       "Integração com iFaceMatch",
		Description: "Integre biometria facial 1:1 e prova de vida via API REST e SDK Web, com exemplos de código e segurança de credenciais.",
	},
	{
		Path:        "/api",
		SourceFile:  "docs-api.html",
		Title:       "Referência da API REST",
		Description: "Contratos de prova de vida, cadastro facial e verificação 1:1 com exemplos de requisições e resultados.",
	},
	{
		Path:        "/upload-image",
		SourceFile:  "upload-image.html",
		Title:       "Cadastro facial por imagem — prévia",
		Description: "Contrato em desenvolvimento: upload JPEG/PNG de referência facial com subjectId, autenticação e consulta de foto; ainda indisponível em produção.",
	},
	{
		Path:        "/sdk",
		SourceFile:  "docs-sdk.html",
		Title:       "SDK Web e captura facial",
		Description: "Integração do SDK Web iFaceMatch, captura segura no navegador e processamento no servidor do cliente.",
	},
	{
		Path:        "/installation",
		SourceFile:  "docs-install.html",
		Title:       "Instalação Customer Hosted",
		Description: "Pré-requisitos, instalação, diagnóstico e recuperação segura do iFaceMatch Customer Hosted.",
	},
	{
		Path:        "/releases",
		SourceFile:  "releases.html",
		Title:       "Releases e histórico de versões",
		Description: "Histórico de versões e compatibilidade dos componentes Server, Console, Secure Core e SDK Web.",
	},
	{
		Path:        "/upgrade",
		SourceFile:  "upgrade-guide.html",
		Title:       "Guia de upgrade e rollback",
		Description: "Como verificar releases assinadas, preservar backups, aplicar upgrades e validar rollback.",
	},
}

func Pages() []Page {
	pages := make([]Page, len(catalog))
	copy(pages, catalog)
	return pages
}

func Lookup(path string) (Page, bool, error) {
	for _, entry := range catalog {
		if entry.Path != path {
			continue
		}

		filename := entry.SourceFile
		body, err := embeddedFiles.ReadFile("pages/" + filename)
		if err != nil {
			return Page{}, true, fmt.Errorf("load page %s: %w", filename, err)
		}

		contents, err := embeddedFiles.ReadFile("toc/" + filename)
		if err != nil {
			return Page{}, true, fmt.Errorf("load contents %s: %w", filename, err)
		}

		entry.Body = string(body)
		entry.Contents = string(contents)
		return entry, true, nil
	}

	return Page{}, false, nil
}

func StaticFiles() (fs.FS, error) {
	return fs.Sub(embeddedFiles, "public")
}

func Search(query string) []SearchResult {
	normalized := strings.ToLower(strings.TrimSpace(query))
	if normalized == "" {
		return nil
	}

	results := make([]SearchResult, 0, len(catalog))

	for _, entry := range catalog {
		searchable := strings.ToLower(entry.Title + " " + entry.Description)
		if !strings.Contains(searchable, normalized) {
			filename := entry.SourceFile
			body, err := embeddedFiles.ReadFile("pages/" + filename)
			if err != nil || !strings.Contains(strings.ToLower(string(body)), normalized) {
				continue
			}
		}

		results = append(results, SearchResult{
			Path:        entry.Path,
			Title:       entry.Title,
			Description: entry.Description,
		})
	}

	return results
}
