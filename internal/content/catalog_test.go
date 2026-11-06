package content

import (
	"strings"
	"testing"
)

func TestEmbeddedPages(t *testing.T) {
	pages := Pages()
	if len(pages) != 6 {
		t.Fatalf("expected six documentation pages, got %d", len(pages))
	}

	for _, entry := range pages {
		document, found, err := Lookup(entry.Path)
		if err != nil || !found {
			t.Fatalf("page missing %s: found=%v err=%v", entry.Path, found, err)
		}
		if !strings.Contains(document.Body, "<h1>") || !strings.Contains(document.Contents, "href=") {
			t.Errorf("invalid HTML fragment: %s", entry.Path)
		}
	}
}

func TestSearch(t *testing.T) {
	results := Search("verification")
	if len(results) == 0 {
		t.Fatal("expected a verification topic")
	}
	if got := Search(" "); len(got) != 0 {
		t.Errorf("empty query must return nothing, got %d", len(got))
	}
}

func TestStaticAssetsAvailable(t *testing.T) {
	files, err := StaticFiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"assets/docs.css",
		"assets/htmx.min.js",
		"openapi.json",
		"llms.txt",
	} {
		file, err := files.Open(name)
		if err != nil {
			t.Errorf("missing %s: %v", name, err)
			continue
		}
		_ = file.Close()
	}
}
