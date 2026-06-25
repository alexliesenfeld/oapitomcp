package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	goapitomcp "github.com/alexliesenfeld/openapimcp"
)

func main() {
	addr := envString("ADDR", ":8082")
	baseURL, err := url.Parse("http://localhost" + addr + "/api")
	if err != nil {
		log.Fatal(err)
	}

	mcpHandler, err := goapitomcp.NewHandlerFromFile(context.Background(), examplePath("openapi.yaml"), goapitomcp.Config{
		BaseURL: baseURL,
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/pets/", handlePet)
	mux.Handle("/mcp", mcpHandler)

	log.Printf("multi-file API: http://localhost%s/api/pets/pet-123", addr)
	log.Printf("MCP endpoint: http://localhost%s/mcp", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func handlePet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/pets/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":   id,
		"name": "Milo",
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func examplePath(name string) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("cannot locate example directory")
	}
	return filepath.Join(filepath.Dir(file), name)
}

func envString(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
