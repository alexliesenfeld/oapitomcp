package goapitomcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExamplePetstoreDocumentLoads(t *testing.T) {
	path := filepath.Join("examples", "petstore", "openapi.yaml")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	catalog, err := LoadCatalog(context.Background(), Config{
		Spec:        file,
		SpecBaseURI: fileURL(t, path),
	})
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	if len(catalog.Operations) != 2 {
		t.Fatalf("operations = %d, want 2", len(catalog.Operations))
	}
	if len(catalog.SecuritySchemes) != 1 || catalog.SecuritySchemes[0].ID != "ApiKeyAuth" {
		t.Fatalf("SecuritySchemes = %#v", catalog.SecuritySchemes)
	}
	op := findOperation(t, catalog, "GET", "/pets")
	if !strings.Contains(op.ResponseDocumentation, "body.pets") {
		t.Fatalf("ResponseDocumentation = %q, want pet fields", op.ResponseDocumentation)
	}
}

func TestExampleMultiFileDocumentLoads(t *testing.T) {
	path := filepath.Join("examples", "multifile", "openapi.yaml")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	catalog, err := LoadCatalog(context.Background(), Config{
		Spec:        file,
		SpecBaseURI: fileURL(t, path),
	})
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	op := findOperation(t, catalog, "GET", "/pets/{petId}")
	if len(op.Parameters) != 1 || op.Parameters[0].Name != "petId" {
		t.Fatalf("Parameters = %#v, want resolved petId", op.Parameters)
	}
	if !strings.Contains(op.ResponseDocumentation, "body.id") {
		t.Fatalf("ResponseDocumentation = %q, want resolved schema fields", op.ResponseDocumentation)
	}
}

func TestExampleFilteringDocumentLoadsWithPublicFilter(t *testing.T) {
	path := filepath.Join("examples", "filtering", "openapi.yaml")
	catalog, err := LoadCatalogFromFile(context.Background(), path, Config{
		Filter: &OperationFilter{
			ExcludePathPatterns: []string{"/admin/*"},
			ExcludeTags:         []string{"debug", "internal"},
			ExcludeMethods:      []string{"DELETE"},
		},
	})
	if err != nil {
		t.Fatalf("LoadCatalogFromFile() error = %v", err)
	}
	if len(catalog.Operations) != 2 {
		t.Fatalf("operations = %d, want 2", len(catalog.Operations))
	}
	assertHasOperation(t, catalog, "GET", "/users")
	assertHasOperation(t, catalog, "POST", "/users")
	assertNoOperation(t, catalog, "DELETE", "/admin/users")
	assertNoOperation(t, catalog, "GET", "/debug/status")
}
