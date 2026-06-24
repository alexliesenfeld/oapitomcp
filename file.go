package goapitomcp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// LoadCatalogFromFile opens an OpenAPI document from disk and defaults
// SpecBaseURI to that file's file:// URI so local relative refs can resolve.
func LoadCatalogFromFile(ctx context.Context, filename string, cfg Config) (*Catalog, error) {
	cfg, file, err := configWithSpecFile(filename, cfg)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return LoadCatalog(ctx, cfg)
}

// NewServerFromFile opens an OpenAPI document from disk and creates an MCP
// server from it.
func NewServerFromFile(ctx context.Context, filename string, cfg Config) (*mcp.Server, error) {
	cfg, file, err := configWithSpecFile(filename, cfg)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return NewServer(ctx, cfg)
}

// NewHandlerFromFile opens an OpenAPI document from disk and returns a standard
// Streamable HTTP MCP handler for it.
func NewHandlerFromFile(ctx context.Context, filename string, cfg Config) (http.Handler, error) {
	cfg, file, err := configWithSpecFile(filename, cfg)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return NewHandler(ctx, cfg)
}

func configWithSpecFile(filename string, cfg Config) (Config, *os.File, error) {
	file, err := os.Open(filename)
	if err != nil {
		return cfg, nil, fmt.Errorf("open OpenAPI spec file: %w", err)
	}
	cfg.Spec = file
	if cfg.SpecBaseURI == nil {
		specBaseURI, err := fileURI(filename)
		if err != nil {
			_ = file.Close()
			return cfg, nil, err
		}
		cfg.SpecBaseURI = specBaseURI
	}
	return cfg, file, nil
}

func fileURI(filename string) (*url.URL, error) {
	absolute, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("resolve OpenAPI spec path: %w", err)
	}
	return &url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(absolute),
	}, nil
}
